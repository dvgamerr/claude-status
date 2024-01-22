package btk

import (
	"go.uber.org/zap"
)

var sugar *zap.SugaredLogger

type Account struct {
	Fulfill      float64
	Total        float64
	Available    float64
	USDT         float64
	TodayPercent float64
	Traders      []map[string]interface{}
	Historys     []map[string]interface{}
}

func (a *Account) Initializer(zp *zap.SugaredLogger) {
	sugar = zp
	a.GetTotalBalance()
}

func (a *Account) GetTotalBalance() {
	ticker := GetMarketTicker("THB_USDT")
	bal := GetBalances()

	a.Fulfill = (83700.0 - 29128.26)
	a.Total = bal.Total
	a.Available = bal.Available
	a.USDT = ticker.Last
}
