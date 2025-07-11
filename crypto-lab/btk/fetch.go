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
	Error   int    `json:"error"`
	Message string `json:"message"`
	Result  any    `json:"result"`
}

func generateSignature(payload string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

func FetchSecure(method string, path string, reqBody any, resPayload any) error {
	resp, err := fetch(true, method, path, reqBody)
	if err != nil {
		return fmt.Errorf("error decoding response: %+v", err)
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&resPayload); err != nil {
		return fmt.Errorf("error decoding response: %+v", err)
	}

	res := resPayload.(*ResponseAPI)

	if res.Error != 0 {
		return fmt.Errorf("error response: %+v", ErrorCode[res.Error])
	}

	return nil
}

func FetchNonSecure(method string, path string, reqBody any, resPayload any) error {
	resp, err := fetch(true, method, path, reqBody)
	if err != nil {
		return fmt.Errorf("error decoding response: %+v", err)
	}
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&resPayload); err != nil {
		return fmt.Errorf("error decoding response: %+v", err)
	}

	return nil
}

func fetch(secure bool, method string, path string, reqBody any) (*http.Response, error) {
	if secure && (apiKey == "" || secretKey == "") {
		apiKey = os.Getenv("BTK_APIKEY")
		secretKey = os.Getenv("BTK_SECRETKEY")
	}

	var payload []byte = nil

	serverTime, err := getServerTime()
	if err != nil {
		return nil, fmt.Errorf("server time: %+v", err)
	}

	if reqBody != nil {
		payload, err = json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("marshaling json: %+v", err)
		}
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", BASE_URL, path), bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("creating request: %+v", err)
	}

	// Generate timestamp and signature
	signaturePayload := fmt.Sprintf(`%s%s%s`, serverTime, req.Method, req.URL.Path)
	signature := generateSignature(signaturePayload)

	// Set the required headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if secure {
		req.Header.Set("X-BTK-TIMESTAMP", serverTime)
		req.Header.Set("X-BTK-APIKEY", apiKey)
		req.Header.Set("X-BTK-SIGN", signature)
	}

	// Make the request
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %+v", err)
	}

	return resp, nil
}

func getServerTime() (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s%s", BASE_URL, "/v3/servertime"))
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
