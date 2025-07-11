package controllers

import (
	"encoding/json"
	"fmt"
	"go_binance_futures/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/expr-lang/expr"
)

type BacktestController struct {
	web.Controller
}

// BacktestRequest 回测请求结构
type BacktestRequest struct {
	Name       string                 `json:"name"`       // 任务名称
	Strategy   string                 `json:"strategy"`   // 策略表达式
	Parameters []map[string]interface{} `json:"parameters"` // 参数组合
	Symbol     string                 `json:"symbol"`     // 测试币种
	StartTime  int64                  `json:"start_time"` // 回测开始时间
	EndTime    int64                  `json:"end_time"`   // 回测结束时间
	Concurrent int                    `json:"concurrent"` // 并发数量，默认5
}

// BacktestResponse 回测响应结构
type BacktestResponse struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// BatchBacktestResponse 批量回测响应结构
type BatchBacktestResponse struct {
	TaskIDs []string `json:"task_ids"`
	Status  string   `json:"status"`
	Message string   `json:"message"`
}

// BacktestResultResponse 回测结果响应结构
type BacktestResultResponse struct {
	TaskID     string                 `json:"task_id"`
	Results    []models.BacktestResult `json:"results"`
	BestResult *models.BacktestResult  `json:"best_result"`
	Status     string                 `json:"status"`
}

// @Title 创建回测任务
// @Description 创建新的回测任务，支持自定义策略表达式、参数组、回测区间、币种等
// @Param body body BacktestRequest true "回测请求参数"
// @Success 200 {object} BacktestResponse
// @router /api/backtest [post]
func (this *BacktestController) Post() {
	var req BacktestRequest
	if err := json.Unmarshal(this.Ctx.Input.RequestBody, &req); err != nil {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Invalid request format: " + err.Error(),
		}
		this.ServeJSON()
		return
	}

	// 验证请求参数
	if req.Name == "" {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Task name is required",
		}
		this.ServeJSON()
		return
	}

	if req.Strategy == "" {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Strategy expression is required",
		}
		this.ServeJSON()
		return
	}

	if req.Symbol == "" {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Symbol is required",
		}
		this.ServeJSON()
		return
	}

	if len(req.Parameters) == 0 {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Parameters are required",
		}
		this.ServeJSON()
		return
	}

	// 验证策略表达式
	if err := this.validateStrategy(req.Strategy); err != nil {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Invalid strategy expression: " + err.Error(),
		}
		this.ServeJSON()
		return
	}

	// 生成任务ID
	taskID := this.generateTaskID()
	
	// 记录操作日志
	this.logOperation("CREATE_BACKTEST", "backtest_task", taskID, req)

	// 创建回测任务
	o := orm.NewOrm()
	task := models.BacktestTask{
		TaskID:     taskID,
		Name:       req.Name,
		Status:     "pending",
		Strategy:   req.Strategy,
		Symbol:     req.Symbol,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		CreatedBy:  this.getCurrentUser(),
		Progress:   0,
		CreateTime: time.Now().Unix(),
		UpdateTime: time.Now().Unix(),
	}

	// 序列化参数
	paramBytes, _ := json.Marshal(req.Parameters)
	task.Parameters = string(paramBytes)

	if _, err := o.Insert(&task); err != nil {
		logs.Error("Failed to create backtest task:", err)
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Failed to create backtest task",
		}
		this.ServeJSON()
		return
	}

	// 异步执行回测
	go this.executeBacktest(taskID, req)

	this.Data["json"] = BacktestResponse{
		TaskID:  taskID,
		Status:  "success",
		Message: "Backtest task created successfully",
	}
	this.ServeJSON()
}

// @Title 获取回测任务列表
// @Description 获取回测任务列表，支持分页和过滤
// @Param page query int false "页码，默认1"
// @Param pageSize query int false "每页数量，默认20"
// @Param status query string false "状态过滤"
// @Success 200 {object} object
// @router /api/backtest [get]
func (this *BacktestController) Get() {
	page, _ := this.GetInt("page", 1)
	pageSize, _ := this.GetInt("pageSize", 20)
	status := this.GetString("status")

	offset := (page - 1) * pageSize
	o := orm.NewOrm()
	
	var tasks []models.BacktestTask
	qs := o.QueryTable("backtest_tasks")
	
	if status != "" {
		qs = qs.Filter("status", status)
	}
	
	// 获取总数
	total, _ := qs.Count()
	
	// 获取数据
	qs.OrderBy("-create_time").Limit(pageSize, offset).All(&tasks)

	this.Data["json"] = map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"tasks": tasks,
			"total": total,
			"page":  page,
			"pageSize": pageSize,
		},
	}
	this.ServeJSON()
}

