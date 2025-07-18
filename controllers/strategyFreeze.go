package controllers

import (
	"encoding/json"
	"go_binance_futures/models"
	"go_binance_futures/utils"
	"strconv"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
)

type StrategyFreezeController struct {
	web.Controller
}

// 请求体结构体，兼容JSON
type FreezeReq struct {
	Symbol       string `json:"symbol"`
	StrategyName string `json:"strategy_name"`
	TradeType    string `json:"trade_type"`
}

// 获取冻结配置列表
func (c *StrategyFreezeController) Get() {
	page, _ := c.GetInt("page", 1)
	pageSize, _ := c.GetInt("pageSize", 20)
	
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	
	freezeService := utils.NewFreezeService()
	freezes, total, err := freezeService.GetAllFreezeConfigs(page, pageSize)
	if err != nil {
		logs.Error("获取冻结配置失败:", err)
		c.Data["json"] = map[string]interface{}{
			"code":    500,
			"message": "获取冻结配置失败",
			"error":   err.Error(),
		}
		c.ServeJSON()
		return
	}
	
	// 计算剩余冻结时间
	now := time.Now().Unix()
	for i := range freezes {
		if freezes[i].FreezeUntil > now {
			freezes[i].FreezeUntil = freezes[i].FreezeUntil - now
		} else {
			freezes[i].FreezeUntil = 0
		}
	}
	
	c.Data["json"] = map[string]interface{}{
		"code":    200,
		"message": "获取成功",
		"data": map[string]interface{}{
			"list":     freezes,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	}
	c.ServeJSON()
}

// 创建或更新冻结配置
func (c *StrategyFreezeController) Post() {
	var freeze models.StrategyFreeze
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &freeze); err != nil {
		logs.Error("解析请求参数失败:", err)
		c.Data["json"] = map[string]interface{}{
			"code":    400,
			"message": "参数格式错误",
			"error":   err.Error(),
		}
		c.ServeJSON()
		return
	}
	
	// 验证必要参数
	if freeze.Symbol == "" || freeze.StrategyName == "" || freeze.TradeType == "" {
		c.Data["json"] = map[string]interface{}{
			"code":    400,
			"message": "symbol、strategy_name、trade_type不能为空",
		}
		c.ServeJSON()
		return
	}
	
	// 验证trade_type
	if freeze.TradeType != "real" && freeze.TradeType != "test" {
		c.Data["json"] = map[string]interface{}{
			"code":    400,
			"message": "trade_type必须是real或test",
		}
		c.ServeJSON()
		return
	}
	
	// 设置默认值
	if freeze.FreezeOnLossCount <= 0 {
		freeze.FreezeOnLossCount = 5
	}
	if freeze.FreezeHours <= 0 {
		freeze.FreezeHours = 24
	}
	
	freezeService := utils.NewFreezeService()
	
	// 检查是否已存在
	existingFreeze, err := freezeService.GetFreezeConfig(freeze.Symbol, freeze.StrategyName, freeze.TradeType)
	if err == nil {
		// 更新现有配置
		existingFreeze.FreezeOnLossCount = freeze.FreezeOnLossCount
		existingFreeze.FreezeHours = freeze.FreezeHours
		if freeze.LossCount >= 0 {
			existingFreeze.LossCount = freeze.LossCount
		}
		if freeze.FreezeUntil > 0 {
			existingFreeze.FreezeUntil = freeze.FreezeUntil
		}
		
		err = freezeService.UpdateFreezeConfig(existingFreeze)
		if err != nil {
			logs.Error("更新冻结配置失败:", err)
			c.Data["json"] = map[string]interface{}{
				"code":    500,
				"message": "更新冻结配置失败",
				"error":   err.Error(),
			}
			c.ServeJSON()
			return
		}
		
		c.Data["json"] = map[string]interface{}{
			"code":    200,
			"message": "更新成功",
			"data":    existingFreeze,
		}
	} else {
		// 创建新配置
		newFreeze, err := freezeService.CreateDefaultFreezeConfig(freeze.Symbol, freeze.StrategyName, freeze.TradeType)
		if err != nil {
			logs.Error("创建冻结配置失败:", err)
			c.Data["json"] = map[string]interface{}{
				"code":    500,
				"message": "创建冻结配置失败",
				"error":   err.Error(),
			}
			c.ServeJSON()
			return
		}
		
		// 更新配置
		newFreeze.FreezeOnLossCount = freeze.FreezeOnLossCount
		newFreeze.FreezeHours = freeze.FreezeHours
		if freeze.LossCount >= 0 {
			newFreeze.LossCount = freeze.LossCount
		}
		if freeze.FreezeUntil > 0 {
			newFreeze.FreezeUntil = freeze.FreezeUntil
		}
		
		err = freezeService.UpdateFreezeConfig(newFreeze)
		if err != nil {
			logs.Error("更新新创建的冻结配置失败:", err)
			c.Data["json"] = map[string]interface{}{
				"code":    500,
				"message": "更新新创建的冻结配置失败",
				"error":   err.Error(),
			}
			c.ServeJSON()
			return
		}
		
		c.Data["json"] = map[string]interface{}{
			"code":    200,
			"message": "创建成功",
			"data":    newFreeze,
		}
	}
	
	c.ServeJSON()
}

