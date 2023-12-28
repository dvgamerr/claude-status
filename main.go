package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/dvgamerr/aide-lab/okx"
	"github.com/dvgamerr/aide-lab/rpi"
	"github.com/gdamore/tcell/v2"
	"github.com/joho/godotenv"
	"github.com/rivo/tview"
)

type Args struct {
	TTY   string `arg:"--tty" help:"tty source"`
	Fetch bool   `atg:"-f,--fetch" help:"fetch api from okx test"`
	DB    bool   `atg:"--db" help:"save history to database"`
}

var cli Args

func init() {
	// Load environment variables from .env
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file")
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
	okxFulfill   float64
	okxTotal     float64
	okxTodayPnL  float64
	okxOnceToday bool
)

func fetchOKXFulfill(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	var asset okx.ResponseAPI
	// Rate Limit: 6 Requests per second
	if err := okx.Fetch("GET", "/api/v5/asset/bills?type=117", nil, &asset); err != nil {
		fmt.Println(err)
	}
	okxFulfill = 0
	for _, e := range asset.Data {
		bal, err := toFloat64(e["bal"])
		if err != nil {
			fmt.Println(e["bal"], ":", err)
		}
		okxFulfill += bal
	}
}

func fetchOKXBalance(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	var account okx.ResponseAPI
	// Rate Limit: 10 requests per 2 seconds
	if err := okx.Fetch("GET", "/api/v5/account/balance", nil, &account); err != nil {
		fmt.Println(err)
	}
	if account.Code != "0" {
		fmt.Println(account.Msg)
	}

	var err error
	if okxTotal, err = toFloat64(account.Data[0]["totalEq"]); err != nil {
		fmt.Println(err)
	}
	hours, minutes, _ := time.Now().Clock()
	if hours == 0 && minutes == 0 || okxTodayPnL == 0 {
		if !okxOnceToday {
			okxTodayPnL = okxTotal
			okxOnceToday = true
		}
	} else {
		okxOnceToday = false
	}
	// _, _, day := time.Now().Date()
	// fmt.Printf("%s %s USD | %s (%s) %s",
	// 	bold("OKX Total PnL:"),
	// 	info(fmt.Sprintf("$%.2f", okxTotal)),
	// 	printfProfit("$%.2f", okxTotal-okxFulfill),
	// 	printfProfit("%.2f%%", percent),
	// 	bold(fmt.Sprintf("%dDay", day)),
	// )

}

func primTextCenter(text string) tview.Primitive {
	return tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(text)
}

func main() {

	arg.MustParse(&cli)
	if cli.Fetch {
		fmt.Println("")
		var asset okx.ResponseAPI
		if err := okx.Fetch("GET", "/api/v5/asset/convert/history", nil, &asset); err != nil {
			fmt.Println(err)
		}
		fmt.Printf("%#v\n", asset)
		os.Exit(0)
	}

	log.Println("Preplare...")

	wg := sync.WaitGroup{}
	wg.Add(3)

	fetchOKXFulfill(&wg)
	fetchStatsOS(&wg)
	fetchOKXBalance(&wg)
	wg.Wait()

	log.Println("Render...")
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

	if err := app.SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
		panic(err)
	}

}
