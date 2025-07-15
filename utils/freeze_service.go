package utils

import (
	"go_binance_futures/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

// 风控服务
type FreezeService struct {
	orm orm.Ormer
}

func NewFreezeService() *FreezeService {
	return &FreezeService{
		orm: orm.NewOrm(),
	}
}

// 检查是否被冻结
func (fs *FreezeService) IsFrozen(symbol, strategyName, tradeType string) bool {
	var freeze models.StrategyFreeze
	err := fs.orm.QueryTable("strategy_freeze").
		Filter("symbol", symbol).
		Filter("strategy_name", strategyName).
		Filter("trade_type", tradeType).
		One(&freeze)

	if err != nil {
		if err == orm.ErrNoRows {
			return false
		}
		logs.Error("查询冻结状态失败:", err)
		return false
	}

	now := time.Now().Unix()
	return freeze.FreezeUntil > now
}

// 获取冻结配置
func (fs *FreezeService) GetFreezeConfig(symbol, strategyName, tradeType string) (*models.StrategyFreeze, error) {
	var freeze models.StrategyFreeze
	err := fs.orm.QueryTable("strategy_freeze").
		Filter("symbol", symbol).
		Filter("strategy_name", strategyName).
		Filter("trade_type", tradeType).
		One(&freeze)

	if err != nil {
		if err == orm.ErrNoRows {
			// 如果不存在，创建默认配置
			return fs.CreateDefaultFreezeConfig(symbol, strategyName, tradeType)
		}
		return nil, err
	}

	return &freeze, nil
}

// 创建默认冻结配置
func (fs *FreezeService) CreateDefaultFreezeConfig(symbol, strategyName, tradeType string) (*models.StrategyFreeze, error) {
	now := time.Now().Unix()
	freeze := &models.StrategyFreeze{
		Symbol:            symbol,
		StrategyName:      strategyName,
		TradeType:         tradeType,
		FreezeUntil:       0,
		LossCount:         0,
		FreezeOnLossCount: 5,  // 默认5次亏损后冻结
		FreezeHours:       24, // 默认冻结24小时
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	id, err := fs.orm.Insert(freeze)
	if err != nil {
		return nil, err
	}

	freeze.ID = id
	return freeze, nil
}

// 记录盈利（清零亏损次数）
func (fs *FreezeService) RecordProfit(symbol, strategyName, tradeType string) error {
	freeze, err := fs.GetFreezeConfig(symbol, strategyName, tradeType)
	if err != nil {
		return err
	}

	freeze.LossCount = 0
	freeze.UpdatedAt = time.Now().Unix()

	_, err = fs.orm.Update(freeze, "loss_count", "updated_at")
	if err != nil {
		logs.Error("更新盈利状态失败:", err)
		return err
	}

	logs.Info("策略盈利，清零亏损次数: %s-%s-%s", symbol, strategyName, tradeType)
	return nil
}

// 记录亏损（增加亏损次数，检查是否需要冻结）
func (fs *FreezeService) RecordLoss(symbol, strategyName, tradeType string) error {
	freeze, err := fs.GetFreezeConfig(symbol, strategyName, tradeType)
	if err != nil {
		return err
	}

	freeze.LossCount++
	freeze.UpdatedAt = time.Now().Unix()

	// 检查是否达到冻结条件
	if freeze.LossCount >= freeze.FreezeOnLossCount {
		// 设置冻结时间
		freeze.FreezeUntil = time.Now().Add(time.Duration(freeze.FreezeHours) * time.Hour).Unix()
		logs.Info("策略达到冻结条件，开始冻结: %s-%s-%s, 亏损次数: %d, 冻结至: %d",
			symbol, strategyName, tradeType, freeze.LossCount, freeze.FreezeUntil)

		_, err = fs.orm.Update(freeze, "loss_count", "freeze_until", "updated_at")
	} else {
		logs.Info("策略亏损次数+1: %s-%s-%s, 当前亏损次数: %d/%d",
			symbol, strategyName, tradeType, freeze.LossCount, freeze.FreezeOnLossCount)

		_, err = fs.orm.Update(freeze, "loss_count", "updated_at")
	}

	if err != nil {
		logs.Error("更新亏损状态失败:", err)
		return err
	}

	return nil
}

// 更新冻结配置
func (fs *FreezeService) UpdateFreezeConfig(freeze *models.StrategyFreeze) error {
	freeze.UpdatedAt = time.Now().Unix()
	_, err := fs.orm.Update(freeze)
	if err != nil {
		logs.Error("更新冻结配置失败:", err)
		return err
	}
	return nil
}

// 获取所有冻结配置
func (fs *FreezeService) GetAllFreezeConfigs(page, pageSize int) ([]models.StrategyFreeze, int64, error) {
	var freezes []models.StrategyFreeze
	qs := fs.orm.QueryTable("strategy_freeze")

	// 获取总数
	total, err := qs.Count()
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	_, err = qs.OrderBy("-updated_at").Limit(pageSize, offset).All(&freezes)
	if err != nil {
		return nil, 0, err
	}

	return freezes, total, nil
}

// 获取剩余冻结时间（秒）
func (fs *FreezeService) GetRemainingFreezeTime(symbol, strategyName, tradeType string) int64 {
	freeze, err := fs.GetFreezeConfig(symbol, strategyName, tradeType)
	if err != nil {
		return 0
	}

	now := time.Now().Unix()
	if freeze.FreezeUntil > now {
		return freeze.FreezeUntil - now
	}

	return 0
}

// 手动解除冻结
func (fs *FreezeService) UnfreezeManually(symbol, strategyName, tradeType string) error {
	freeze, err := fs.GetFreezeConfig(symbol, strategyName, tradeType)
	if err != nil {
		return err
	}

	freeze.FreezeUntil = 0
	freeze.UpdatedAt = time.Now().Unix()

	_, err = fs.orm.Update(freeze, "freeze_until", "updated_at")
	if err != nil {
		logs.Error("手动解除冻结失败:", err)
		return err
	}

	logs.Info("手动解除冻结: %s-%s-%s", symbol, strategyName, tradeType)
	return nil
}

// 重置亏损次数
func (fs *FreezeService) ResetLossCount(symbol, strategyName, tradeType string) error {
	freeze, err := fs.GetFreezeConfig(symbol, strategyName, tradeType)
	if err != nil {
		return err
	}

	freeze.LossCount = 0
	freeze.UpdatedAt = time.Now().Unix()

	_, err = fs.orm.Update(freeze, "loss_count", "updated_at")
	if err != nil {
		logs.Error("重置亏损次数失败:", err)
		return err
	}

	logs.Info("重置亏损次数: %s-%s-%s", symbol, strategyName, tradeType)
	return nil
}

// GetAllSymbols 获取所有唯一symbol
func (fs *FreezeService) GetAllSymbols() ([]string, error) {
	return fs.GetDistinctValues("symbol")
}

// GetAllStrategies 获取所有唯一strategy_name
func (fs *FreezeService) GetAllStrategies() ([]string, error) {
	return fs.GetDistinctValues("strategy_name")
}

// GetDistinctValues 用于通用distinct字段
func (fs *FreezeService) GetDistinctValues(field string) ([]string, error) {
	var result orm.ParamsList
	_, err := fs.orm.QueryTable("strategy_freeze").Distinct().ValuesFlat(&result, field)
	return result, err
}
