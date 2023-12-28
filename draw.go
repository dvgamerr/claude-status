package main

import (
	"fmt"
	"log"
	"time"

	"github.com/dvgamerr/aide-lab/rpi"
	"github.com/gdamore/tcell/v2"
	"github.com/leekchan/accounting"
	"github.com/rivo/tview"
)

// type OSStats struct {
// 	CPUTemp string
// 	GPUTemp string
// 	CPUVolt string
// }

// Define a draw function to draw a cross in the center of each cell.
func drawOSHeaderValue(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	mem, err := rpi.MemoryInfo()
	if err != nil {
		log.Fatal("Error:", err)
	}

	w, h := screen.Size()
	uptime, _ := rpi.GetUptime()
	tview.Print(screen, time.Now().Format("15:04:05"), x, y, width, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, fmt.Sprintf("(%dx%d)", w, h), x+9, y, width, tview.AlignLeft, tcell.ColorGrey)
	tview.Print(screen, uptime, x, y+1, width, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, gpuTemp, x, y+2, width, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, fmt.Sprintf("%s (%s)", cpuTemp, cpuVolt), x, y+3, width, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, fmt.Sprintf("%s (%s)", mem.Percent(), mem.UsedText()), x, y+4, width, tview.AlignLeft, tcell.ColorWhite)
	return 0, 0, 0, 0
}

// Define a draw function to draw a cross in the center of each cell.
func drawOSHeaderText(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	tview.Print(screen, "Time:", x-1, y, width, tview.AlignRight, tcell.ColorBlue)
	tview.Print(screen, "Uptime:", x-1, y+1, width, tview.AlignRight, tcell.ColorBlue)
	tview.Print(screen, "GPU:", x-1, y+2, width, tview.AlignRight, tcell.ColorBlue)
	tview.Print(screen, "CPU:", x-1, y+3, width, tview.AlignRight, tcell.ColorBlue)
	tview.Print(screen, "MEM:", x-1, y+4, width, tview.AlignRight, tcell.ColorBlue)
	return 0, 0, 0, 0
}

var aNum accounting.Accounting = accounting.Accounting{Symbol: "₮", Precision: 2, Thousand: ",", Format: "%s%v"}

// Define a draw function to draw a cross in the center of each cell.
func drawOverviewHeader(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {

	okxTodayPnLText, okxTodayPnLColor := showUSD(okxTotal - okxTodayPnL)
	balanceName := "OKX Est:"
	balanceText := aNum.FormatMoney(okxTotal)
	fulfillText := aNum.FormatMoney(okxFulfill)
	okxPnLText, okxPnLColor := showUSD(okxTotal - okxFulfill)

	tview.Print(screen, fmt.Sprintf("(Capital: %s)", fulfillText), x-2, y+1, width, tview.AlignRight, tcell.ColorWhite)

	tview.Print(screen, balanceName, x+4, y+1, width, tview.AlignLeft, tcell.ColorBlue)
	tview.Print(screen, balanceText, x+5+len(balanceName), y+1, width, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, fmt.Sprintf("(%s)", okxPnLText), x+5+len(balanceName)+len(fulfillText)-1, y+1, width, tview.AlignLeft, okxPnLColor)

	tview.Print(screen, "Today's PnL", x+1, y+2, width, tview.AlignLeft, tcell.ColorBlue)
	tview.Print(screen, fmt.Sprintf("%s (%s)", okxTodayPnLText, showPercent(okxTotal*100/okxTodayPnL-100)), x+13+(len(balanceText)-len(okxTodayPnLText)), y+2, width, tview.AlignLeft, okxTodayPnLColor)

	// btkFulfill := 90000
	// btkPnLText, btkPnLColor := showUSD(1000)
	// tview.Print(screen, "BTK PnL:", x+vw, y+1, vw, tview.AlignLeft, tcell.ColorGreen)
	// tview.Print(screen, aNum.FormatMoney(btkFulfill), x+vw, y+1, vw, tview.AlignLeft, tcell.ColorWhite)
	// tview.Print(screen, fmt.Sprintf("(%s)", btkPnLText), x+vw, y+1, vw, tview.AlignLeft, btkPnLColor)

	return 0, 0, 0, 0
}
