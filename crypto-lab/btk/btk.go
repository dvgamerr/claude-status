package btk

type Account struct {
	Account      any
	USDT         float64
	TodayPercent float64
	Traders      []map[string]interface{}
	Historys     []map[string]interface{}
}

func GetBalancer() (*Account, error) {
	ticker, err := GetMarketTicker("THB_USDT")
	if err != nil {
		return nil, err
	}
	bal, err := GetBalances()
	if err != nil {
		return nil, err
	}

	return &Account{
		Account: bal,
		USDT:    ticker.Last,
	}, nil
}