// @Title 获取回测结果
// @Description 获取指定任务的回测结果
// @Param taskId path string true "任务ID"
// @Success 200 {object} BacktestResultResponse
// @router /api/backtest/:taskId/results [get]
func (this *BacktestController) GetResults() {
	taskID := this.Ctx.Input.Param(":taskId")
	
	o := orm.NewOrm()
	
	// 获取任务状态
	var task models.BacktestTask
	if err := o.QueryTable("backtest_tasks").Filter("task_id", taskID).One(&task); err != nil {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Task not found",
		}
		this.ServeJSON()
		return
	}

	// 获取回测结果
	var results []models.BacktestResult
	o.QueryTable("backtest_results").Filter("task_id", taskID).OrderBy("-sharpe_ratio").All(&results)

	var bestResult *models.BacktestResult
	if len(results) > 0 {
		bestResult = &results[0]
	}

	this.Data["json"] = BacktestResultResponse{
		TaskID:     taskID,
		Results:    results,
		BestResult: bestResult,
		Status:     task.Status,
	}
	this.ServeJSON()
}

// @Title 删除回测任务
// @Description 删除指定的回测任务及其结果
// @Param taskId path string true "任务ID"
// @Success 200 {object} object
// @router /api/backtest/:taskId [delete]
func (this *BacktestController) Delete() {
	taskID := this.Ctx.Input.Param(":taskId")
	
	o := orm.NewOrm()
	
	// 记录操作日志
	this.logOperation("DELETE_BACKTEST", "backtest_task", taskID, nil)

	// 删除任务
	if _, err := o.QueryTable("backtest_tasks").Filter("task_id", taskID).Delete(); err != nil {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Failed to delete task",
		}
		this.ServeJSON()
		return
	}

	// 删除相关结果
	o.QueryTable("backtest_results").Filter("task_id", taskID).Delete()

	this.Data["json"] = map[string]interface{}{
		"status":  "success",
		"message": "Task deleted successfully",
	}
	this.ServeJSON()
}

// 验证策略表达式
func (this *BacktestController) validateStrategy(strategy string) error {
	// 创建一个简单的环境来验证表达式
	env := map[string]interface{}{
		"close": 100.0,
		"open":  99.0,
		"high":  101.0,
		"low":   98.0,
		"ma5":   100.5,
		"ma20":  99.8,
		"rsi":   60.0,
	}

	_, err := expr.Eval(strategy, env)
	return err
}

// 生成任务ID
func (this *BacktestController) generateTaskID() string {
	return fmt.Sprintf("bt_%d_%d", time.Now().Unix(), time.Now().UnixNano()%1000000)
}

// 获取当前用户
func (this *BacktestController) getCurrentUser() string {
	// 从JWT token或session中获取用户信息
	// 这里先返回默认值，后续可以根据实际认证系统调整
	return "admin"
}

// 记录操作日志
func (this *BacktestController) logOperation(operation, resourceType, resourceID string, details interface{}) {
	o := orm.NewOrm()
	
	detailsBytes, _ := json.Marshal(details)
	
	log := models.OperationLog{
		UserID:       this.getCurrentUser(),
		Operation:    operation,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      string(detailsBytes),
		IPAddress:    this.Ctx.Input.IP(),
		UserAgent:    this.Ctx.Input.UserAgent(),
		CreateTime:   time.Now().Unix(),
	}
	
	o.Insert(&log)
}

// 执行回测
func (this *BacktestController) executeBacktest(taskID string, req BacktestRequest) {
	o := orm.NewOrm()
	
	// 更新任务状态
	o.QueryTable("backtest_tasks").Filter("task_id", taskID).Update(orm.Params{
		"status":      "running",
		"update_time": time.Now().Unix(),
	})

	defer func() {
		if r := recover(); r != nil {
			logs.Error("Backtest execution panic:", r)
			o.QueryTable("backtest_tasks").Filter("task_id", taskID).Update(orm.Params{
				"status":        "failed",
				"error_message": fmt.Sprintf("Execution failed: %v", r),
				"update_time":   time.Now().Unix(),
			})
		}
	}()

	// 设置并发数
	concurrent := req.Concurrent
	if concurrent <= 0 {
		concurrent = 5
	}
	if concurrent > 20 {
		concurrent = 20 // 限制最大并发数
	}

	// 创建并发控制
	semaphore := make(chan struct{}, concurrent)
	resultChan := make(chan *models.BacktestResult, len(req.Parameters))
	
	// 并发执行回测
	for i, params := range req.Parameters {
		go func(index int, parameters map[string]interface{}) {
			semaphore <- struct{}{} // 获取信号量
			defer func() { <-semaphore }() // 释放信号量
			
			result := this.runSingleBacktest(taskID, req, parameters, index)
			resultChan <- result
			
			// 更新进度
			progress := float64(index+1) / float64(len(req.Parameters)) * 100
			o.QueryTable("backtest_tasks").Filter("task_id", taskID).Update(orm.Params{
				"progress":    progress,
				"update_time": time.Now().Unix(),
			})
		}(i, params)
	}

	// 收集结果
	for i := 0; i < len(req.Parameters); i++ {
		result := <-resultChan
		if result != nil {
			o.Insert(result)
		}
	}

	// 更新任务状态为完成
	o.QueryTable("backtest_tasks").Filter("task_id", taskID).Update(orm.Params{
		"status":      "completed",
		"progress":    100,
		"update_time": time.Now().Unix(),
	})
}

