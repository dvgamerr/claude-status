package okx

import (
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
)

var sugar *zap.SugaredLogger

type Account struct {
	wg           *sync.WaitGroup
	Fulfill      float64
	Total        float64
	TodayPnL     float64
	TodayPercent float64
	Traders      []map[string]interface{}
	Historys     []map[string]interface{}
}

func (a *Account) WaitDone() {
	if a.wg != nil {
		a.wg.Done()
	}
}

func (a *Account) Initializer(s *zap.SugaredLogger) {
	sugar = s
	a.wg = &sync.WaitGroup{}
	// Fix
	a.Fulfill = 2872.57
	go a.GetHistoryPositions()
	go a.GetBalances()
	go a.GetTreaders()
	a.wg.Add(3)

	a.wg.Wait()
	a.wg = nil
}

const UnitCrossType = 10
const YYYYMMDD = "2006-01-02"
const HHmm = "15:04"

type RealizedPnL struct {
	PnL        float64
	Closed     float64
	Lever      float64
	PnLPercent float64
}

func CalcRealizedPnL(e map[string]interface{}) *RealizedPnL {
	lever, _ := toFloat64(e["lever"])
	closeAvgPx, _ := toFloat64(e["closeAvgPx"])
	closeTotalPos, _ := toFloat64(e["closeTotalPos"])
	// openAvgPx, _ := toFloat64(e["openAvgPx"])
	// openMaxPos, _ := toFloat64(e["openMaxPos"])
	mgnMode := e["mgnMode"].(string)
	orderType := e["type"].(string)

	realizedPnl, _ := toFloat64(e["realizedPnl"])
	pnl, _ := toFloat64(e["pnl"])
	pnlRatio, _ := toFloat64(e["pnlRatio"])

	closed := closeTotalPos * closeAvgPx
	if orderType == "3" {

	} else if orderType == "2" {
		pnl *= -1
	}
	if mgnMode == "cross" && orderType == "1" {
		closed = closeTotalPos * closeAvgPx
	}

	// if mgnMode == "isolated" && orderType == "2" {
	// 	pnl *= -1
	// }

	// if mgnMode == "cross" {
	// 	// closed = closeTotalPos * closeAvgPx
	// 	// if orderType == "1" {
	// 	// 	closed = closeTotalPos * closeAvgPx
	// 	// }
	// }

	//   "closeAvgPx": "2.66",
	// "closeTotalPos": "38",
	return &RealizedPnL{
		PnL:        realizedPnl,
		Closed:     closed + pnl,
		Lever:      lever,
		PnLPercent: pnlRatio * 100,
	}
}

func (a *Account) GetHistoryPositions() {
	defer a.WaitDone()

	var err error
	a.Historys, err = GETAccountPositionsHistory()
	if err != nil {
		sugar.Fatalln(err)
	}

	a.TodayPnL = 0
	a.TodayPercent = 0
	totalClosed := 0.0
	startOfDay := GetStartOfDate(0, 0, 0, 0)
	for _, e := range a.Historys {
		if startOfDay.Sub(ParseUnixDate(e["uTime"])).Hours() > 0.0 {
			continue
		}
		pnl := CalcRealizedPnL(e)
		a.TodayPnL += pnl.PnL
		totalClosed += pnl.Closed
		// lever, _ := toFloat64(e["lever"])
		// closeAvgPx, _ := toFloat64(e["closeAvgPx"])
		// closeTotalPos, _ := toFloat64(e["closeTotalPos"])

		// pnl, _ := toFloat64(e["realizedPnl"])

		// closed := closeTotalPos / closeAvgPx
		// if e["mgnMode"] == "cross" {
		// 	closed = closeAvgPx * closeTotalPos
		// }
		// totalClosed += closed / lever
		// a.TodayPnL += pnl
	}
	a.TodayPercent += a.TodayPnL * 100 / totalClosed
}

func (a *Account) GetTreaders() {
	defer a.WaitDone()

	var err error
	a.Traders, err = GETCopytradingCurrentLeadTraders()
	if err != nil {
		sugar.Fatalln(err)
	}

	// a.TodayPnL = 0
	// todayPnl := 0.0
	// todayMargin := 0.0
	// for _, td := range a.Traders {
	// 	pnl, _ := toFloat64(td["todayPnl"])
	// 	margin, _ := toFloat64(td["margin"])

	// 	if err != nil {
	// 		sugar.Errorln(err)
	// 	}
	// 	a.TodayPnL += pnl
	// 	if margin > 0 {
	// 		todayPnl += pnl
	// 		todayMargin += margin
	// 	}
	// }
	// if todayMargin > 0 {
	// 	a.TodayPercent = todayPnl * 100 / todayMargin
	// }
}
func (a *Account) GetFulfill() {
	defer a.WaitDone()

	asset, err := GETAssetBills(117)
	if err != nil {
		sugar.Fatalln(err)
	}
	a.Fulfill = 0
	for _, e := range asset {
		bal, err := toFloat64(e["bal"])
		if err != nil {
			sugar.Errorln(e["bal"], ":", err)
		}
		a.Fulfill += bal
	}
}

func (a *Account) GetBalances() {
	defer a.WaitDone()
	bWg := sync.WaitGroup{}
	bWg.Add(3)

	var (
		err     error
		asset   map[string]interface{}
		account map[string]interface{}
		saving  []map[string]interface{}
	)

	go func() {
		asset, err = GETAssetBalances()
		if err != nil {
			sugar.Fatalln(err)
		}
		bWg.Done()
	}()
	go func() {
		account, err = GETAccountBalances()
		if err != nil {
			sugar.Fatalln(err)
		}
		bWg.Done()
	}()
	go func() {
		saving, err = GETFinanceSavingsBalance()
		if err != nil {
			sugar.Fatalln(err)
		}
		bWg.Done()
	}()
	bWg.Wait()

	a.Total = 0
	var bal float64
	if bal, err = toFloat64(asset["bal"]); err != nil {
		sugar.Errorln(err)
	}
	a.Total += bal

	if bal, err = toFloat64(account["totalEq"]); err != nil {
		sugar.Errorln(err)
	}
	a.Total += bal

	for _, finn := range saving {
		if finn["ccy"] != "USDT" {
			continue
		}

		if bal, err = toFloat64(finn["amt"]); err != nil {
			sugar.Errorln(err)
		}
		a.Total += bal
	}
}

func ParseUnixDate(utime any) time.Time {
	i, err := strconv.ParseInt(utime.(string), 10, 64)
	if err != nil {
		panic(err)
	}

	return time.Unix(i/1000, 0)
}

func GetStartOfDate(y int, m int, d int, a int) time.Time {
	cur := time.Now()
	year, month, day := cur.Add(time.Duration(a) * 24 * time.Hour).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, cur.Location())
}
