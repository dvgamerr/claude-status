package btk

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
)

var stdJson = jsoniter.ConfigCompatibleWithStandardLibrary

type BitkubBalances struct {
	Total float64
	Coins map[string]Balance
}

type Balance struct {
	Available float64 `json:"available"`
	Reserved  float64 `json:"reserved"`
}

type Ticker struct {
	ID            int     `json:"id"`
	Last          float64 `json:"last"`
	LowestAsk     float64 `json:"lowestAsk"`
	HighestBid    float64 `json:"highestBid"`
	PercentChange float64 `json:"percentChange"`
	BaseVolume    float64 `json:"baseVolume"`
	QuoteVolume   float64 `json:"quoteVolume"`
	IsFrozen      int     `json:"isFrozen"`
	High24hr      float64 `json:"high24hr"`
	Low24hr       float64 `json:"low24hr"`
	Change        float64 `json:"change"`
	PrevClose     float64 `json:"prevClose"`
	PrevOpen      float64 `json:"prevOpen"`
}

func GetBalances() BitkubBalances {
	var result ResponseAPI

	if err := FetchSecure("POST", "/v3/market/balances", nil, &result); err != nil {
		sugar.Error(err)
	}

	byteData, err := stdJson.Marshal(result.Result)
	if err != nil {
		sugar.Errorln("Error marshaling:", err)
	}

	data := BitkubBalances{
		Total: 0,
		Coins: map[string]Balance{},
	}

	if err = stdJson.Unmarshal(byteData, &data.Coins); err != nil {
		sugar.Errorln("Error unmarshaling:", err)
	}

	data.Total = data.Coins["THB"].Available + data.Coins["THB"].Reserved

	return data
}

func GetMarketTicker(symbol string) Ticker {
	var res map[string]Ticker
	if err := FetchNonSecure("GET", fmt.Sprintf("/market/ticker?sym=%s", symbol), nil, &res); err != nil {
		sugar.Errorln(err)
	}

	return res[symbol]
}
