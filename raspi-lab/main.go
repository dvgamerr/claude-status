package main

import (
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/dvgamerr/aide-lab/raspi-lab/btk"
	"github.com/dvgamerr/aide-lab/raspi-lab/okx"
	"github.com/dvgamerr/aide-lab/raspi-lab/rpi"
	"github.com/dvgamerr/aide-lab/raspi-lab/sys"
	"github.com/joho/godotenv"
	"github.com/leekchan/accounting"
	"github.com/rivo/tview"
	"go.uber.org/zap"
)

type Args struct {
	Tty     string `arg:"--tty" help:"tty source"`
	Develop bool   `arg:"--dev" help:"develop coding test"`
	DB      bool   `arg:"--db" help:"save history to database"`
}

var (
	lab    Args
	logger *zap.Logger
	sugar  *zap.SugaredLogger
)

func init() {
	arg.MustParse(&lab)

	logger = NewLogger(&lab)
	sugar = logger.Sugar()
	aNum = accounting.Accounting{Symbol: symbolMoney, Precision: 2, Thousand: ",", Format: "%s%v"}

	// Load environment variables from .env
	if _, err := os.Stat(".env"); err != nil {
		logger.Fatal("error .env not found: " + err.Error())
	}

	if err := godotenv.Load(); err != nil {
		logger.Fatal(err.Error())
	}

	// Check that all required environment variables are set
	checkEnvVars("OKX_APIKEY", "OKX_SECRETKEY", "OKX_PASSPHRASE", "BTK_APIKEY", "BTK_SECRETKEY")
}

var (
	okxAcc okx.Account
	btkAcc btk.Account
	rpiOs  rpi.StatsOS
	stats  sys.StatsOnline
)

func main() {
	go httpController()

	defer sugar.Sync()
	app := tview.NewApplication()

	// if lab.Develop {
	// 	pinger, err := ping.NewPinger("10.203.1.202")
	// 	if runtime.GOOS == "windows" {
	// 		pinger.SetPrivileged(true)
	// 	}
	// 	if err != nil {
	// 		sugar.Panicln(err)
	// 	}
	// 	pinger.Timeout = 500 * time.Millisecond
	// 	pinger.Count = 3
	// 	// if err := pinger.Run(); err != nil { // Blocks until finished.
	// 	// 	sugar.Panicln(err)
	// 	// }
	// 	stats := pinger.Statistics()
	// 	fmt.Printf("%.1fms\n", float64(stats.AvgRtt)/float64(time.Millisecond))

	// 	// var res map[string]btk.Ticker
	// 	// if err := btk.FetchNonSecure("GET", "/market/ticker?sym=THB_USDT", nil, &res); err != nil {
	// 	// 	sugar.Errorln(err)
	// 	// }
	// 	// sugar.Debugf(" %+v\n", res["THB_USDT"].Last)

	// 	// bal := btk.GetBalances()
	// 	// fmt.Printf(" Total: %.2f\n", bal.Total)
	// 	// checkResponseOKX()
	// 	os.Exit(0)
	// }

	sugar.Infoln("OKX preplare initializing...")
	rpiOs.Initializer(sugar)
	stats.Initializer(sugar)
	btkAcc.Initializer(sugar)
	okxAcc.Initializer(sugar)

	okxIntervel := 3 * time.Second
	if os.Getenv("ENV") == "development" {
		okxIntervel = 10 * time.Second
	}

	sugar.Infoln("Ticker interval setting...")
	go setTickerInterval(500*time.Millisecond, func() { app.Draw() })()
	if lab.Tty != "" {
		go setTickerInterval(1*time.Second, stats.CheckAll)()
		go setTickerInterval(1*time.Second, rpiOs.GetOSStats)()
		go setTickerInterval(okxIntervel, okxAcc.GetBalances)()
		go setTickerInterval(okxIntervel, okxAcc.GetTreaders)()
		go setTickerInterval(okxIntervel, btkAcc.GetTotalBalance)()

		go setTickerInterval(5*time.Second, okxAcc.GetHistoryPositions)()
	}

	sugar.Infoln("Dashboard Renderer...")

	flex := tview.NewFlex().
		AddItem(tview.NewBox().SetDrawFunc(boxDrawOverview), 0, 1, false).
		AddItem(tview.NewBox().SetDrawFunc(boxDrawStatsLabel), 9, 0, false).
		AddItem(tview.NewBox().SetDrawFunc(boxDrawStatsValue), 22, 0, false)

	grid := tview.NewGrid().SetRows(6, 0).SetColumns(32, 0).AddItem(flex, 0, 0, 1, 2, 0, 0, false)

	// Layout for screens wider than 100 cells.
	grid.
		AddItem(tview.NewBox().SetDrawFunc(boxDrawCopyTrader), 1, 0, 1, 1, 0, 100, false).
		AddItem(tview.NewBox().SetDrawFunc(boxDrawOrderPosition), 1, 1, 1, 1, 0, 100, false)

	if err := app.SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
		panic(err)
	}

}
