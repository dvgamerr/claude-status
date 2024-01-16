package main

import (
	"fmt"
	"os"
	"strconv"
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

func getUnixDate(y int, m int, d int) int64 {
	cur := time.Now()
	year, month, day := cur.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, cur.Location()).UnixMilli()
}

func parseUnixDate(utime any) string {
	i, err := strconv.ParseInt(utime.(string), 10, 64)
	if err != nil {
		panic(err)
	}

	return time.Unix(i/1000, 0).Format(time.RFC3339)
}

// {
//   "cTime": "1705376011966",
//   "ccy": "USDT",
//   "closeAvgPx": "19.3921176470588235",
//   "closeTotalPos": "17",
//   "direction": "short",
//   "fee": "-0.329806",
//   "fundingFee": "0",
//   "instId": "TIA-USDT-SWAP",
//   "instType": "SWAP",
//   "lever": "3.0",
//   "liqPenalty": "0",
//   "mgnMode": "isolated",
//   "openAvgPx": "19.4085882352941176",
//   "openMaxPos": "17",
//   "pnl": "0.2800000000000014",
//   "pnlRatio": "-0.0004528559218781",
//   "posId": "667330593805803531",
//   "realizedPnl": "-0.0498059999999986",
//   "triggerPx": "",
//   "type": "2",
//   "uTime": "1705376143323",
//   "uly": "TIA-USDT"
// }

func checkResponseOKX() {
	var err error
	var res okx.ResponseAPI
	// var endpoint = fmt.Sprintf("/api/v5/account/positions-history?instType=SWAP&before=%d", time.Date(year, month, day, 0, 0, 0, 0, cur.Location()).UnixMilli())
	var endpoint = fmt.Sprintf("/api/v5/account/positions-history?%sbefore=%d", "", getUnixDate(0, 0, 0))

	// var endpoint = fmt.Sprintf("/api/v5/copytrading/current-lead-traders%s", "")
	// Get Setting Copy Trading
	// var endpoint = fmt.Sprintf("/api/v5/asset/bills?type=117%s", "")
	if err = okx.Fetch("GET", endpoint, nil, &res); err != nil {
		sugar.Fatalw(err.Error())
	}
	var data string
	pnlTotal := 0.0
	if len(res.Data) > 0 {
		var showData interface{}
		for i, e := range res.Data {
			if showData == nil {
				showData = res.Data[i+1]
			}
			lever, _ := toFloat64(e["lever"])
			openAvgPx, _ := toFloat64(e["openAvgPx"])
			openMaxPos, _ := toFloat64(e["openMaxPos"])
			costs := 0.0
			if e["mgnMode"] == "isolated" {
				costs = openAvgPx * openMaxPos / lever
			}

			pnl, _ := toFloat64(e["realizedPnl"])
			pnlTotal += pnl
			fmt.Printf("%s [%s] direction: %s - %s | Held: %.2f -> %.2f\n", parseUnixDate(e["uTime"]), e["uly"], e["direction"], e["mgnMode"], costs, pnl)
		}
		sugar.Debugf("PnL: %.2f", pnlTotal)
		data, err = PrettyStruct(nil)
	} else {
		data, err = PrettyStruct(res)
	}
	if err != nil {
		sugar.Errorln(err.Error())
	}
	sugar.Debugf("Total: %d\n%s", len(res.Data), data)
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

	go setTickerInterval(3*time.Second, rpiOs.GetOSStats)()
	go setTickerInterval(okxIntervel, okxAcc.GetBalances)()
	go setTickerInterval(okxIntervel, okxAcc.GetTreaders)()

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
		AddItem(tview.NewBox().SetDrawFunc(drawTraderList), 1, 0, 1, 1, 0, 100, false).
		AddItem(primTextCenter("Orders"), 1, 1, 1, 1, 0, 100, false)

	go httpController()
	if err := app.SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
		panic(err)
	}

}
