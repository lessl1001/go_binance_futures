package test

import (
	"go_binance_futures/models"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	. "github.com/smartystreets/goconvey/convey"
)

// 简化的FreezeService用于测试
type FreezeService struct {
	orm orm.Ormer
}

func NewFreezeService() *FreezeService {
	return &FreezeService{
		orm: orm.NewOrm(),
	}
}

func (fs *FreezeService) IsFrozen(symbol, strategyName, tradeType string) bool {
	var freeze models.StrategyFreeze
	err := fs.orm.QueryTable("strategy_freeze").
		Filter("symbol", symbol).
		Filter("strategy_name", strategyName).
		Filter("trade_type", tradeType).
		One(&freeze)
	
	if err != nil {
		return false
	}
	
	now := time.Now().Unix()
	return freeze.FreezeUntil > now
}

func (fs *FreezeService) GetFreezeConfig(symbol, strategyName, tradeType string) (*models.StrategyFreeze, error) {
	var freeze models.StrategyFreeze
	err := fs.orm.QueryTable("strategy_freeze").
		Filter("symbol", symbol).
		Filter("strategy_name", strategyName).
		Filter("trade_type", tradeType).
		One(&freeze)
	
	if err != nil {
		if err == orm.ErrNoRows {
			return fs.CreateDefaultFreezeConfig(symbol, strategyName, tradeType)
		}
		return nil, err
	}
	
	return &freeze, nil
}

func (fs *FreezeService) CreateDefaultFreezeConfig(symbol, strategyName, tradeType string) (*models.StrategyFreeze, error) {
	now := time.Now().Unix()
	freeze := &models.StrategyFreeze{
		Symbol:            symbol,
		StrategyName:      strategyName,
		TradeType:         tradeType,
		FreezeUntil:       0,
		LossCount:         0,
		FreezeOnLossCount: 5,
		FreezeHours:       24,
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

func (fs *FreezeService) RecordProfit(symbol, strategyName, tradeType string) error {
	freeze, err := fs.GetFreezeConfig(symbol, strategyName, tradeType)
	if err != nil {
		return err
	}
	
	freeze.LossCount = 0
	freeze.UpdatedAt = time.Now().Unix()
	
	_, err = fs.orm.Update(freeze, "loss_count", "updated_at")
	return err
}

func (fs *FreezeService) RecordLoss(symbol, strategyName, tradeType string) error {
	freeze, err := fs.GetFreezeConfig(symbol, strategyName, tradeType)
	if err != nil {
		return err
	}
	
	freeze.LossCount++
	freeze.UpdatedAt = time.Now().Unix()
	
	if freeze.LossCount >= freeze.FreezeOnLossCount {
		freeze.FreezeUntil = time.Now().Add(time.Duration(freeze.FreezeHours) * time.Hour).Unix()
		_, err = fs.orm.Update(freeze, "loss_count", "freeze_until", "updated_at")
	} else {
		_, err = fs.orm.Update(freeze, "loss_count", "updated_at")
	}
	
	return err
}

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

func init() {
	orm.RegisterDriver("sqlite", orm.DRSqlite)
	orm.RegisterDataBase("default", "sqlite3", ":memory:")
	orm.RegisterModel(new(models.StrategyFreeze))
	orm.RunSyncdb("default", false, true)
}

func TestFreezeService(t *testing.T) {
	Convey("测试风控服务", t, func() {
		freezeService := NewFreezeService()
		
		Convey("创建默认配置", func() {
			freeze, err := freezeService.CreateDefaultFreezeConfig("BTCUSDT", "line3_coin6", "real")
			So(err, ShouldBeNil)
			So(freeze, ShouldNotBeNil)
			So(freeze.Symbol, ShouldEqual, "BTCUSDT")
			So(freeze.StrategyName, ShouldEqual, "line3_coin6")
			So(freeze.TradeType, ShouldEqual, "real")
			So(freeze.FreezeOnLossCount, ShouldEqual, 5)
			So(freeze.FreezeHours, ShouldEqual, 24)
			So(freeze.LossCount, ShouldEqual, 0)
			So(freeze.FreezeUntil, ShouldEqual, 0)
		})
		
		Convey("检查冻结状态", func() {
			isFrozen := freezeService.IsFrozen("BTCUSDT", "line3_coin6", "real")
			So(isFrozen, ShouldBeFalse)
		})
		
		Convey("记录盈利", func() {
			err := freezeService.RecordProfit("BTCUSDT", "line3_coin6", "real")
			So(err, ShouldBeNil)
			
			freeze, err := freezeService.GetFreezeConfig("BTCUSDT", "line3_coin6", "real")
			So(err, ShouldBeNil)
			So(freeze.LossCount, ShouldEqual, 0)
		})
		
		Convey("记录亏损", func() {
			for i := 0; i < 4; i++ {
				err := freezeService.RecordLoss("BTCUSDT", "line3_coin6", "real")
				So(err, ShouldBeNil)
			}
			
			freeze, err := freezeService.GetFreezeConfig("BTCUSDT", "line3_coin6", "real")
			So(err, ShouldBeNil)
			So(freeze.LossCount, ShouldEqual, 4)
			So(freeze.FreezeUntil, ShouldEqual, 0)
			
			isFrozen := freezeService.IsFrozen("BTCUSDT", "line3_coin6", "real")
			So(isFrozen, ShouldBeFalse)
		})
		
		Convey("达到冻结条件", func() {
			err := freezeService.RecordLoss("BTCUSDT", "line3_coin6", "real")
			So(err, ShouldBeNil)
			
			freeze, err := freezeService.GetFreezeConfig("BTCUSDT", "line3_coin6", "real")
			So(err, ShouldBeNil)
			So(freeze.LossCount, ShouldEqual, 5)
			So(freeze.FreezeUntil, ShouldBeGreaterThan, time.Now().Unix())
			
			isFrozen := freezeService.IsFrozen("BTCUSDT", "line3_coin6", "real")
			So(isFrozen, ShouldBeTrue)
			
			remainingTime := freezeService.GetRemainingFreezeTime("BTCUSDT", "line3_coin6", "real")
			So(remainingTime, ShouldBeGreaterThan, 0)
		})
		
		Convey("测试不同交易类型", func() {
			testFrozen := freezeService.IsFrozen("BTCUSDT", "test_strategy", "test")
			So(testFrozen, ShouldBeFalse)
			
			for i := 0; i < 5; i++ {
				err := freezeService.RecordLoss("BTCUSDT", "test_strategy", "test")
				So(err, ShouldBeNil)
			}
			
			testFrozen = freezeService.IsFrozen("BTCUSDT", "test_strategy", "test")
			So(testFrozen, ShouldBeTrue)
			
			realFrozen := freezeService.IsFrozen("BTCUSDT", "line3_coin6", "real")
			So(realFrozen, ShouldBeTrue) // 这个应该还是冻结状态
		})
	})
}