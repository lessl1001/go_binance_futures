// Boll Breakout Strategy - Simplified Optimized Version
package line

import (
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy"
	"go_binance_futures/types"
	"go_binance_futures/utils"
	"math"
	"strconv"
	"sync"
	"time"
)

// 策略参数配置
type BollParams struct {
	EmaShort      int
	EmaLong       int
	BollPeriod    int
	BollStdDev    float64
	RSIPeriod     int
	ATRPeriod     int
	MinProfit     float64
	RewardRatio   float64
	MaxDrawdown   float64
}

type TradeLineBoll struct {
	params      BollParams
	positionMtx sync.Mutex
}

func NewTradeLineBoll(params BollParams) *TradeLineBoll {
	if params.EmaShort == 0 { params.EmaShort = 7 }
	if params.EmaLong == 0 { params.EmaLong = 25 }
	if params.BollPeriod == 0 { params.BollPeriod = 20 }
	if params.BollStdDev == 0 { params.BollStdDev = 2 }
	if params.RSIPeriod == 0 { params.RSIPeriod = 14 }
	if params.ATRPeriod == 0 { params.ATRPeriod = 14 }
	if params.MinProfit == 0 { params.MinProfit = 0.003 }
	if params.RewardRatio == 0 { params.RewardRatio = 2 }
	if params.MaxDrawdown == 0 { params.MaxDrawdown = 0.05 }
	
	return &TradeLineBoll{params: params}
}

func (t *TradeLineBoll) GetCanLongOrShort(openParams strategy.OpenParams) strategy.OpenResult {
	result := strategy.OpenResult{}
	symbol := openParams.Symbols.Symbol

	// 并行获取K线数据
	var wg sync.WaitGroup
	var kline15m, kline1h []types.KLine
	var err15m, err1h error
	
	wg.Add(2)
	go func() {
		defer wg.Done()
		kline15m, err15m = binance.GetKlineData(symbol, "15m", 100)
	}()
	go func() {
		defer wg.Done()
		kline1h, err1h = binance.GetKlineData(symbol, "1h", 100)
	}()
	wg.Wait()
	
	if err15m != nil || err1h != nil {
		return result
	}
	
	if len(kline15m) < 50 || len(kline1h) < 50 {
		return result
	}
	
	closes15m := getCloses(kline15m)
	closes1h := getCloses(kline1h)
	volumes15m := getVolumes(kline15m)

	// 计算指标
	ema15mShort := utils.Ema(closes15m, t.params.EmaShort)
	ema15mLong := utils.Ema(closes15m, t.params.EmaLong)
	ema1hShort := utils.Ema(closes1h, t.params.EmaShort)
	ema1hLong := utils.Ema(closes1h, t.params.EmaLong)
	
	rsi15m := utils.Rsi(closes15m, t.params.RSIPeriod)
	boll15m := utils.Boll(closes15m, t.params.BollPeriod, t.params.BollStdDev)
	
	// 布林带宽度过滤
	bollWidth := (boll15m.Upper[0] - boll15m.Lower[0]) / boll15m.Mid[0]
	minBollWidth := 0.03
	
	// 成交量确认
	volumeAvg := utils.Sma(volumes15m, 5)
	volumeCondition := volumes15m[0] > volumeAvg[0]*1.2
	
	marketTrendOK := getBasicMarketTrend(closes1h) > 1.0 && 
		time.Since(binance.GetSystemStartTime()) > 10*time.Minute

	// 多头信号
	emaCrossLong := ema15mShort[0] > ema15mLong[0] && ema15mShort[1] < ema15mLong[1]
	bollBreakLong := closes15m[0] > boll15m.Upper[0] && bollWidth > minBollWidth
	rsiLongOK := rsi15m[0] > 40 && rsi15m[0] < 70
	atr := utils.GetAtr(symbol, "15m", t.params.ATRPeriod)
	volatilityOK := atr/closes15m[0] > 0.005
	
	result.CanLong = marketTrendOK && (emaCrossLong || bollBreakLong) && 
		ema1hShort[0] > ema1hLong[0] && 
		rsiLongOK && 
		volumeCondition && 
		volatilityOK

	// 空头信号
	emaCrossShort := ema15mShort[0] < ema15mLong[0] && ema15mShort[1] > ema15mLong[1]
	bollBreakShort := closes15m[0] < boll15m.Lower[0] && bollWidth > minBollWidth
	rsiShortOK := rsi15m[0] > 30 && rsi15m[0] < 60
	
	result.CanShort = marketTrendOK && (emaCrossShort || bollBreakShort) && 
		ema1hShort[0] < ema1hLong[0] && 
		rsiShortOK && 
		volumeCondition && 
		volatilityOK

	return result
}

