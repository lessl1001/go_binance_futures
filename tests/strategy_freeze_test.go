package test

import (
	"encoding/json"
	"go_binance_futures/models"
	"go_binance_futures/utils"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	// 初始化数据库连接
	orm.RegisterDriver("sqlite", orm.DRSqlite)
	orm.RegisterDataBase("default", "sqlite3", ":memory:")
	orm.RegisterModel(new(models.StrategyFreeze))
	orm.RunSyncdb("default", false, true)
}

func TestFreezeService(t *testing.T) {
	Convey("测试风控服务", t, func() {
		freezeService := utils.NewFreezeService()
		
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
			// 初始状态应该不被冻结
			isFrozen := freezeService.IsFrozen("BTCUSDT", "line3_coin6", "real")
			So(isFrozen, ShouldBeFalse)
		})
		
		Convey("记录盈利", func() {
			err := freezeService.RecordProfit("BTCUSDT", "line3_coin6", "real")
			So(err, ShouldBeNil)
			
			// 检查亏损次数是否被清零
			freeze, err := freezeService.GetFreezeConfig("BTCUSDT", "line3_coin6", "real")
			So(err, ShouldBeNil)
			So(freeze.LossCount, ShouldEqual, 0)
		})
		
		Convey("记录亏损", func() {
			// 记录4次亏损，不应该被冻结
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
			// 再记录一次亏损，应该被冻结
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
		
		Convey("手动解除冻结", func() {
			err := freezeService.UnfreezeManually("BTCUSDT", "line3_coin6", "real")
			So(err, ShouldBeNil)
			
			isFrozen := freezeService.IsFrozen("BTCUSDT", "line3_coin6", "real")
			So(isFrozen, ShouldBeFalse)
		})
		
		Convey("重置亏损次数", func() {
			err := freezeService.ResetLossCount("BTCUSDT", "line3_coin6", "real")
			So(err, ShouldBeNil)
			
			freeze, err := freezeService.GetFreezeConfig("BTCUSDT", "line3_coin6", "real")
			So(err, ShouldBeNil)
			So(freeze.LossCount, ShouldEqual, 0)
		})
		
		Convey("测试不同交易类型", func() {
			// 测试交易类型应该独立于实盘交易
			testFrozen := freezeService.IsFrozen("BTCUSDT", "test_strategy", "test")
			So(testFrozen, ShouldBeFalse)
			
			// 记录测试交易亏损
			for i := 0; i < 5; i++ {
				err := freezeService.RecordLoss("BTCUSDT", "test_strategy", "test")
				So(err, ShouldBeNil)
			}
			
			testFrozen = freezeService.IsFrozen("BTCUSDT", "test_strategy", "test")
			So(testFrozen, ShouldBeTrue)
			
			// 实盘交易应该不受影响
			realFrozen := freezeService.IsFrozen("BTCUSDT", "line3_coin6", "real")
			So(realFrozen, ShouldBeFalse)
		})
	})
}

func TestStrategyFreezeController(t *testing.T) {
	Convey("测试策略冻结控制器", t, func() {
		
		Convey("创建冻结配置", func() {
			reqBody := `{
				"symbol": "ETHUSDT",
				"strategy_name": "line3_coin6",
				"trade_type": "real",
				"freeze_on_loss_count": 3,
				"freeze_hours": 12
			}`
			
			r, _ := http.NewRequest("POST", "/strategy-freeze", strings.NewReader(reqBody))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			web.BeeApp.Handlers.ServeHTTP(w, r)
			
			So(w.Code, ShouldEqual, 200)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["code"], ShouldEqual, 200)
			So(response["message"], ShouldEqual, "创建成功")
		})
		
		Convey("获取冻结配置列表", func() {
			r, _ := http.NewRequest("GET", "/strategy-freeze", nil)
			w := httptest.NewRecorder()
			web.BeeApp.Handlers.ServeHTTP(w, r)
			
			So(w.Code, ShouldEqual, 200)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["code"], ShouldEqual, 200)
			
			data := response["data"].(map[string]interface{})
			So(data["total"], ShouldBeGreaterThan, 0)
		})
		
		Convey("获取单个冻结配置", func() {
			r, _ := http.NewRequest("GET", "/strategy-freeze/config?symbol=ETHUSDT&strategy_name=line3_coin6&trade_type=real", nil)
			w := httptest.NewRecorder()
			web.BeeApp.Handlers.ServeHTTP(w, r)
			
			So(w.Code, ShouldEqual, 200)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["code"], ShouldEqual, 200)
			
			data := response["data"].(map[string]interface{})
			So(data["is_frozen"], ShouldBeFalse)
			So(data["remaining_time"], ShouldEqual, 0)
		})
		
		Convey("手动解除冻结", func() {
			reqBody := `{
				"symbol": "ETHUSDT",
				"strategy_name": "line3_coin6",
				"trade_type": "real"
			}`
			
			r, _ := http.NewRequest("POST", "/strategy-freeze/unfreeze", strings.NewReader(reqBody))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			web.BeeApp.Handlers.ServeHTTP(w, r)
			
			So(w.Code, ShouldEqual, 200)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["code"], ShouldEqual, 200)
			So(response["message"], ShouldEqual, "解除冻结成功")
		})
		
		Convey("重置亏损次数", func() {
			reqBody := `{
				"symbol": "ETHUSDT",
				"strategy_name": "line3_coin6",
				"trade_type": "real"
			}`
			
			r, _ := http.NewRequest("POST", "/strategy-freeze/reset-loss", strings.NewReader(reqBody))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			web.BeeApp.Handlers.ServeHTTP(w, r)
			
			So(w.Code, ShouldEqual, 200)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response["code"], ShouldEqual, 200)
			So(response["message"], ShouldEqual, "重置亏损次数成功")
		})
	})
}