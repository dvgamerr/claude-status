package okx

import "errors"

func GETAssetBalances() (map[string]interface{}, error) {
	// Rate Limit: 6 requests per second
	var res ResponseAPI
	if err := Fetch("GET", "/api/v5/asset/balances", nil, &res); err != nil {
		return nil, err
	}

	if res.Code != "0" {
		return nil, errors.New(res.Msg)
	}

	return res.Data[0], nil
}

func GETAccountBalances() (map[string]interface{}, error) {
	// Rate Limit: 10 requests per 2 seconds
	var res ResponseAPI
	if err := Fetch("GET", "/api/v5/account/balance", nil, &res); err != nil {
		return nil, err
	}
	if res.Code != "0" {
		return nil, errors.New(res.Msg)
	}

	return res.Data[0], nil
}

func GETFinanceSavingsBalance() ([]map[string]interface{}, error) {
	// Rate Limit: 6 requests per second
	var res ResponseAPI
	if err := Fetch("GET", "/api/v5/finance/savings/balance", nil, &res); err != nil {
		return []map[string]interface{}{}, err
	}
	if res.Code != "0" {
		return []map[string]interface{}{}, errors.New(res.Msg)
	}

	return res.Data, nil
}
