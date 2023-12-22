package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/dvgamerr/aide-lab/okx"
	"github.com/fatih/color"
	"github.com/joho/godotenv"
)

func init() {
	// Load environment variables from .env
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		return
	}

	// Check that all required environment variables are set
	checkEnvVars("OKX_APIKEY", "OKX_SECRETKEY", "OKX_PASSPHRASE")
}

// checkEnvVars checks that all specified environment variables are set and not empty.
func checkEnvVars(vars ...string) {
	for _, v := range vars {
		if os.Getenv(v) == "" {
			fmt.Printf("Error: %s environment variable is not set\n", v)
			os.Exit(1)
		}
	}
}

func main() {
	var asset okx.ResponseAPI
	// Rate Limit: 6 Requests per second
	if err := okx.Fetch("GET", "/api/v5/asset/bills?type=117", nil, &asset); err != nil {
		fmt.Println(err)
	}
	fulfill := 0.0

	for _, e := range asset.Data {
		bal, err := toFloat64(e["bal"])
		if err != nil {
			fmt.Println(e["bal"], ":", err)
		}
		fulfill += bal
	}

	var account okx.ResponseAPI
	// Rate Limit: 10 requests per 2 seconds
	if err := okx.Fetch("GET", "/api/v5/account/balance", nil, &account); err != nil {
		fmt.Println(err)
	}
	if account.Code != "0" {
		fmt.Println(account.Msg)
	}

	totalEqual, err := toFloat64(account.Data[0]["totalEq"])
	if err != nil {
		fmt.Println(err)
	}

	bold := color.New(color.FgHiBlack).SprintFunc()
	info := color.New(color.FgHiBlue).SprintFunc()

	percent := (totalEqual * 100 / fulfill) - 100
	_, _, day := time.Now().Date()
	fmt.Printf("%s %s USD | %s (%s) %s",
		bold("OKX Total PnL:"),
		info(fmt.Sprintf("$%.2f", totalEqual)),
		printfProfit("$%.2f", totalEqual-fulfill),
		printfProfit("%.2f%%", percent),
		bold(fmt.Sprintf("%dDay", day)),
	)

	fmt.Println("")
	// var asset okx.ResponseAPI
	// if err := okx.Fetch("GET", fmt.Sprintf("/api/v5/asset/convert/history?after=%s&before=%s", toUnix("01-12-2023"), toUnix("")), nil, &asset); err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Printf("%#v\n", asset)
}

func printfProfit(f string, n float64) string {
	high := color.New(color.FgGreen).SprintFunc()
	low := color.New(color.FgRed).SprintFunc()
	if n == 0.0 {
		return fmt.Sprintf(f, n)
	} else if n > 0.0 {
		return high(fmt.Sprintf(f, n))
	} else {
		return low(fmt.Sprintf(f, n))
	}
}

// func toUnix(date string) string {
// 	ct := time.Now()
// 	if date != "" {
// 		ct, _ = time.Parse("02-01-2006", date)
// 	}

// 	// Convert to Unix timestamp in milliseconds
// 	return fmt.Sprintf("%d", ct.UnixNano()/int64(time.Millisecond))
// }

// Parse string to float64
func toFloat64(s interface{}) (float64, error) {
	f, err := strconv.ParseFloat(s.(string), 64)
	if err != nil {
		return 0, err
	}

	return math.Ceil(f*10000) / 10000, nil
}
