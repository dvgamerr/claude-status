package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/fatih/color"
)

// checkEnvVars checks that all specified environment variables are set and not empty.
func checkEnvVars(envs ...string) {
	for _, v := range envs {
		if os.Getenv(v) == "" {
			fmt.Printf("Error: %s environment variable is not set\n", v)
			os.Exit(1)
		}
	}
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

func setTickerInterval(t time.Duration, run func()) func() {
	return func() {
		tick := time.NewTicker(t)
		for {
			if _, ok := <-tick.C; !ok {
				break
			}
			run()
		}
	}
}
