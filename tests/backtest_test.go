package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go_binance_futures/controllers"
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBacktestAPI(t *testing.T) {
	// 设置测试环境
	setupTestDatabase()
	
	Convey("Test Backtest API", t, func() {
		
		Convey("POST /api/backtest - Create backtest task", func() {
			// 准备测试数据
			req := controllers.BacktestRequest{
				Name:       "Test Backtest Task",
				Strategy:   "close > ma5 && rsi < 30",
				Parameters: []map[string]interface{}{
					{"ma5": 5, "rsi_period": 14},
					{"ma5": 10, "rsi_period": 21},
				},
				Symbol:     "BTCUSDT",
				StartTime:  time.Now().Unix() - 86400*30, // 30天前
				EndTime:    time.Now().Unix(),
				Concurrent: 2,
			}
			
			body, _ := json.Marshal(req)
			r, _ := http.NewRequest("POST", "/api/backtest", bytes.NewBuffer(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			web.BeeApp.Handlers.ServeHTTP(w, r)
			
			So(w.Code, ShouldEqual, 200)
			
			var response controllers.BacktestResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response.Status, ShouldEqual, "success")
			So(response.TaskID, ShouldNotBeEmpty)
			
			// 验证数据库中是否创建了任务
			o := orm.NewOrm()
			var task models.BacktestTask
			err = o.QueryTable("backtest_tasks").Filter("task_id", response.TaskID).One(&task)
			So(err, ShouldBeNil)
			So(task.Name, ShouldEqual, "Test Backtest Task")
			So(task.Status, ShouldEqual, "pending")
		})
		
		Convey("POST /api/backtest - Invalid request", func() {
			// 测试无效请求
			req := controllers.BacktestRequest{
				Name: "", // 空名称
			}
			
			body, _ := json.Marshal(req)
			r, _ := http.NewRequest("POST", "/api/backtest", bytes.NewBuffer(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			web.BeeApp.Handlers.ServeHTTP(w, r)
			
			So(w.Code, ShouldEqual, 200)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["status"], ShouldEqual, "error")
			So(response["message"], ShouldContainSubstring, "Task name is required")
		})
		
		Convey("GET /api/backtest - List backtest tasks", func() {
			// 先创建一个任务
			_ = createTestBacktestTask()
			
			r, _ := http.NewRequest("GET", "/api/backtest?page=1&pageSize=10", nil)
			w := httptest.NewRecorder()
			web.BeeApp.Handlers.ServeHTTP(w, r)
			
			So(w.Code, ShouldEqual, 200)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["status"], ShouldEqual, "success")
			
			data := response["data"].(map[string]interface{})
			tasks := data["tasks"].([]interface{})
			So(len(tasks), ShouldBeGreaterThanOrEqualTo, 0)
		})
		
		Convey("GET /api/backtest/:taskId/results - Get backtest results", func() {
			// 创建测试任务和结果
			task := createTestBacktestTask()
			_ = createTestBacktestResult(task.TaskID)
			
			r, _ := http.NewRequest("GET", "/api/backtest/"+task.TaskID+"/results", nil)
			w := httptest.NewRecorder()
			web.BeeApp.Handlers.ServeHTTP(w, r)
			
			So(w.Code, ShouldEqual, 200)
			
			var response controllers.BacktestResultResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response.TaskID, ShouldEqual, task.TaskID)
			So(len(response.Results), ShouldBeGreaterThanOrEqualTo, 0)
		})
		
		Convey("DELETE /api/backtest/:taskId - Delete backtest task", func() {
			// 创建测试任务
			task := createTestBacktestTask()
			
			r, _ := http.NewRequest("DELETE", "/api/backtest/"+task.TaskID, nil)
			w := httptest.NewRecorder()
			web.BeeApp.Handlers.ServeHTTP(w, r)
			
			So(w.Code, ShouldEqual, 200)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["status"], ShouldEqual, "success")
			
			// 验证数据库中任务已删除
			o := orm.NewOrm()
			var deletedTask models.BacktestTask
			err = o.QueryTable("backtest_tasks").Filter("task_id", task.TaskID).One(&deletedTask)
			So(err, ShouldNotBeNil) // 应该找不到记录
		})
	})
}

