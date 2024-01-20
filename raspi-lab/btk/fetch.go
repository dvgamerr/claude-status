package btk

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const BASE_URL = "https://api.bitkub.com/api"

var (
	apiKey    string
	secretKey string
)

type ResponseAPI struct {
	Error  string                 `json:"error"`
	Result map[string]interface{} `json:"result"`
}
type ResponseAPIArray struct {
	Error  string                   `json:"error"`
	Result []map[string]interface{} `json:"result"`
}

func generateSignature(payload string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

func Fetch(method string, path string, reqBody interface{}, resPayload interface{}) error {

	if apiKey == "" || secretKey == "" {
		apiKey = os.Getenv("BTK_APIKEY")
		secretKey = os.Getenv("BTK_SECRETKEY")
	}

	var payload []byte = nil

	serverTime, err := getServerTime()
	if err != nil {
		return fmt.Errorf("server time: %+v", err)
	}

	if reqBody != nil {
		payload, err = json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling json: %+v", err)
		}
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", BASE_URL, path), bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("creating request: %+v", err)
	}

	// Generate timestamp and signature
	signaturePayload := fmt.Sprintf(`%s%s%s`, serverTime, req.Method, req.URL.Path)
	signature := generateSignature(signaturePayload)

	// Set the required headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BTK-TIMESTAMP", serverTime)
	req.Header.Set("X-BTK-APIKEY", apiKey)
	req.Header.Set("X-BTK-SIGN", signature)

	// Make the request
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("making request: %+v", err)
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&resPayload); err != nil {
		return fmt.Errorf("error decoding response: %+v", err)
	}

	return nil
}

func getServerTime() (string, error) {
	resp, err := http.Get("/servertime")
	if err != nil {
		return "0", err
	}
	defer resp.Body.Close()

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return "0", err
	}

	return string(result), nil
}

func GetBalances() {
	url := "/v3/market/balances"

	// Prepare the request payload
	// payloadBytes, err := json.Marshal(map[string]interface{}{
	// 	"sym": "THB_BTC",
	// })
	// if err != nil {
	// 	fmt.Println("Error marshaling JSON:", err)
	// 	return
	// }

	// Create the HTTP request , bytes.NewBuffer(payloadBytes)
	_, err := http.NewRequest("POST", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// for symbol, coin := range wallet.Result {
	// 	if coin <= 0 {
	// 		continue
	// 	}

	// 	fmt.Printf("- %s:%f\n", symbol, coin)
	// }

}

type BitkubBalance struct {
	Error  int `json:"error"`
	Result map[string]struct {
		Available float64 `json:"available"`
		Reserved  float64 `json:"reserved"`
	} `json:"result"`
}

type BitkubWallet struct {
	Error  int                `json:"error"`
	Result map[string]float64 `json:"result"`
}
