package test

import (
	"go_binance_futures/models"
	"go_binance_futures/utils"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	// 确保数据库已初始化
	orm.RegisterDriver("sqlite", orm.DRSqlite)
	orm.RegisterDataBase("default", "sqlite3", ":memory:")
	orm.RegisterModel(new(models.StrategyFreeze))
	orm.RunSyncdb("default", false, true)
}

func TestOptionsWithEmptyDatabase(t *testing.T) {
	Convey("测试空数据库时的选项获取", t, func() {
		freezeService := utils.NewFreezeService()

		Convey("当数据库为空时，应该返回默认symbols", func() {
			// 确保数据库为空
			o := orm.NewOrm()
			o.Raw("DELETE FROM strategy_freeze").Exec()

			symbols, err := freezeService.GetAllSymbols()
			So(err, ShouldBeNil)
			
			// 应该返回默认币种
			So(len(symbols), ShouldEqual, 3)
			So(symbols, ShouldContain, "BTCUSDT")
			So(symbols, ShouldContain, "ETHUSDT")
			So(symbols, ShouldContain, "BNBUSDT")
		})

		Convey("当数据库为空时，应该返回默认strategies", func() {
			// 确保数据库为空
			o := orm.NewOrm()
			o.Raw("DELETE FROM strategy_freeze").Exec()

			strategies, err := freezeService.GetAllStrategies()
			So(err, ShouldBeNil)
			
			// 应该返回默认策略
			So(len(strategies), ShouldEqual, 2)
			So(strategies, ShouldContain, "line3_coin6")
			So(strategies, ShouldContain, "trend_follow")
		})

		Convey("当数据库有数据时，应该返回数据库中的值", func() {
			// 清空数据库
			o := orm.NewOrm()
			o.Raw("DELETE FROM strategy_freeze").Exec()

			// 添加一些测试数据
			_, err := freezeService.CreateDefaultFreezeConfig("ADAUSDT", "test_strategy", "real")
			So(err, ShouldBeNil)
			_, err = freezeService.CreateDefaultFreezeConfig("DOTUSDT", "another_strategy", "test")
			So(err, ShouldBeNil)

			symbols, err := freezeService.GetAllSymbols()
			So(err, ShouldBeNil)
			// 应该返回数据库中的值，而不是默认值
			So(len(symbols), ShouldEqual, 2)
			So(symbols, ShouldContain, "ADAUSDT")
			So(symbols, ShouldContain, "DOTUSDT")

			strategies, err := freezeService.GetAllStrategies()
			So(err, ShouldBeNil)
			So(len(strategies), ShouldEqual, 2)
			So(strategies, ShouldContain, "test_strategy")
			So(strategies, ShouldContain, "another_strategy")
		})
	})
}