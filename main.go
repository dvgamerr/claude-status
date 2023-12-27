package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/dvgamerr/aide-lab/okx"
	"github.com/dvgamerr/aide-lab/rpi"
	"github.com/gdamore/tcell/v2"
	"github.com/joho/godotenv"
	"github.com/rivo/tview"
)

type Args struct {
	TTY string `arg:"--tty" help:"tty source"`
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

func fetchStatsOS() {
	cpuTemp = rpi.CPUTemp()
	cpuVolt = rpi.CPUCoreVolt()
	gpuTemp = rpi.GPUTemp()
}

var (
	okxFulfill    float64
	okxTotal      float64
	okxPnlPercent float64
)

func fetchOKXData() {
	var asset okx.ResponseAPI
	var err error
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

	var account okx.ResponseAPI
	// Rate Limit: 10 requests per 2 seconds
	if err := okx.Fetch("GET", "/api/v5/account/balance", nil, &account); err != nil {
		fmt.Println(err)
	}
	if account.Code != "0" {
		fmt.Println(account.Msg)
	}

	if okxTotal, err = toFloat64(account.Data[0]["totalEq"]); err != nil {
		fmt.Println(err)
	}

	okxPnlPercent = (okxTotal * 100 / okxFulfill) - 100
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
	log.Println("Preplare...")
	fetchStatsOS()
	fetchOKXData()

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

	go setTickerInterval(3*time.Second, fetchStatsOS)()
	go setTickerInterval(500*time.Millisecond, func() { app.Draw() })()

	okxIntervel := 3 * time.Second
	if os.Getenv("ENV") == "development" {
		okxIntervel = 10 * time.Second
	}

	go setTickerInterval(okxIntervel, fetchOKXData)()

	if err := app.SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
		panic(err)
	}

	// fmt.Println("")
	// var asset okx.ResponseAPI
	// if err := okx.Fetch("GET", fmt.Sprintf("/api/v5/asset/convert/history?after=%s&before=%s", toUnix("01-12-2023"), toUnix("")), nil, &asset); err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Printf("%#v\n", asset)
}
