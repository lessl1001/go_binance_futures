package controllers

import (
	"encoding/json"
	"fmt"
	"go_binance_futures/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
)

type DeployStrategyController struct {
	web.Controller
}

// DeployStrategyRequest 部署策略请求结构
type DeployStrategyRequest struct {
	Name             string                 `json:"name"`               // 策略名称
	Symbol           string                 `json:"symbol"`             // 应用币种
	Parameters       map[string]interface{} `json:"parameters"`         // 最优参数
	Strategy         string                 `json:"strategy"`           // 策略表达式
	BacktestTaskID   string                 `json:"backtest_task_id"`   // 关联的回测任务ID
	BacktestResultID string                 `json:"backtest_result_id"` // 关联的回测结果ID
	Force            bool                   `json:"force"`              // 强制部署（覆盖现有策略）
}

// DeployStrategyResponse 部署策略响应结构
type DeployStrategyResponse struct {
	StrategyID string `json:"strategy_id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

// @Title 部署策略
// @Description 部署最优参数/策略到实盘配置，包含权限校验与操作日志
// @Param body body DeployStrategyRequest true "部署策略请求参数"
// @Success 200 {object} DeployStrategyResponse
// @router /api/deploy_strategy [post]
func (this *DeployStrategyController) Post() {
	var req DeployStrategyRequest
	if err := json.Unmarshal(this.Ctx.Input.RequestBody, &req); err != nil {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Invalid request format: " + err.Error(),
		}
		this.ServeJSON()
		return
	}

	// 权限校验
	if !this.checkPermission("deploy_strategy") {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Permission denied: insufficient privileges to deploy strategy",
		}
		this.ServeJSON()
		return
	}

	// 验证请求参数
	if req.Name == "" {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Strategy name is required",
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

	if req.Strategy == "" {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Strategy expression is required",
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

	// 验证回测结果是否存在
	if req.BacktestTaskID != "" && req.BacktestResultID != "" {
		if !this.validateBacktestResult(req.BacktestTaskID, req.BacktestResultID) {
			this.Data["json"] = map[string]interface{}{
				"status":  "error",
				"message": "Invalid backtest result reference",
			}
			this.ServeJSON()
			return
		}
	}

	o := orm.NewOrm()

	// 检查是否已存在该币种的策略
	var existingStrategy models.DeployedStrategy
	err := o.QueryTable("deployed_strategies").Filter("symbol", req.Symbol).Filter("status", "active").One(&existingStrategy)
	if err == nil && !req.Force {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("Active strategy already exists for symbol %s. Use force=true to override.", req.Symbol),
		}
		this.ServeJSON()
		return
	}

	// 生成策略ID
	strategyID := this.generateStrategyID()

	// 记录操作日志
	this.logOperation("DEPLOY_STRATEGY", "deployed_strategy", strategyID, req)

	// 如果存在现有策略，先停用
	if err == nil && req.Force {
		this.logOperation("DEACTIVATE_STRATEGY", "deployed_strategy", existingStrategy.StrategyID, map[string]interface{}{
			"reason":         "replaced_by_new_strategy",
			"new_strategy_id": strategyID,
		})
		
		o.QueryTable("deployed_strategies").Filter("id", existingStrategy.ID).Update(orm.Params{
			"status":           "inactive",
			"last_update_time": time.Now().Unix(),
			"update_time":      time.Now().Unix(),
		})
	}

	// 创建新的部署策略
	deployedStrategy := models.DeployedStrategy{
		StrategyID:       strategyID,
		Name:             req.Name,
		Symbol:           req.Symbol,
		Strategy:         req.Strategy,
		Status:           "active",
		DeployedBy:       this.getCurrentUser(),
		DeployTime:       time.Now().Unix(),
		BacktestTaskID:   req.BacktestTaskID,
		BacktestResultID: req.BacktestResultID,
		LiveReturn:       0,
		LiveTradeCount:   0,
		LastUpdateTime:   time.Now().Unix(),
		CreateTime:       time.Now().Unix(),
		UpdateTime:       time.Now().Unix(),
	}

	// 序列化参数
	paramBytes, _ := json.Marshal(req.Parameters)
	deployedStrategy.Parameters = string(paramBytes)

	if _, err := o.Insert(&deployedStrategy); err != nil {
		logs.Error("Failed to deploy strategy:", err)
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Failed to deploy strategy",
		}
		this.ServeJSON()
		return
	}

	// 应用策略到实盘交易配置
	if err := this.applyStrategyToLiveTrading(deployedStrategy); err != nil {
		logs.Error("Failed to apply strategy to live trading:", err)
		
		// 回滚部署
		o.QueryTable("deployed_strategies").Filter("id", deployedStrategy.ID).Update(orm.Params{
			"status":      "error",
			"update_time": time.Now().Unix(),
		})
		
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Failed to apply strategy to live trading: " + err.Error(),
		}
		this.ServeJSON()
		return
	}

	this.Data["json"] = DeployStrategyResponse{
		StrategyID: strategyID,
		Status:     "success",
		Message:    "Strategy deployed successfully",
	}
	this.ServeJSON()
}

// @Title 获取已部署策略列表
// @Description 获取已部署策略列表，支持分页和过滤
// @Param page query int false "页码，默认1"
// @Param pageSize query int false "每页数量，默认20"
// @Param status query string false "状态过滤"
// @Param symbol query string false "币种过滤"
// @Success 200 {object} object
// @router /api/deploy_strategy [get]
func (this *DeployStrategyController) Get() {
	page, _ := this.GetInt("page", 1)
	pageSize, _ := this.GetInt("pageSize", 20)
	status := this.GetString("status")
	symbol := this.GetString("symbol")

	offset := (page - 1) * pageSize
	o := orm.NewOrm()
	
	var strategies []models.DeployedStrategy
	qs := o.QueryTable("deployed_strategies")
	
	if status != "" {
		qs = qs.Filter("status", status)
	}
	
	if symbol != "" {
		qs = qs.Filter("symbol", symbol)
	}
	
	// 获取总数
	total, _ := qs.Count()
	
	// 获取数据
	qs.OrderBy("-deploy_time").Limit(pageSize, offset).All(&strategies)

	this.Data["json"] = map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"strategies": strategies,
			"total":      total,
			"page":       page,
			"pageSize":   pageSize,
		},
	}
	this.ServeJSON()
}

// @Title 更新策略状态
// @Description 更新已部署策略的状态（激活/停用）
// @Param strategyId path string true "策略ID"
// @Param body body object true "更新参数 {\"status\": \"active|inactive\"}"
// @Success 200 {object} object
// @router /api/deploy_strategy/:strategyId [put]
func (this *DeployStrategyController) Put() {
	strategyID := this.Ctx.Input.Param(":strategyId")
	
	var req map[string]interface{}
	if err := json.Unmarshal(this.Ctx.Input.RequestBody, &req); err != nil {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Invalid request format: " + err.Error(),
		}
		this.ServeJSON()
		return
	}

	// 权限校验
	if !this.checkPermission("manage_strategy") {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Permission denied: insufficient privileges to manage strategy",
		}
		this.ServeJSON()
		return
	}

	newStatus, ok := req["status"].(string)
	if !ok || (newStatus != "active" && newStatus != "inactive") {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Invalid status value. Must be 'active' or 'inactive'",
		}
		this.ServeJSON()
		return
	}

	o := orm.NewOrm()
	
	// 检查策略是否存在
	var strategy models.DeployedStrategy
	if err := o.QueryTable("deployed_strategies").Filter("strategy_id", strategyID).One(&strategy); err != nil {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Strategy not found",
		}
		this.ServeJSON()
		return
	}

	// 记录操作日志
	this.logOperation("UPDATE_STRATEGY_STATUS", "deployed_strategy", strategyID, map[string]interface{}{
		"old_status": strategy.Status,
		"new_status": newStatus,
	})

	// 更新策略状态
	if _, err := o.QueryTable("deployed_strategies").Filter("strategy_id", strategyID).Update(orm.Params{
		"status":           newStatus,
		"last_update_time": time.Now().Unix(),
		"update_time":      time.Now().Unix(),
	}); err != nil {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Failed to update strategy status",
		}
		this.ServeJSON()
		return
	}

	// 应用状态变更到实盘交易
	strategy.Status = newStatus
	if err := this.applyStrategyToLiveTrading(strategy); err != nil {
		logs.Error("Failed to apply strategy status change to live trading:", err)
	}

	this.Data["json"] = map[string]interface{}{
		"status":  "success",
		"message": "Strategy status updated successfully",
	}
	this.ServeJSON()
}

// @Title 删除策略
// @Description 删除已部署的策略
// @Param strategyId path string true "策略ID"
// @Success 200 {object} object
// @router /api/deploy_strategy/:strategyId [delete]
func (this *DeployStrategyController) Delete() {
	strategyID := this.Ctx.Input.Param(":strategyId")
	
	// 权限校验
	if !this.checkPermission("delete_strategy") {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Permission denied: insufficient privileges to delete strategy",
		}
		this.ServeJSON()
		return
	}

	o := orm.NewOrm()
	
	// 检查策略是否存在
	var strategy models.DeployedStrategy
	if err := o.QueryTable("deployed_strategies").Filter("strategy_id", strategyID).One(&strategy); err != nil {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Strategy not found",
		}
		this.ServeJSON()
		return
	}

	// 记录操作日志
	this.logOperation("DELETE_STRATEGY", "deployed_strategy", strategyID, strategy)

	// 删除策略
	if _, err := o.QueryTable("deployed_strategies").Filter("strategy_id", strategyID).Delete(); err != nil {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Failed to delete strategy",
		}
		this.ServeJSON()
		return
	}

	this.Data["json"] = map[string]interface{}{
		"status":  "success",
		"message": "Strategy deleted successfully",
	}
	this.ServeJSON()
}

// @Title 获取操作日志
// @Description 获取操作日志列表
// @Param page query int false "页码，默认1"
// @Param pageSize query int false "每页数量，默认20"
// @Param operation query string false "操作类型过滤"
// @Param resourceType query string false "资源类型过滤"
// @Success 200 {object} object
// @router /api/operation_logs [get]
func (this *DeployStrategyController) GetOperationLogs() {
	// 权限校验
	if !this.checkPermission("view_operation_logs") {
		this.Data["json"] = map[string]interface{}{
			"status":  "error",
			"message": "Permission denied: insufficient privileges to view operation logs",
		}
		this.ServeJSON()
		return
	}

	page, _ := this.GetInt("page", 1)
	pageSize, _ := this.GetInt("pageSize", 20)
	operation := this.GetString("operation")
	resourceType := this.GetString("resourceType")

	offset := (page - 1) * pageSize
	o := orm.NewOrm()
	
	var logs []models.OperationLog
	qs := o.QueryTable("operation_logs")
	
	if operation != "" {
		qs = qs.Filter("operation", operation)
	}
	
	if resourceType != "" {
		qs = qs.Filter("resource_type", resourceType)
	}
	
	// 获取总数
	total, _ := qs.Count()
	
	// 获取数据
	qs.OrderBy("-create_time").Limit(pageSize, offset).All(&logs)

	this.Data["json"] = map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"logs":     logs,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	}
	this.ServeJSON()
}

// 权限校验
func (this *DeployStrategyController) checkPermission(permission string) bool {
	// 这里实现具体的权限校验逻辑
	// 可以根据JWT token、角色、用户权限等进行校验
	// 暂时返回true，后续可以根据实际需求实现
	user := this.getCurrentUser()
	if user == "" {
		return false
	}
	
	// 简化权限校验：admin用户有所有权限
	return user == "admin"
}

// 验证回测结果
func (this *DeployStrategyController) validateBacktestResult(taskID, resultID string) bool {
	o := orm.NewOrm()
	var result models.BacktestResult
	err := o.QueryTable("backtest_results").Filter("task_id", taskID).Filter("result_id", resultID).One(&result)
	return err == nil
}

// 生成策略ID
func (this *DeployStrategyController) generateStrategyID() string {
	return fmt.Sprintf("ds_%d_%d", time.Now().Unix(), time.Now().UnixNano()%1000000)
}

// 获取当前用户
func (this *DeployStrategyController) getCurrentUser() string {
	// 从JWT token或session中获取用户信息
	// 这里先返回默认值，后续可以根据实际认证系统调整
	return "admin"
}

// 记录操作日志
func (this *DeployStrategyController) logOperation(operation, resourceType, resourceID string, details interface{}) {
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

// 应用策略到实盘交易配置
func (this *DeployStrategyController) applyStrategyToLiveTrading(strategy models.DeployedStrategy) error {
	// 这里实现将策略应用到实盘交易配置的逻辑
	// 可以包括：
	// 1. 更新symbols表的策略配置
	// 2. 重新加载交易引擎配置
	// 3. 发送通知到交易模块
	
	o := orm.NewOrm()
	
	// 解析参数
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(strategy.Parameters), &params); err != nil {
		return fmt.Errorf("failed to parse strategy parameters: %v", err)
	}
	
	// 更新symbols表的策略配置
	updateParams := orm.Params{
		"strategy":      strategy.Strategy,
		"technology":    strategy.Parameters, // 将参数存储为技术配置
		"strategy_type": "custom",
	}
	
	if strategy.Status == "active" {
		updateParams["enable"] = 1
	} else {
		updateParams["enable"] = 0
	}
	
	if _, err := o.QueryTable("symbols").Filter("symbol", strategy.Symbol).Update(updateParams); err != nil {
		return fmt.Errorf("failed to update symbols table: %v", err)
	}
	
	logs.Info("Strategy applied to live trading:", strategy.StrategyID, strategy.Symbol, strategy.Status)
	return nil
}

// 更新策略实盘表现
func (this *DeployStrategyController) updateLivePerformance(strategyID string, liveReturn float64, tradeCount int64) error {
	o := orm.NewOrm()
	
	_, err := o.QueryTable("deployed_strategies").Filter("strategy_id", strategyID).Update(orm.Params{
		"live_return":       liveReturn,
		"live_trade_count":  tradeCount,
		"last_update_time":  time.Now().Unix(),
		"update_time":       time.Now().Unix(),
	})
	
	return err
}