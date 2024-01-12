package main

import (
	"fmt"
	"log"
	"time"

	"github.com/dvgamerr/aide-lab/raspi-lab/rpi"
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
	tview.Print(screen, rpiOs.CPUTemp, x, y+2, width, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, "GPU:", x+9, y+2, width, tview.AlignLeft, tcell.ColorNavy)
	tview.Print(screen, rpiOs.GPUTemp, x+14, y+2, width, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, fmt.Sprintf("%s (%s)", mem.Percent(), mem.UsedText()), x, y+3, width, tview.AlignLeft, tcell.ColorWhite)
	return 0, 0, 0, 0
}

// Define a draw function to draw a cross in the center of each cell.
func drawOSHeaderText(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	tview.Print(screen, "Time:", x-1, y, width, tview.AlignRight, tcell.ColorNavy)
	tview.Print(screen, "Uptime:", x-1, y+1, width, tview.AlignRight, tcell.ColorNavy)
	tview.Print(screen, "CPU:", x-1, y+2, width, tview.AlignRight, tcell.ColorNavy)
	tview.Print(screen, "MEM:", x-1, y+3, width, tview.AlignRight, tcell.ColorNavy)
	tview.Print(screen, "K8S:", x-1, y+4, width, tview.AlignRight, tcell.ColorNavy)
	tview.Print(screen, "DNS:", x-1, y+5, width, tview.AlignRight, tcell.ColorNavy)
	return 0, 0, 0, 0
}

var aNum accounting.Accounting

// Define a draw function to draw a cross in the center of each cell.
func drawOverviewHeader(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	okxTodayPnLText, okxTodayPnLColor := showUSD(okxAcc.TodayPnL, true)
	balanceName := "OKX Est"
	balanceText := aNum.FormatMoney(okxAcc.Total)
	fulfillText := aNum.FormatMoney(okxAcc.Fulfill)
	okxPnLText, okxPnLColor := showUSD(okxAcc.Total-okxAcc.Fulfill, true)
	posBalanceText := 11 - len(balanceText)

	tview.Print(screen, fmt.Sprintf("(Capital: %s)", fulfillText), x-2, y+1, width, tview.AlignRight, tcell.ColorWhite)

	tview.Print(screen, balanceName, x+5, y+1, width, tview.AlignLeft, tcell.ColorDarkSlateGray)
	tview.Print(screen, balanceText, x+6+len(balanceName)+posBalanceText, y+1, width, tview.AlignLeft, tcell.ColorWhite)

	totalPnLPos := x + 5 + len(balanceName) + len(fulfillText)
	tview.Print(screen, "[", totalPnLPos+posBalanceText, y+1, width, tview.AlignLeft, tcell.ColorGray)
	tview.Print(screen, okxPnLText, totalPnLPos+posBalanceText+2, y+1, width, tview.AlignLeft, okxPnLColor)
	tview.Print(screen, "]", totalPnLPos+len(okxPnLText)+posBalanceText+1, y+1, width, tview.AlignLeft, tcell.ColorGray)

	tview.Print(screen, "Today's PnL", x+1, y+2, width, tview.AlignLeft, tcell.ColorDarkSlateGray)
	tview.Print(screen, fmt.Sprintf("%s (%s)", okxTodayPnLText, showPercent(okxAcc.TodayPercent)), x+13+(len(balanceText)-len(okxTodayPnLText)), y+2, width, tview.AlignLeft, okxTodayPnLColor)

	balanceName = "BTK Est"
	btkFulfill := 300.0
	btkBalance := 302.0
	btkPnLText, btkPnLColor := showUSD(btkBalance-btkFulfill, true)
	balanceText = aNum.FormatMoney(btkBalance)
	posBalanceText = 11 - len(balanceText)
	tview.Print(screen, balanceName, x+5, y+3, width, tview.AlignLeft, tcell.ColorDarkSlateGray)
	tview.Print(screen, balanceText, x+6+len(balanceName)+posBalanceText, y+3, width, tview.AlignLeft, tcell.ColorWhite)

	totalPnLPos = x + 5 + len(balanceName) + len(btkPnLText)
	tview.Print(screen, "[", totalPnLPos+posBalanceText+1, y+3, width, tview.AlignLeft, tcell.ColorGray)
	tview.Print(screen, btkPnLText, totalPnLPos+posBalanceText+3, y+3, width, tview.AlignLeft, btkPnLColor)
	tview.Print(screen, "]", totalPnLPos+posBalanceText+len(btkPnLText)+2, y+3, width, tview.AlignLeft, tcell.ColorGray)

	return 0, 0, 0, 0
}

// ColorMaroon = 1
// ColorGreen = 2
// ColorOlive
// ColorNavy
// ColorPurple
// ColorTeal
// ColorSilver
// ColorGray
// ColorRed
// ColorLime
// ColorYellow
// ColorBlue
// ColorFuchsia
// ColorAqua
// ColorWhite

// Define a draw function to draw a cross in the center of each cell.
func drawTraderList(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	line := 2
	tview.Print(screen, " -------- COPY TRADERS -------- ", x, y, width, tview.AlignLeft, tcell.ColorAqua)
	for i, e := range okxAcc.Traders {
		pnl, _ := toFloat64(e["copyTotalPnl"])
		todayPnl, _ := toFloat64(e["todayPnl"])
		margin, _ := toFloat64(e["margin"])
		pnLText, _ := showUSD(pnl, true)
		todayPnlText, todayPnlColor := showUSD(todayPnl, false)
		pnlLen := 7 - len(todayPnlText)
		// copyLen := 7 - len(showMoney(copyTotalPnl))
		tview.Print(screen, pnLText, x-1, y+(i*line)+1, width, tview.AlignRight, tcell.ColorWhite)
		if margin == 0 {
			tview.Print(screen, e["nickName"].(string), x+1, y+(i*line)+1, width, tview.AlignLeft, tcell.ColorGray)
			tview.Print(screen, "Today:", x+2, y+(i*line)+2, width, tview.AlignLeft, tcell.ColorGray)
			tview.Print(screen, todayPnlText, x+2+6+pnlLen, y+(i*line)+2, width, tview.AlignLeft, todayPnlColor)
			tview.Print(screen, fmt.Sprintf("Inv. %s", showMoney(margin)), x+2+6+8, y+(i*line)+2, width, tview.AlignLeft, tcell.ColorGray)
		} else {
			tview.Print(screen, e["nickName"].(string), x+1, y+(i*line)+1, width, tview.AlignLeft, tcell.ColorOlive)
			tview.Print(screen, "Today:", x+2, y+(i*line)+2, width, tview.AlignLeft, tcell.ColorFuchsia)
			tview.Print(screen, todayPnlText, x+2+6+pnlLen, y+(i*line)+2, width, tview.AlignLeft, todayPnlColor)
			tview.Print(screen, fmt.Sprintf("Inv. %s", showMoney(margin)), x+2+6+8, y+(i*line)+2, width, tview.AlignLeft, tcell.ColorFuchsia)
		}

	}
	return 0, 0, 0, 0
}
