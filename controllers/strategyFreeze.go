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

// 获取单个冻结配置
func (c *StrategyFreezeController) GetOne() {
	symbol := c.GetString("symbol")
	strategyName := c.GetString("strategy_name")
	tradeType := c.GetString("trade_type")
	
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

// 手动解除冻结
func (c *StrategyFreezeController) Unfreeze() {
	symbol := c.GetString("symbol")
	strategyName := c.GetString("strategy_name")
	tradeType := c.GetString("trade_type")
	
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

// 重置亏损次数
func (c *StrategyFreezeController) ResetLossCount() {
	symbol := c.GetString("symbol")
	strategyName := c.GetString("strategy_name")
	tradeType := c.GetString("trade_type")
	
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