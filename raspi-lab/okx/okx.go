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
	TotalTrade   float64
	TotalFund    float64
	TodayPnL     float64
	TodayPercent float64
	Traders      []map[string]interface{}
	Ongoing      []map[string]interface{}
	Historys     []map[string]interface{}
}

func (a *Account) WaitDone() {
	if a.wg != nil {
		a.wg.Done()
	}
}

func (a *Account) Initializer(zp *zap.SugaredLogger) {
	sugar = zp
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
	PnL         float64
	PnLRealized float64
	PnLPercent  float64
	Closed      float64
	Lever       float64
	Fee         float64
}

func CalcRealizedPnL(e map[string]interface{}) *RealizedPnL {
	lever, _ := ParseMoney(e["lever"])
	// closeAvgPx, _ := toFloat64(e["closeAvgPx"])
	// closeTotalPos, _ := toFloat64(e["closeTotalPos"])
	// openAvgPx, _ := toFloat64(e["openAvgPx"])
	// openMaxPos, _ := toFloat64(e["openMaxPos"])
	// mgnMode := e["mgnMode"].(string)
	// orderType := e["type"].(string)

	realizedPnl, _ := ParseMoney(e["realizedPnl"])
	pnl, _ := ParseMoney(e["upl"])
	if e["upl"] == nil {
		pnl, _ = ParseMoney(e["pnl"])
	}

	pnlRatio, _ := ParsePercent(e["uplRatio"])
	if e["uplRatio"] == nil {
		pnlRatio, _ = ParsePercent(e["pnlRatio"])
	}

	fee, _ := ParseMoney(e["fee"])
	fundingFee, _ := ParseMoney(e["fundingFee"])

	closed := pnl / (pnlRatio * 100) * 100
	// if closed < 0 {
	// 	calcPnL *= -1
	// }

	return &RealizedPnL{
		PnL:         pnl,
		PnLRealized: realizedPnl,
		Closed:      closed,
		Lever:       lever,
		PnLPercent:  pnlRatio * 100,
		Fee:         fee + fundingFee,
	}
}

func (a *Account) GetHistoryPositions() {
	defer a.WaitDone()
	var err error
	a.Ongoing, err = GETAccountPositions()
	if err != nil {
		sugar.Errorln(err)
		return
	}

	a.Historys, err = GETAccountPositionsHistory()
	if err != nil {
		sugar.Errorln(err)
		return
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
		sugar.Errorln(err)
		return
	}
}
func (a *Account) GetFulfill() {
	defer a.WaitDone()

	asset, err := GETAssetBills(117)
	if err != nil {
		sugar.Errorln(err)
		return
	}
	a.Fulfill = 0
	for _, e := range asset {
		bal, err := ParseMoney(e["bal"])
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
			sugar.Errorln(err)
			return
		}
		bWg.Done()
	}()
	go func() {
		account, err = GETAccountBalances()
		if err != nil {
			sugar.Errorln(err)
			return
		}
		bWg.Done()
	}()
	go func() {
		saving, err = GETFinanceSavingsBalance()
		if err != nil {
			sugar.Errorln(err)
			return
		}
		bWg.Done()
	}()
	bWg.Wait()

	a.TotalTrade = 0
	var bal float64
	if bal, err = ParseMoney(account["totalEq"]); err != nil {
		sugar.Errorln(err)
	}
	a.TotalTrade += bal

	a.TotalFund = 0
	if bal, err = ParseMoney(asset["bal"]); err != nil {
		sugar.Errorln(err)
	}
	a.TotalFund += bal

	for _, finn := range saving {
		if finn["ccy"] != "USDT" {
			continue
		}

		if bal, err = ParseMoney(finn["amt"]); err != nil {
			sugar.Errorln(err)
		}
		a.TotalFund += bal
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
