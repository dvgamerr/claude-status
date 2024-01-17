package main

import (
	"fmt"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/dvgamerr/aide-lab/raspi-lab/okx"
	"github.com/dvgamerr/aide-lab/raspi-lab/rpi"
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

func init() {
	arg.MustParse(&lab)

	logger = NewLogger(&lab)
	sugar = logger.Sugar()

	// if lab.Tty == "" {
	// 	symbolMoney = "₮"
	// }
	aNum = accounting.Accounting{Symbol: symbolMoney, Precision: 2, Thousand: ",", Format: "%s%v"}

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
	app    *tview.Application
	okxAcc okx.Account
	rpiOs  rpi.StatsOS
)

func checkResponseOKX() {

	var err error
	var res okx.ResponseAPI
	// var endpoint = fmt.Sprintf("/api/v5/account/positions-history?instType=SWAP&before=%d", time.Date(year, month, day, 0, 0, 0, 0, cur.Location()).UnixMilli())
	var endpoint = fmt.Sprintf("/api/v5/account/positions-history?before=%d", okx.GetStartOfDate(0, 0, 0, 0).UnixMilli())

	// var endpoint = fmt.Sprintf("/api/v5/copytrading/current-lead-traders%s", "")
	// Get Setting Copy Trading
	// var endpoint = fmt.Sprintf("/api/v5/asset/bills?type=117%s", "")
	if err = okx.Fetch("GET", endpoint, nil, &res); err != nil {
		sugar.Fatalw(err.Error())
	}
	var data string
	pnlTotal := 0.0
	closedTotal := 0.0
	if len(res.Data) > 0 {
		data, err = PrettyStruct(res.Data[2])
	} else {
		data, err = PrettyStruct(res)
	}
	if err != nil {
		sugar.Errorln(err.Error())
	}
	sugar.Debugf("Structure:\n%s", data)

	if len(res.Data) > 0 {
		var showData interface{}
		for i, e := range res.Data {
			if showData == nil {
				showData = res.Data[i]
			}
			pnl, closed := okx.CalcRealizedPnL(e)

			pnlTotal += pnl
			closedTotal += closed

			closeAvgPx, _ := toFloat64(e["closeAvgPx"])
			closeTotalPos, _ := toFloat64(e["closeTotalPos"])
			percent := pnl * 100 / closed

			fmt.Printf("%s [%s] PnL: %.2f (%.2f%%) Closed: %.2f\n", okx.ParseUnixDate(e["uTime"]).Format(okx.YYYYMMDD), e["uly"], pnl, percent, closed)
			fmt.Printf("           Lever: %s CloseTotalPos: %.2f CloseAvgPx: %.2f\n", e["lever"], closeTotalPos, closeAvgPx)
		}
		sugar.Debugf("Closed: %.2f - PnL: %.2f (%.2f%%)", closedTotal, pnlTotal, pnlTotal*100/closedTotal)
	}

}

func main() {
	if lab.Fetch {
		checkResponseOKX()
		os.Exit(0)
	}

	okxIntervel := 3 * time.Second
	if os.Getenv("ENV") == "development" {
		okxIntervel = 10 * time.Second
	}

	sugar.Infoln("OKX preplare initializing...")
	defer sugar.Sync()
	okxAcc.Initializer(sugar)
	rpiOs.Initializer(sugar)

	sugar.Infoln("Ticker interval setting...")
	go setTickerInterval(500*time.Millisecond, func() { app.Draw() })()

	go setTickerInterval(1*time.Second, rpiOs.GetOSStats)()

	go setTickerInterval(okxIntervel, okxAcc.GetBalances)()
	go setTickerInterval(okxIntervel, okxAcc.GetTreaders)()

	go setTickerInterval(11000*time.Millisecond, okxAcc.GetHistoryPositions)()

	sugar.Infoln("Dashboard Renderer...")
	app = tview.NewApplication()

	flex := tview.NewFlex().
		AddItem(tview.NewBox().SetDrawFunc(drawOverviewHeader), 0, 1, false).
		AddItem(tview.NewBox().SetDrawFunc(drawOSHeaderText), 10, 0, false).
		AddItem(tview.NewBox().SetDrawFunc(drawOSHeaderValue), 22, 0, false)
	grid := tview.NewGrid().
		SetRows(6, 0).
		SetColumns(32, 0).
		AddItem(flex, 0, 0, 1, 2, 0, 0, false)

	// Layout for screens wider than 100 cells.
	grid.
		AddItem(tview.NewBox().SetDrawFunc(drawCopyTrader), 1, 0, 1, 1, 0, 100, false).
		AddItem(tview.NewBox().SetDrawFunc(drawOrderPositionHistory), 1, 1, 1, 1, 0, 100, false)

	go httpController()
	if err := app.SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
		panic(err)
	}

}
