package okx

import (
	"sync"

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
}

func (a *Account) WaitDone() {
	if a.wg != nil {
		a.wg.Done()
	}
}

func (a *Account) Initializer(s *zap.SugaredLogger) {
	sugar = s
	a.wg = &sync.WaitGroup{}
	a.wg.Add(3)

	go a.GetFulfill()
	go a.GetBalances()
	go a.GetTreaders()

	a.wg.Wait()
	a.wg = nil
}

func (a *Account) GetTreaders() {
	defer a.WaitDone()

	var err error
	a.Traders, err = GETCopytradingCurrentLeadTraders()
	if err != nil {
		sugar.Fatalln(err)
	}

	a.TodayPnL = 0
	todayMargin := 0.0
	for _, td := range a.Traders {
		pnl, _ := toFloat64(td["todayPnl"])
		margin, _ := toFloat64(td["margin"])

		if err != nil {
			sugar.Errorln(err)
		}
		a.TodayPnL += pnl
		todayMargin += margin
	}
	a.TodayPercent = a.TodayPnL * 100 / todayMargin
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