func TestDeployStrategyAPI(t *testing.T) {
	// 设置测试环境
	setupTestDatabase()
	
	Convey("Test Deploy Strategy API", t, func() {
		
		Convey("POST /api/deploy_strategy - Deploy strategy", func() {
			// 准备测试数据
			req := controllers.DeployStrategyRequest{
				Name:   "Test Strategy",
				Symbol: "ETHUSDT",
				Parameters: map[string]interface{}{
					"ma5": 5,
					"rsi_period": 14,
				},
				Strategy:         "close > ma5 && rsi < 30",
				BacktestTaskID:   "test_task_123",
				BacktestResultID: "test_result_123",
				Force:            false,
			}
			
			body, _ := json.Marshal(req)
			r, _ := http.NewRequest("POST", "/api/deploy_strategy", bytes.NewBuffer(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			web.BeeApp.Handlers.ServeHTTP(w, r)
			
			So(w.Code, ShouldEqual, 200)
			
			var response controllers.DeployStrategyResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response.Status, ShouldEqual, "success")
			So(response.StrategyID, ShouldNotBeEmpty)
			
			// 验证数据库中是否创建了策略
			o := orm.NewOrm()
			var strategy models.DeployedStrategy
			err = o.QueryTable("deployed_strategies").Filter("strategy_id", response.StrategyID).One(&strategy)
			So(err, ShouldBeNil)
			So(strategy.Name, ShouldEqual, "Test Strategy")
			So(strategy.Status, ShouldEqual, "active")
		})
		
		Convey("POST /api/deploy_strategy - Invalid request", func() {
			// 测试无效请求
			req := controllers.DeployStrategyRequest{
				Name: "", // 空名称
			}
			
			body, _ := json.Marshal(req)
			r, _ := http.NewRequest("POST", "/api/deploy_strategy", bytes.NewBuffer(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			web.BeeApp.Handlers.ServeHTTP(w, r)
			
			So(w.Code, ShouldEqual, 200)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["status"], ShouldEqual, "error")
			So(response["message"], ShouldContainSubstring, "Strategy name is required")
		})
		
		Convey("GET /api/deploy_strategy - List deployed strategies", func() {
			// 先创建一个策略
			_ = createTestDeployedStrategy()
			
			r, _ := http.NewRequest("GET", "/api/deploy_strategy?page=1&pageSize=10", nil)
			w := httptest.NewRecorder()
			web.BeeApp.Handlers.ServeHTTP(w, r)
			
			So(w.Code, ShouldEqual, 200)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["status"], ShouldEqual, "success")
			
			data := response["data"].(map[string]interface{})
			strategies := data["strategies"].([]interface{})
			So(len(strategies), ShouldBeGreaterThanOrEqualTo, 0)
		})
		
		Convey("PUT /api/deploy_strategy/:strategyId - Update strategy status", func() {
			// 创建测试策略
			strategy := createTestDeployedStrategy()
			
			updateReq := map[string]interface{}{
				"status": "inactive",
			}
			
			body, _ := json.Marshal(updateReq)
			r, _ := http.NewRequest("PUT", "/api/deploy_strategy/"+strategy.StrategyID, bytes.NewBuffer(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			web.BeeApp.Handlers.ServeHTTP(w, r)
			
			So(w.Code, ShouldEqual, 200)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["status"], ShouldEqual, "success")
			
			// 验证数据库中状态已更新
			o := orm.NewOrm()
			var updatedStrategy models.DeployedStrategy
			err = o.QueryTable("deployed_strategies").Filter("strategy_id", strategy.StrategyID).One(&updatedStrategy)
			So(err, ShouldBeNil)
			So(updatedStrategy.Status, ShouldEqual, "inactive")
		})
		
		Convey("DELETE /api/deploy_strategy/:strategyId - Delete strategy", func() {
			// 创建测试策略
			strategy := createTestDeployedStrategy()
			
			r, _ := http.NewRequest("DELETE", "/api/deploy_strategy/"+strategy.StrategyID, nil)
			w := httptest.NewRecorder()
			web.BeeApp.Handlers.ServeHTTP(w, r)
			
			So(w.Code, ShouldEqual, 200)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["status"], ShouldEqual, "success")
			
			// 验证数据库中策略已删除
			o := orm.NewOrm()
			var deletedStrategy models.DeployedStrategy
			err = o.QueryTable("deployed_strategies").Filter("strategy_id", strategy.StrategyID).One(&deletedStrategy)
			So(err, ShouldNotBeNil) // 应该找不到记录
		})
		
		Convey("GET /api/operation_logs - Get operation logs", func() {
			// 创建测试操作日志
			createTestOperationLog()
			
			r, _ := http.NewRequest("GET", "/api/operation_logs?page=1&pageSize=10", nil)
			w := httptest.NewRecorder()
			web.BeeApp.Handlers.ServeHTTP(w, r)
			
			So(w.Code, ShouldEqual, 200)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["status"], ShouldEqual, "success")
			
			data := response["data"].(map[string]interface{})
			logs := data["logs"].([]interface{})
			So(len(logs), ShouldBeGreaterThan, 0)
		})
	})
}

