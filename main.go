package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/dvgamerr/aide-lab/okx"
	"github.com/dvgamerr/aide-lab/rpi"
	"github.com/gdamore/tcell/v2"
	"github.com/joho/godotenv"
	"github.com/leekchan/accounting"
	"github.com/rivo/tview"
	"go.uber.org/zap"
)

type Args struct {
	Tty   string `arg:"--tty" help:"tty source"`
	Fetch bool   `atg:"-f,--fetch" help:"fetch api from okx test"`
	DB    bool   `atg:"--db" help:"save history to database"`
}

var (
	lab    Args
	logger *zap.Logger
	sugar  *zap.SugaredLogger
)

func getLoggingFilepath() (string, error) {
	// exeFile, err := os.Executable()
	// if err != nil {
	// 	return "", err
	// }
	dirFile, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Get the base name of the executable
	return filepath.Join(dirFile, "raspi-lab.log"), nil
}

func NewLogger(cli *Args) *zap.Logger {
	var cfg zap.Config
	if lab.Tty != "" {
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

func init() {
	arg.MustParse(&lab)

	logger = NewLogger(&lab)
	sugar = logger.Sugar()

	if lab.Tty == "" {
		symbolMoney = "₮"
	}
	aNum = accounting.Accounting{Symbol: symbolMoney, Precision: 2, Thousand: ",", Format: "%s%v"}

	executablePath, err := os.Executable()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	// Get the base name of the executable
	executableBase := filepath.Base(strings.ReplaceAll(executablePath, filepath.Ext(executablePath), ".log"))

	sugar.Infoln(filepath.Join(currentDir, executableBase))

	// Load environment variables from .env
	if _, err := os.Stat(".env"); err != nil {
		logger.Fatal("error .env not found: " + err.Error())
	}

	if err := godotenv.Load(); err != nil {
		logger.Fatal(err.Error())
	}

	// Check that all required environment variables are set
	checkEnvVars("OKX_APIKEY", "OKX_SECRETKEY", "OKX_PASSPHRASE")
}

var (
	app *tview.Application
)
var (
	cpuTemp string
	gpuTemp string
	cpuVolt string
)

func fetchStatsOS(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	cpuTemp = rpi.CPUTemp()
	cpuVolt = rpi.CPUCoreVolt()
	gpuTemp = rpi.GPUTemp()
}

var (
	okxFulfill  float64
	okxTotal    float64
	okxTodayPnL float64
	// okxOnceToday bool
)

func fetchOKXFulfill(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	var asset okx.ResponseAPI
	// Rate Limit: 6 Requests per second
	if err := okx.Fetch("GET", "/api/v5/asset/bills?type=117", nil, &asset); err != nil {
		sugar.Fatalw(err.Error())
	}
	okxFulfill = 0
	for _, e := range asset.Data {
		bal, err := toFloat64(e["bal"])
		if err != nil {
			sugar.Errorln(e["bal"], ":", err)
		}
		okxFulfill += bal
	}
}

func fetchOKXBalance(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}

	asset, err := okx.GETAssetBalances()
	if err != nil {
		sugar.Errorln(err)
	}

	account, err := okx.GETAccountBalances()
	if err != nil {
		sugar.Errorln(err)
	}

	saving, err := okx.GETFinanceSavingsBalance()
	if err != nil {
		sugar.Errorln(err)
	}

	okxTotal = 0
	var bal float64
	if asset["bal"] != nil {
		if bal, err = toFloat64(asset["bal"]); err != nil {
			sugar.Errorln(err)
		}
		okxTotal += bal
	}

	if account["totalEq"] != nil {
		if bal, err = toFloat64(account["totalEq"]); err != nil {
			sugar.Errorln(err)
		}
		okxTotal += bal
	}

	for _, finn := range saving {
		if finn["ccy"] != "USDT" {
			continue
		}

		if bal, err = toFloat64(finn["amt"]); err != nil {
			sugar.Errorln(err)
		}
		okxTotal += bal
	}

	traders, err := okx.GETCopytradingCurrentLeadTraders()
	if err != nil {
		sugar.Errorln(err)
	}

	okxTodayPnL = 0
	for _, td := range traders {
		pnl, err := toFloat64(td["todayPnl"])
		if err != nil {
			sugar.Errorln(err)
		}
		okxTodayPnL += pnl
	}

	// hours, minutes, _ := time.Now().Clock()
	// if hours == 0 && minutes == 0 || okxTodayPnL == 0 {
	// 	if !okxOnceToday {
	// 		okxTodayPnL = okxTotal
	// 		okxOnceToday = true
	// 	}
	// } else {
	// 	okxOnceToday = false
	// }
}

func main() {
	if lab.Fetch {
		cur := time.Now()
		year, month, day := cur.Date()

		var err error
		var res okx.ResponseAPI
		var endpoint = fmt.Sprintf("/api/v5/account/positions-history?before=%d", time.Date(year, month, day, 0, 0, 0, 0, cur.Location()).UnixMilli())
		if err = okx.Fetch("GET", endpoint, nil, &res); err != nil {
			sugar.Fatalw(err.Error())
		}
		var data string
		if len(res.Data) > 0 {
			data, err = PrettyStruct(res.Data[0])
		} else {
			data, err = PrettyStruct(res)
		}
		if err != nil {
			sugar.Errorln(err.Error())
		}
		sugar.Debugf("%s Total: %d\n%v\n", endpoint, len(res.Data), data)
		os.Exit(0)
	}

	defer sugar.Sync()
	sugar.Infoln("Preplare...")

	wg := sync.WaitGroup{}
	wg.Add(3)

	fetchOKXFulfill(&wg)
	fetchStatsOS(&wg)
	fetchOKXBalance(&wg)
	wg.Wait()

	sugar.Infoln("Render...")
	app = tview.NewApplication()

	flex := tview.NewFlex().
		AddItem(tview.NewBox().
			SetBorder(true).
			SetBorderColor(tcell.ColorGray).
			SetTitle("Dashboard").
			SetTitleAlign(tview.AlignLeft).
			SetDrawFunc(drawOverviewHeader), 0, 1, false).
		AddItem(tview.NewBox().SetDrawFunc(drawOSHeaderText), 10, 0, false).
		AddItem(tview.NewBox().SetDrawFunc(drawOSHeaderValue), 22, 0, false)
	grid := tview.NewGrid().
		SetRows(6, 0).
		SetColumns(32, 0).
		AddItem(flex, 0, 0, 1, 2, 0, 0, false)

	// Layout for screens wider than 100 cells.
	grid.AddItem(primTextCenter("Trader"), 1, 0, 1, 1, 0, 100, false).
		AddItem(primTextCenter("Orders"), 1, 1, 1, 1, 0, 100, false)

	go setTickerInterval(3*time.Second, func() { fetchStatsOS(nil) })()
	go setTickerInterval(500*time.Millisecond, func() { app.Draw() })()

	okxIntervel := 3 * time.Second
	if os.Getenv("ENV") == "development" {
		okxIntervel = 10 * time.Second
	}

	go setTickerInterval(okxIntervel, func() { fetchOKXBalance(nil) })()

	go httpController()
	if err := app.SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
		panic(err)
	}

}
