package btk

import (
	"go.uber.org/zap"
)

var sugar *zap.SugaredLogger

type Account struct {
	Fulfill      float64
	Total        float64
	TodayPnL     float64
	TodayPercent float64
	Traders      []map[string]interface{}
	Historys     []map[string]interface{}
}

func (a *Account) Initializer(zp *zap.SugaredLogger) {
	sugar = zp

	bal := GetBalances()
	a.Fulfill = (83700.0 - 29128.26)
	a.Total = bal.Total
}