// 获取单个冻结配置（兼容query和body）
func (c *StrategyFreezeController) GetOne() {
	var symbol, strategyName, tradeType string
	// 优先尝试从body解析
	var req FreezeReq
	if len(c.Ctx.Input.RequestBody) > 0 {
		if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err == nil {
			symbol = req.Symbol
			strategyName = req.StrategyName
			tradeType = req.TradeType
		}
	}
	// 如果body没有，则尝试query
	if symbol == "" {
		symbol = c.GetString("symbol")
	}
	if strategyName == "" {
		strategyName = c.GetString("strategy_name")
	}
	if tradeType == "" {
		tradeType = c.GetString("trade_type")
	}

	if symbol == "" || strategyName == "" || tradeType == "" {
		c.Data["json"] = map[string]interface{}{
			"code":    400,
			"message": "symbol、strategy_name、trade_type不能为空",
		}
		c.ServeJSON()
		return
	}
	
	freezeService := utils.NewFreezeService()
	freeze, err := freezeService.GetFreezeConfig(symbol, strategyName, tradeType)
	if err != nil {
		logs.Error("获取冻结配置失败:", err)
		c.Data["json"] = map[string]interface{}{
			"code":    500,
			"message": "获取冻结配置失败",
			"error":   err.Error(),
		}
		c.ServeJSON()
		return
	}
	
	// 计算剩余冻结时间
	now := time.Now().Unix()
	remainingTime := int64(0)
	if freeze.FreezeUntil > now {
		remainingTime = freeze.FreezeUntil - now
	}
	
	c.Data["json"] = map[string]interface{}{
		"code":    200,
		"message": "获取成功",
		"data": map[string]interface{}{
			"config":          freeze,
			"is_frozen":       freeze.FreezeUntil > now,
			"remaining_time":  remainingTime,
		},
	}
	c.ServeJSON()
}

// 更新冻结配置
func (c *StrategyFreezeController) Edit() {
	idStr := c.Ctx.Input.Param(":id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.Data["json"] = map[string]interface{}{
			"code":    400,
			"message": "无效的ID",
		}
		c.ServeJSON()
		return
	}
	
	var freeze models.StrategyFreeze
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &freeze); err != nil {
		logs.Error("解析请求参数失败:", err)
		c.Data["json"] = map[string]interface{}{
			"code":    400,
			"message": "参数格式错误",
			"error":   err.Error(),
		}
		c.ServeJSON()
		return
	}
	
	freeze.ID = id
	freezeService := utils.NewFreezeService()
	err = freezeService.UpdateFreezeConfig(&freeze)
	if err != nil {
		logs.Error("更新冻结配置失败:", err)
		c.Data["json"] = map[string]interface{}{
			"code":    500,
			"message": "更新冻结配置失败",
			"error":   err.Error(),
		}
		c.ServeJSON()
		return
	}
	
	c.Data["json"] = map[string]interface{}{
		"code":    200,
		"message": "更新成功",
		"data":    freeze,
	}
	c.ServeJSON()
}

