package okx

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

func Fetch(method string, path string, reqPayload interface{}, resPayload interface{}) error {
	var client *http.Client = http.DefaultClient
	var payload []byte = nil

	// Get current time in ISO format
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	// Create the HMAC SHA256 signature
	signature := generateSignature(timestamp+method+path, os.Getenv("OKX_SECRETKEY"))

	if reqPayload != nil {
		var err error
		if payload, err = json.Marshal(reqPayload); err != nil {
			return fmt.Errorf("marshaling json: %+v", err)
		}
	}

	req, err := http.NewRequest(method, "https://www.okex.com"+path, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("eror new request: %v", err)
	}

	// Set headers
	req.Header.Add("OK-ACCESS-KEY", os.Getenv("OKX_APIKEY"))
	req.Header.Add("OK-ACCESS-SIGN", signature)
	req.Header.Add("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Add("OK-ACCESS-PASSPHRASE", os.Getenv("OKX_PASSPHRASE"))

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("eror new client: %v", err)
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&resPayload); err != nil {
		return fmt.Errorf("error decoding response: %+v", err)
	}

	// if res.Code != "0" {
	// 	return fmt.Errorf("error http response: %+v", res.Msg)
	// }

	return nil
}

func generateSignature(message, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

type ResponseAPI struct {
	Code string                   `json:"code"`
	Data []map[string]interface{} `json:"data"`
	Msg  string                   `json:"msg"`
}
