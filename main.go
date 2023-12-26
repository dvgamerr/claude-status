package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alexflint/go-arg"
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

// checkEnvVars checks that all specified environment variables are set and not empty.
func checkEnvVars(envs ...string) {
	for _, v := range envs {
		if os.Getenv(v) == "" {
			fmt.Printf("Error: %s environment variable is not set\n", v)
			os.Exit(1)
		}
	}
}

const refreshInterval = 500 * time.Millisecond
const osInterval = 3000 * time.Millisecond

var (
	app *tview.Application
)
var (
	cpuTemp string
	gpuTemp string
	cpuVolt string
)

func getAllStats() {
	cpuTemp = rpi.CPUTemp()
	cpuVolt = rpi.CPUCoreVolt()
	gpuTemp = rpi.GPUTemp()
}

func osTimer() {
	tick := time.NewTicker(osInterval)
	for {
		if _, ok := <-tick.C; !ok {
			break
		}
		getAllStats()
	}
}

// Define a draw function to draw a cross in the center of each cell.
func drawOSHeaderValue(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	mem, err := rpi.MemoryInfo()
	if err != nil {
		log.Fatal("Error:", err)
	}

	uptime, _ := rpi.GetUptime()
	tview.Print(screen, time.Now().Format("15:04:05"), x, y, width, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, uptime, x, y+1, width, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, gpuTemp, x, y+2, width, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, fmt.Sprintf("%s (%s)", cpuTemp, cpuVolt), x, y+3, width, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, fmt.Sprintf("%s (%s)", mem.Percent(), mem.UsedText()), x, y+4, width, tview.AlignLeft, tcell.ColorWhite)
	return 0, 0, 0, 0
}

// Define a draw function to draw a cross in the center of each cell.
func drawOSHeaderText(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	tview.Print(screen, "Time:", x-1, y, width, tview.AlignRight, tcell.ColorTeal)
	tview.Print(screen, "Uptime:", x-1, y+1, width, tview.AlignRight, tcell.ColorTeal)
	tview.Print(screen, "GPU:", x-1, y+2, width, tview.AlignRight, tcell.ColorTeal)
	tview.Print(screen, "CPU:", x-1, y+3, width, tview.AlignRight, tcell.ColorTeal)
	tview.Print(screen, "MEM:", x-1, y+4, width, tview.AlignRight, tcell.ColorTeal)
	return 0, 0, 0, 0
}

func refreshTimer() {
	tick := time.NewTicker(refreshInterval)
	for {
		if _, ok := <-tick.C; !ok {
			break
		}
		app.Draw()
	}
}

func main() {
	arg.MustParse(&cli)

	log.Println("Preplare...")
	getAllStats()

	app = tview.NewApplication()
	// view = tview.NewBox().SetDrawFunc(drawTimer)
	// if err := app.SetRoot(view, true).Run(); err != nil {
	// 	panic(err)
	// }

	primTextCenter := func(text string) tview.Primitive {
		return tview.NewTextView().
			SetTextAlign(tview.AlignRight).
			SetText(text)
	}
	menu := primTextCenter("Menu")
	// main := primTextCenter("Content")
	flex := tview.NewFlex().
		AddItem(tview.NewBox().SetBorder(true).SetTitle("Left (1/2 x width of Top)"), 0, 1, false).
		AddItem(tview.NewBox().SetDrawFunc(drawOSHeaderText), 22, 0, false).
		AddItem(tview.NewBox().SetDrawFunc(drawOSHeaderValue), 18, 0, false)
	grid := tview.NewGrid().
		SetRows(6, 0).
		SetColumns(32, 0).
		AddItem(flex, 0, 0, 1, 2, 0, 0, false)

	// Layout for screens wider than 100 cells.
	grid.AddItem(menu, 1, 0, 1, 1, 0, 100, false).
		AddItem(tview.NewBox().SetBorder(true).SetTitle("Content"), 1, 1, 1, 1, 0, 100, false)

	go osTimer()
	go refreshTimer()
	if err := app.SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
		panic(err)
	}

	// var asset okx.ResponseAPI
	// // Rate Limit: 6 Requests per second
	// if err := okx.Fetch("GET", "/api/v5/asset/bills?type=117", nil, &asset); err != nil {
	// 	fmt.Println(err)
	// }
	// fulfill := 0.0

	// for _, e := range asset.Data {
	// 	bal, err := toFloat64(e["bal"])
	// 	if err != nil {
	// 		fmt.Println(e["bal"], ":", err)
	// 	}
	// 	fulfill += bal
	// }

	// var account okx.ResponseAPI
	// // Rate Limit: 10 requests per 2 seconds
	// if err := okx.Fetch("GET", "/api/v5/account/balance", nil, &account); err != nil {
	// 	fmt.Println(err)
	// }
	// if account.Code != "0" {
	// 	fmt.Println(account.Msg)
	// }

	// totalEqual, err := toFloat64(account.Data[0]["totalEq"])
	// if err != nil {
	// 	fmt.Println(err)
	// }

	// percent := (totalEqual * 100 / fulfill) - 100
	// _, _, day := time.Now().Date()
	// fmt.Printf("%s %s USD | %s (%s) %s",
	// 	bold("OKX Total PnL:"),
	// 	info(fmt.Sprintf("$%.2f", totalEqual)),
	// 	printfProfit("$%.2f", totalEqual-fulfill),
	// 	printfProfit("%.2f%%", percent),
	// 	bold(fmt.Sprintf("%dDay", day)),
	// )

	fmt.Println("")
	// var asset okx.ResponseAPI
	// if err := okx.Fetch("GET", fmt.Sprintf("/api/v5/asset/convert/history?after=%s&before=%s", toUnix("01-12-2023"), toUnix("")), nil, &asset); err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Printf("%#v\n", asset)
}
