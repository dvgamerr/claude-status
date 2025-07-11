package btk

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
)

var stdJson = jsoniter.ConfigCompatibleWithStandardLibrary

type BitkubBalances struct {
	Total      float64
	Available  float64
	InOrder    float64
	InWithdraw float64
	Coins      map[string]Balance
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

func GetBalances() (*BitkubBalances, error) {
	var result ResponseAPI

	log.Debug().Msg("POST /v3/market/balances")
	if err := FetchSecure("POST", "/v3/market/balances", nil, &result); err != nil {
		return nil, err
	}

	byteData, err := stdJson.Marshal(result.Result)
	if err != nil {
		return nil, fmt.Errorf("marshaling: %w", err)
	}

	data := &BitkubBalances{
		Total:     0,
		Available: 0,
		Coins:     map[string]Balance{},
	}

	if err = stdJson.Unmarshal(byteData, &data.Coins); err != nil {
		return nil, fmt.Errorf("unmarshaling: %w", err)
	}
	for ccy, coin := range data.Coins {
		if coin.Available == 0 && coin.Reserved == 0 {
			continue
		}

		rate := 1.0
		if ccy != "THB" {
			ticker, err := GetMarketTicker(fmt.Sprintf("THB_%s", ccy))
			if err != nil {
				return nil, fmt.Errorf("market ticker for %s: %w", ccy, err)
			}
			rate = ticker.Last

			data.InOrder += coin.Available * rate
		} else {
			data.Available += coin.Available
		}
		data.Total += (coin.Available + coin.Reserved) * rate
	}

	log.Debug().Interface("data", data).Msg("Response")
	return data, nil
}

func GetMarketTicker(symbol string) (*Ticker, error) {
	var res map[string]Ticker
	url := fmt.Sprintf("/market/ticker?sym=%s", symbol)
	log.Debug().Msgf("GET %s", url)
	if err := FetchNonSecure("GET", url, nil, &res); err != nil {
		return nil, err
	}

	log.Debug().Msgf("Response: %#v\n", res[symbol])
	ticker := res[symbol]
	return &ticker, nil
}
