package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/leekchan/accounting"
)

var symbolMoney = "$" // "₮"

func getLoggingFilepath() (string, error) {
	exeFile, err := os.Executable()
	if err != nil {
		return "", err
	}
	dirFile, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// Get the base name of the executable
	logging := filepath.Join(dirFile, filepath.Base(exeFile)+".log")
	if lab.Tty == "" {
		logging = exeFile + ".log"
	}
	return logging, nil
}

// checkEnvList checks that all specified environment variables are set and not empty.
func checkEnvList(envs ...string) {
	for _, v := range envs {
		if os.Getenv(v) == "" {
			fmt.Printf("Error: %s environment variable is not set\n", v)
			os.Exit(1)
		}
	}
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func showMoney(n float64) string {
	ac := accounting.Accounting{Symbol: symbolMoney, Precision: 2, Thousand: ","}
	return ac.FormatMoney(n)
}

func getAmountUsdtColor(n float64, showSymbol bool) (string, tcell.Color) {
	ac := accounting.Accounting{
		Symbol:         map[bool]string{true: symbolMoney, false: ""}[showSymbol],
		Precision:      2,
		Thousand:       ",",
		Format:         "+%s%v",
		FormatNegative: "-%s%v",
	}

	txt := ac.FormatMoney(n)
	color := tcell.ColorGray
	switch {
	case n > 0.0:
		color = tcell.ColorGreen
	case n < 0.0:
		color = tcell.ColorMaroon
	}

	return txt, color
}

func showPercent(n float64) string {
	if n >= 0.0 {
		return "+" + fmt.Sprintf("%.2f%%", n)
	}
	return fmt.Sprintf("%.2f%%", n)
}

func setTickerInterval(t time.Duration, run func()) func() {
	return func() {
		tick := time.NewTicker(t)
		defer tick.Stop()
		for range tick.C {
			run()
		}
	}
}

func PrettyStruct(data interface{}) (string, error) {
	val, err := json.MarshalIndent(data, "", "  ")
	return string(val), err
}