func (t *TradeLineBoll) CanOrderComplete(closeParams strategy.CloseParams) bool {
	position := closeParams.Position
	
	// 最大回撤止损
	if closeParams.EquityDrawdown >= t.params.MaxDrawdown {
		return true
	}
	
	profitPercent := closeParams.NowProfit / position.EntryPrice
	targetProfit := t.params.MinProfit
	stopLoss := t.getStopLossPrice(position)
	
	// 动态止盈：风险回报比
	risk := math.Abs(position.EntryPrice - stopLoss)
	reward := risk * t.params.RewardRatio
	dynamicProfit := reward / position.EntryPrice

	// 达到任何止盈条件
	if profitPercent >= targetProfit || profitPercent >= dynamicProfit {
		// 获取3分钟线确认趋势
		kline3m, err := binance.GetKlineData(position.Symbol, "3m", 3)
		if err != nil {
			return false
		}
		
		closes := make([]float64, len(kline3m))
		for i, k := range kline3m {
			closes[i], _ = strconv.ParseFloat(k.Close, 64)
		}
		
		// 多头：价格下跌或低于短期EMA
		if position.Side == "LONG" {
			trend := utils.Ema(closes, 3)
			return closes[0] < trend[0] || closes[0] < closes[1]
		}
		
		// 空头：价格上涨或高于短期EMA
		if position.Side == "SHORT" {
			trend := utils.Ema(closes, 3)
			return closes[0] > trend[0] || closes[0] > closes[1]
		}
	}
	
	return false
}

func (t *TradeLineBoll) AutoStopOrder(closeParams strategy.CloseParams) bool {
	position := closeParams.Position
	
	// 获取当前价格
	kline15m, err := binance.GetKlineData(position.Symbol, "15m", 1)
	if err != nil {
		return false
	}
	currentPrice, _ := strconv.ParseFloat(kline15m[0].Close, 64)
	
	// 动态止损价格
	stopPrice := t.getStopLossPrice(position)
	
	// 基础止损
	if (position.Side == "LONG" && currentPrice <= stopPrice) || 
	   (position.Side == "SHORT" && currentPrice >= stopPrice) {
		return true
	}
	
	// 移动止损
	atr := utils.GetAtr(position.Symbol, "15m", t.params.ATRPeriod)
	
	if position.Side == "LONG" {
		profit := currentPrice - position.EntryPrice
		if profit > atr {
			newStop := position.EntryPrice + profit - atr*0.5
			return currentPrice < newStop
		}
	}
	
	if position.Side == "SHORT" {
		profit := position.EntryPrice - currentPrice
		if profit > atr {
			newStop := position.EntryPrice - profit + atr*0.5
			return currentPrice > newStop
		}
	}
	
	return false
}

// 获取动态止损价格
func (t *TradeLineBoll) getStopLossPrice(position *types.Position) float64 {
	atr := utils.GetAtr(position.Symbol, "15m", t.params.ATRPeriod)
	
	if position.Side == "LONG" {
		return position.EntryPrice - atr*1.5
	} else {
		return position.EntryPrice + atr*1.5
	}
}

// 市场趋势判断
func getBasicMarketTrend(closes []float64) float64 {
	if len(closes) < 200 {
		return 0
	}
	
	ema50 := utils.Ema(closes, 50)
	ema200 := utils.Ema(closes, 200)
	
	if ema50 > ema200 {
		return 2.0
	} else if ema50 < ema200 {
		return -2.0
	}
	return 0
}

func getCloses(kline []types.KLine) []float64 {
	r := make([]float64, len(kline))
	for i, v := range kline {
		f, _ := strconv.ParseFloat(v.Close, 64)
		r[i] = f
	}
	return r
}

func getVolumes(kline []types.KLine) []float64 {
	r := make([]float64, len(kline))
	for i, v := range kline {
		f, _ := strconv.ParseFloat(v.Volume, 64)
		r[i] = f
	}
	return r
}