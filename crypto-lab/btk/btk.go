package btk

type Account struct {
	Fulfill      float64
	Total        float64
	Available    float64
	USDT         float64
	TodayPercent float64
	Traders      []map[string]interface{}
	Historys     []map[string]interface{}
}

func New() *Account {
	ticker := GetMarketTicker("THB_USDT")
	bal := GetBalances()

	return &Account{
		Fulfill:   (83700.0 - 29128.26),
		Total:     bal.Total,
		Available: bal.Available,
		USDT:      ticker.Last,
	}
}