// 手动解除冻结（兼容JSON body）
func (c *StrategyFreezeController) Unfreeze() {
	var symbol, strategyName, tradeType string
	var req FreezeReq
	if len(c.Ctx.Input.RequestBody) > 0 {
		if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err == nil {
			symbol = req.Symbol
			strategyName = req.StrategyName
			tradeType = req.TradeType
		}
	}
	if symbol == "" {
		symbol = c.GetString("symbol")
	}
	if strategyName == "" {
		strategyName = c.GetString("strategy_name")
	}
	if tradeType == "" {
		tradeType = c.GetString("trade_type")
	}
	
	if symbol == "" || strategyName == "" || tradeType == "" {
		c.Data["json"] = map[string]interface{}{
			"code":    400,
			"message": "symbol、strategy_name、trade_type不能为空",
		}
		c.ServeJSON()
		return
	}
	
	freezeService := utils.NewFreezeService()
	err := freezeService.UnfreezeManually(symbol, strategyName, tradeType)
	if err != nil {
		logs.Error("解除冻结失败:", err)
		c.Data["json"] = map[string]interface{}{
			"code":    500,
			"message": "解除冻结失败",
			"error":   err.Error(),
		}
		c.ServeJSON()
		return
	}
	
	c.Data["json"] = map[string]interface{}{
		"code":    200,
		"message": "解除冻结成功",
	}
	c.ServeJSON()
}

// 重置亏损次数（兼容JSON body）
func (c *StrategyFreezeController) ResetLossCount() {
	var symbol, strategyName, tradeType string
	var req FreezeReq
	if len(c.Ctx.Input.RequestBody) > 0 {
		if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err == nil {
			symbol = req.Symbol
			strategyName = req.StrategyName
			tradeType = req.TradeType
		}
	}
	if symbol == "" {
		symbol = c.GetString("symbol")
	}
	if strategyName == "" {
		strategyName = c.GetString("strategy_name")
	}
	if tradeType == "" {
		tradeType = c.GetString("trade_type")
	}
	
	if symbol == "" || strategyName == "" || tradeType == "" {
		c.Data["json"] = map[string]interface{}{
			"code":    400,
			"message": "symbol、strategy_name、trade_type不能为空",
		}
		c.ServeJSON()
		return
	}
	
	freezeService := utils.NewFreezeService()
	err := freezeService.ResetLossCount(symbol, strategyName, tradeType)
	if err != nil {
		logs.Error("重置亏损次数失败:", err)
		c.Data["json"] = map[string]interface{}{
			"code":    500,
			"message": "重置亏损次数失败",
			"error":   err.Error(),
		}
		c.ServeJSON()
		return
	}
	
	c.Data["json"] = map[string]interface{}{
		"code":    200,
		"message": "重置亏损次数成功",
	}
	c.ServeJSON()
}

// Options 获取币种、策略、交易类型选项
func (c *StrategyFreezeController) Options() {
	freezeService := utils.NewFreezeService()

	// 获取所有symbol
	symbols, err := freezeService.GetAllSymbols()
	if err != nil {
		c.Data["json"] = map[string]interface{}{
			"code":    500,
			"message": "获取symbol失败",
			"error":   err.Error(),
		}
		c.ServeJSON()
		return
	}
	// 获取所有strategy_name
	strategies, err := freezeService.GetAllStrategies()
	if err != nil {
		c.Data["json"] = map[string]interface{}{
			"code":    500,
			"message": "获取策略失败",
			"error":   err.Error(),
		}
		c.ServeJSON()
		return
	}
	// 交易类型一般是固定的
	tradeTypes := []map[string]string{
		{"label": "实盘", "value": "real"},
		{"label": "测试", "value": "test"},
	}

	// 构造前端需要的格式
	symbolOpts := []map[string]string{}
	for _, s := range symbols {
		symbolOpts = append(symbolOpts, map[string]string{"label": s, "value": s})
	}
	strategyOpts := []map[string]string{}
	for _, s := range strategies {
		strategyOpts = append(strategyOpts, map[string]string{"label": s, "value": s})
	}

	c.Data["json"] = map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"symbols":    symbolOpts,
			"strategies": strategyOpts,
			"tradeTypes": tradeTypes,
		},
	}
	c.ServeJSON()
}

// Delete选项
func (c *StrategyFreezeController) Delete() {
	idStr := c.Ctx.Input.Param(":id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.Data["json"] = map[string]interface{}{
			"code":    400,
			"message": "无效的ID",
		}
		c.ServeJSON()
		return
	}

	freezeService := utils.NewFreezeService()
	err = freezeService.DeleteFreezeConfig(id)
	if err != nil {
		c.Data["json"] = map[string]interface{}{
			"code":    500,
			"message": "删除失败",
			"error":   err.Error(),
		}
		c.ServeJSON()
		return
	}

	c.Data["json"] = map[string]interface{}{
		"code":    200,
		"message": "删除成功",
	}
	c.ServeJSON()
}
