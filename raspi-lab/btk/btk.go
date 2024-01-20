package btk

type Account struct {
	Fulfill      float64
	Total        float64
	TodayPnL     float64
	TodayPercent float64
	Traders      []map[string]interface{}
	Historys     []map[string]interface{}
}
