package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/leekchan/accounting"
	"github.com/rivo/tview"
	"go.uber.org/zap"
)

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
	if lab.Tty != "" {
		return filepath.Join(dirFile, filepath.Base(exeFile)+".log"), nil
	} else {
		return exeFile + ".log", nil
	}
}

func NewLogger(cli *Args) *zap.Logger {
	var cfg zap.Config
	if !lab.Fetch {
		logging, _ := getLoggingFilepath()

		cfg = zap.NewProductionConfig()
		cfg.OutputPaths = []string{logging}
		cfg.ErrorOutputPaths = []string{logging}
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.Level = zap.NewAtomicLevel()
		cfg.Level.SetLevel(zap.DebugLevel)
		cfg.OutputPaths = []string{"stdout"}
	}
	log, _ := cfg.Build()
	return log
}

// checkEnvVars checks that all specified environment variables are set and not empty.
func checkEnvVars(envs ...string) {
	for _, v := range envs {
		if os.Getenv(v) == "" {
			fmt.Printf("Error: %s environment variable is not set\n", v)
			os.Exit(1)
		}
	}
}

var symbolMoney string = "$" // "₮"
func showMoney(n float64) string {
	ac := accounting.Accounting{Symbol: symbolMoney, Precision: 2, Thousand: ","}
	return ac.FormatMoney(n)
}

func showUSD(n float64, showSymbol bool) (string, tcell.Color) {
	ac := accounting.Accounting{Symbol: symbolMoney, Precision: 2, Thousand: ",", Format: "+%s%v", FormatNegative: "-%s%v"}
	if !showSymbol {
		ac = accounting.Accounting{Symbol: "", Precision: 2, Thousand: ",", Format: "+%s%v", FormatNegative: "-%s%v"}
	}
	if n >= 0.0 {
		return ac.FormatMoney(n), tcell.ColorGreen
	} else {
		return ac.FormatMoney(n), tcell.ColorOrangeRed
	}
}
func showPercent(n float64) string {
	if n >= 0.0 {
		return "+" + fmt.Sprintf("%.2f%%", n)
	} else {
		return fmt.Sprintf("%.2f%%", n)
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

func primTextCenter(text string) tview.Primitive {
	return tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(text)
}
func PrettyStruct(data interface{}) (string, error) {
	val, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(val), nil
}