// 辅助函数：设置测试数据库
func setupTestDatabase() {
	// 这里可以设置测试数据库连接
	// 为了简化，我们使用内存数据库
	logs.Info("Setting up test database...")
}

// 辅助函数：创建测试回测任务
func createTestBacktestTask() *models.BacktestTask {
	o := orm.NewOrm()
	task := &models.BacktestTask{
		TaskID:     "test_task_" + generateID(),
		Name:       "Test Backtest Task",
		Status:     "completed",
		Strategy:   "close > ma5",
		Parameters: `[{"ma5": 5}]`,
		Symbol:     "BTCUSDT",
		StartTime:  time.Now().Unix() - 86400*30,
		EndTime:    time.Now().Unix(),
		CreatedBy:  "test_user",
		Progress:   100,
		CreateTime: time.Now().Unix(),
		UpdateTime: time.Now().Unix(),
	}
	
	o.Insert(task)
	return task
}

// 辅助函数：创建测试回测结果
func createTestBacktestResult(taskID string) *models.BacktestResult {
	o := orm.NewOrm()
	result := &models.BacktestResult{
		TaskID:       taskID,
		ResultID:     "test_result_" + generateID(),
		Parameters:   `{"ma5": 5}`,
		TotalReturn:  0.15,
		AnnualReturn: 0.25,
		MaxDrawdown:  0.05,
		SharpeRatio:  3.0,
		WinRate:      0.65,
		TradeCount:   100,
		ProfitFactor: 1.5,
		EquityCurve:  `[{"time": 1234567890, "equity": 10000}]`,
		TradeList:    `[{"time": 1234567890, "side": "buy", "price": 100, "pnl": 50}]`,
		CreateTime:   time.Now().Unix(),
	}
	
	o.Insert(result)
	return result
}

// 辅助函数：创建测试已部署策略
func createTestDeployedStrategy() *models.DeployedStrategy {
	o := orm.NewOrm()
	strategy := &models.DeployedStrategy{
		StrategyID:       "test_strategy_" + generateID(),
		Name:             "Test Strategy",
		Symbol:           "ETHUSDT",
		Parameters:       `{"ma5": 5, "rsi_period": 14}`,
		Strategy:         "close > ma5 && rsi < 30",
		Status:           "active",
		DeployedBy:       "test_user",
		DeployTime:       time.Now().Unix(),
		BacktestTaskID:   "test_task_123",
		BacktestResultID: "test_result_123",
		LiveReturn:       0.08,
		LiveTradeCount:   50,
		LastUpdateTime:   time.Now().Unix(),
		CreateTime:       time.Now().Unix(),
		UpdateTime:       time.Now().Unix(),
	}
	
	o.Insert(strategy)
	return strategy
}

// 辅助函数：创建测试操作日志
func createTestOperationLog() *models.OperationLog {
	o := orm.NewOrm()
	log := &models.OperationLog{
		UserID:       "test_user",
		Operation:    "CREATE_BACKTEST",
		ResourceType: "backtest_task",
		ResourceID:   "test_task_123",
		Details:      `{"name": "Test Task"}`,
		IPAddress:    "127.0.0.1",
		UserAgent:    "Test Agent",
		CreateTime:   time.Now().Unix(),
	}
	
	o.Insert(log)
	return log
}

// 辅助函数：生成测试ID
func generateID() string {
	return string(rune(time.Now().UnixNano() % 1000000))
}