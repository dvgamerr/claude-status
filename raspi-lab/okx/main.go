package okx

import (
	"errors"
	"fmt"
)

func GETAssetBalances() (map[string]interface{}, error) {
	// Rate Limit: 6 requests per second
	sugar.Debugln("GET /api/v5/asset/balances")
	var res ResponseAPI
	if err := Fetch("GET", "/api/v5/asset/balances", nil, &res); err != nil {
		return nil, err
	}

	if res.Code != "0" {
		return nil, errors.New(res.Msg)
	}
	if len(res.Data) == 0 {
		return map[string]interface{}{}, nil
	}

	return res.Data[0], nil
}

func GETAccountBalances() (map[string]interface{}, error) {
	// Rate Limit: 10 requests per 2 seconds
	sugar.Debugln("GET /api/v5/account/balance")
	var res ResponseAPI
	if err := Fetch("GET", "/api/v5/account/balance", nil, &res); err != nil {
		return nil, err
	}
	if res.Code != "0" {
		return nil, errors.New(res.Msg)
	}
	if len(res.Data) == 0 {
		return map[string]interface{}{}, nil
	}

	return res.Data[0], nil
}

func GETAssetBills(t int) ([]map[string]interface{}, error) {
	// Rate Limit: 6 Requests per second
	uriEndpoint := "/api/v5/asset/bills"
	if t != 0 {
		uriEndpoint = fmt.Sprintf("/api/v5/asset/bills?type=%d", t)
	}
	sugar.Debugf("GET %s", uriEndpoint)
	var res ResponseAPI
	if err := Fetch("GET", uriEndpoint, nil, &res); err != nil {
		return []map[string]interface{}{}, err
	}
	if res.Code != "0" {
		return []map[string]interface{}{}, errors.New(res.Msg)
	}

	return res.Data, nil
}

func GETFinanceSavingsBalance() ([]map[string]interface{}, error) {
	// Rate Limit: 6 requests per second
	sugar.Debugln("GET /api/v5/finance/savings/balance")
	var res ResponseAPI
	if err := Fetch("GET", "/api/v5/finance/savings/balance", nil, &res); err != nil {
		return []map[string]interface{}{}, err
	}
	if res.Code != "0" {
		return []map[string]interface{}{}, errors.New(res.Msg)
	}

	return res.Data, nil
}
func GETCopytradingCurrentLeadTraders() ([]map[string]interface{}, error) {
	// Rate limit: 5 requests per 2 seconds
	sugar.Debugf("GET /api/v5/copytrading/current-lead-traders")
	var res ResponseAPI
	if err := Fetch("GET", "/api/v5/copytrading/current-lead-traders", nil, &res); err != nil {
		return []map[string]interface{}{}, err
	}
	if res.Code != "0" {
		return []map[string]interface{}{}, errors.New(res.Msg)
	}

	return res.Data, nil
}
func GETAccountPositionsHistory() ([]map[string]interface{}, error) {
	// Rate Limit: 1 request per 10 seconds
	var res ResponseAPI
	if err := Fetch("GET", fmt.Sprintf("/api/v5/account/positions-history?before=%d", GetStartOfDate(0, 0, 0, -7).UnixMilli()), nil, &res); err != nil {
		return []map[string]interface{}{}, err
	}
	if res.Code != "0" {
		return []map[string]interface{}{}, errors.New(res.Msg)
	}

	return res.Data, nil
}
func GETAccountPositions() ([]map[string]interface{}, error) {
	// Rate Limit: 1 request per 10 seconds
	var res ResponseAPI
	if err := Fetch("GET", "/api/v5/account/positions", nil, &res); err != nil {
		return []map[string]interface{}{}, err
	}
	if res.Code != "0" {
		return []map[string]interface{}{}, errors.New(res.Msg)
	}

	return res.Data, nil
}