// 执行单个回测
func (this *BacktestController) runSingleBacktest(taskID string, req BacktestRequest, parameters map[string]interface{}, index int) *models.BacktestResult {
	// 生成结果ID
	resultID := fmt.Sprintf("%s_%d", taskID, index)
	
	// 这里是简化的回测逻辑
	// 实际应用中需要根据具体的策略和数据源来实现
	result := &models.BacktestResult{
		TaskID:     taskID,
		ResultID:   resultID,
		Parameters: this.serializeParameters(parameters),
		CreateTime: time.Now().Unix(),
	}

	// 模拟回测计算
	// 这里需要根据实际的回测引擎来实现
	result.TotalReturn = this.calculateTotalReturn(req, parameters)
	result.AnnualReturn = this.calculateAnnualReturn(result.TotalReturn, req.StartTime, req.EndTime)
	result.MaxDrawdown = this.calculateMaxDrawdown(req, parameters)
	result.SharpeRatio = this.calculateSharpeRatio(result.TotalReturn, result.MaxDrawdown)
	result.WinRate = this.calculateWinRate(req, parameters)
	result.TradeCount = this.calculateTradeCount(req, parameters)
	result.ProfitFactor = this.calculateProfitFactor(req, parameters)

	// 生成模拟的权益曲线和交易记录
	result.EquityCurve = this.generateEquityCurve(req, parameters)
	result.TradeList = this.generateTradeList(req, parameters)

	return result
}

// 序列化参数
func (this *BacktestController) serializeParameters(params map[string]interface{}) string {
	bytes, _ := json.Marshal(params)
	return string(bytes)
}

// 以下是简化的回测计算函数，实际应用中需要根据具体需求实现
func (this *BacktestController) calculateTotalReturn(req BacktestRequest, params map[string]interface{}) float64 {
	// 简化计算，实际应根据策略和历史数据计算
	return 0.15 + (float64(len(params))*0.01) // 示例：15%基础收益 + 参数复杂度调整
}

func (this *BacktestController) calculateAnnualReturn(totalReturn float64, startTime, endTime int64) float64 {
	days := float64(endTime-startTime) / 86400
	years := days / 365.25
	if years <= 0 {
		return 0
	}
	return totalReturn / years
}

func (this *BacktestController) calculateMaxDrawdown(req BacktestRequest, params map[string]interface{}) float64 {
	// 简化计算
	return 0.05 // 示例：5%最大回撤
}

func (this *BacktestController) calculateSharpeRatio(totalReturn, maxDrawdown float64) float64 {
	if maxDrawdown == 0 {
		return 0
	}
	return totalReturn / maxDrawdown
}

func (this *BacktestController) calculateWinRate(req BacktestRequest, params map[string]interface{}) float64 {
	// 简化计算
	return 0.65 // 示例：65%胜率
}

func (this *BacktestController) calculateTradeCount(req BacktestRequest, params map[string]interface{}) int64 {
	// 简化计算
	return 100 // 示例：100笔交易
}

func (this *BacktestController) calculateProfitFactor(req BacktestRequest, params map[string]interface{}) float64 {
	// 简化计算
	return 1.5 // 示例：1.5利润因子
}

func (this *BacktestController) generateEquityCurve(req BacktestRequest, params map[string]interface{}) string {
	// 生成简化的权益曲线数据
	curve := make([]map[string]interface{}, 100)
	for i := 0; i < 100; i++ {
		curve[i] = map[string]interface{}{
			"time":   req.StartTime + int64(i)*86400,
			"equity": 10000 + float64(i)*100,
		}
	}
	bytes, _ := json.Marshal(curve)
	return string(bytes)
}

func (this *BacktestController) generateTradeList(req BacktestRequest, params map[string]interface{}) string {
	// 生成简化的交易记录
	trades := make([]map[string]interface{}, 10)
	for i := 0; i < 10; i++ {
		trades[i] = map[string]interface{}{
			"time":   req.StartTime + int64(i)*8640,
			"symbol": req.Symbol,
			"side":   "buy",
			"price":  100.0 + float64(i),
			"qty":    1.0,
			"pnl":    50.0,
		}
	}
	bytes, _ := json.Marshal(trades)
	return string(bytes)
}