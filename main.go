package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/fatih/color"
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
const osInterval = 3 * time.Second

var (
	app *tview.Application
)
var (
	cpuTemp string = "n/a"
	gpuTemp string = "n/a"
)

func osTimer() {
	tick := time.NewTicker(osInterval)
	for {
		if _, ok := <-tick.C; !ok {
			break
		}
		cpuTemp = CPUTemp()
		gpuTemp = GPUTemp()
	}
}

// Define a draw function to draw a cross in the center of each cell.
func drawHeader(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	// mem, _ := MemoryInfo()
	// if err != nil {
	// 	log.Fatal("Error:", err)
	// }
	// fmt.Printf("GPU --- Temp: %s Memory: %s\n",
	// 	high(GPUTemp()),
	// 	high(GPUMemory()),
	// )
	// fmt.Printf("Total Memory: %v bytes\n", mem.Total)
	// fmt.Printf("Used Memory: %v bytes\n", mem.Used)

	uptime, _ := GetUptime()
	w, h := screen.Size()
	tview.Print(screen, fmt.Sprintf("(%dx%dpx) Time:", w, h), x-15, y, width, tview.AlignRight, tcell.ColorGray)
	tview.Print(screen, "Time:", x-15, y, width, tview.AlignRight, tcell.ColorTeal)
	tview.Print(screen, "Uptime:", x-15, y+1, width, tview.AlignRight, tcell.ColorTeal)
	tview.Print(screen, "GPU:", x-15, y+2, width, tview.AlignRight, tcell.ColorTeal)
	tview.Print(screen, "CPU:", x-15, y+3, width, tview.AlignRight, tcell.ColorTeal)
	tview.Print(screen, "MEM:", x-15, y+4, width, tview.AlignRight, tcell.ColorTeal)
	tview.Print(screen, time.Now().Format("15:04:05"), x+86, y, width, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, uptime, x+86, y+1, width, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, cpuTemp, x+86, y+2, width, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, gpuTemp, x+86, y+3, width, tview.AlignLeft, tcell.ColorWhite)

	// tview.Print(screen, fmt.Sprintf("CPU: %s (%sV)", CPUTemp(), CPUCoreVolt()), x, y+1, width, tview.AlignRight, tcell.ColorWhite)
	// tview.Print(screen, fmt.Sprintf("MEM: %s (%s)", mem.Percent(), mem.UsedText()), x, y+3, width, tview.AlignRight, tcell.ColorWhite)
	// tview.Print(screen, fmt.Sprintf("GPU: %s MEM: %s", GPUTemp(), GPUMemory()), x, y+2, width, tview.AlignRight, tcell.ColorWhite)

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
	app = tview.NewApplication()
	// view = tview.NewBox().SetDrawFunc(drawTimer)
	// if err := app.SetRoot(view, true).Run(); err != nil {
	// 	panic(err)
	// }

	newPrimitive := func(text string) tview.Primitive {
		return tview.NewTextView().
			SetTextAlign(tview.AlignRight).
			SetText(text)
	}

	menu := newPrimitive("Menu")
	main := newPrimitive("Main content")
	header := tview.NewBox().SetDrawFunc(drawHeader)
	grid := tview.NewGrid().
		SetRows(5, 0).
		SetColumns(32, 0).
		AddItem(header, 0, 0, 1, 2, 0, 0, false)

	// Layout for screens narrower than 100 cells (menu and side bar are hidden).
	grid.AddItem(menu, 0, 0, 0, 0, 0, 0, false).
		AddItem(main, 1, 0, 1, 2, 0, 0, false)

	// Layout for screens wider than 100 cells.
	grid.AddItem(menu, 1, 0, 1, 1, 0, 100, false).
		AddItem(main, 1, 1, 1, 1, 0, 100, false)

	go osTimer()
	go refreshTimer()
	if err := app.SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
		panic(err)
	}

	// if err := app.SetRoot(box, true).Run(); err != nil {
	// 	panic(err)
	// }
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
