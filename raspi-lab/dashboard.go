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
	tview.Print(screen, balanceText, x+4+len(balanceName)+posBalanceText, y+1, width, tview.AlignLeft, tcell.ColorWhite)

	totalPnLPos := x + 5 + len(balanceName) + len(fulfillText)
	tview.Print(screen, "[", totalPnLPos+posBalanceText, y+1, width, tview.AlignLeft, tcell.ColorGray)
	tview.Print(screen, okxPnLText, totalPnLPos+posBalanceText+2, y+1, width, tview.AlignLeft, okxPnLColor)
	tview.Print(screen, "]", totalPnLPos+len(okxPnLText)+posBalanceText+3, y+1, width, tview.AlignLeft, tcell.ColorGray)

	tview.Print(screen, "Today's PnL", x+1, y+2, width, tview.AlignLeft, tcell.ColorDarkSlateGray)
	tview.Print(screen, fmt.Sprintf("%s (%s)", okxTodayPnLText, showPercent(okxAcc.TodayPercent)), x+13+(len(balanceText)-len(okxTodayPnLText)), y+2, width, tview.AlignLeft, okxTodayPnLColor)

	balanceName = "BTK Est"
	btkFulfill := 300.0
	btkBalance := 302.0
	btkPnLText, btkPnLColor := showUSD(btkBalance-btkFulfill, true)
	balanceText = aNum.FormatMoney(btkBalance)
	posBalanceText = 11 - len(balanceText)
	tview.Print(screen, balanceName, x+5, y+3, width, tview.AlignLeft, tcell.ColorDarkSlateGray)
	tview.Print(screen, balanceText, x+4+len(balanceName)+posBalanceText, y+3, width, tview.AlignLeft, tcell.ColorWhite)

	totalPnLPos = x + 5 + len(balanceName) + len(btkPnLText)
	tview.Print(screen, "[", totalPnLPos+posBalanceText+1, y+3, width, tview.AlignLeft, tcell.ColorGray)
	tview.Print(screen, btkPnLText, totalPnLPos+posBalanceText+3, y+3, width, tview.AlignLeft, btkPnLColor)
	tview.Print(screen, "]", totalPnLPos+posBalanceText+len(btkPnLText)+4, y+3, width, tview.AlignLeft, tcell.ColorGray)

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
	lineHeight := 2
	header := "───────── COPY TRADERS ──────── "
	tview.Print(screen, header, x, y, width, tview.AlignLeft, tcell.ColorAqua)

	maxLenPnL := 0
	for _, trader := range okxAcc.Traders {
		pnl, _ := toFloat64(trader["copyTotalPnl"])
		pnLText, _ := showUSD(pnl, true)
		if len(pnLText) > maxLenPnL {
			maxLenPnL = len(pnLText)
		}
	}

	for i, trader := range okxAcc.Traders {
		pnl, _ := toFloat64(trader["copyTotalPnl"])
		todayPnl, _ := toFloat64(trader["todayPnl"])
		margin, _ := toFloat64(trader["margin"])

		pnLText, pnLColor := showUSD(pnl, true)
		todayPnlText, todayPnlColor := showUSD(todayPnl, false)
		todayPnLLen := 7 - len(todayPnlText)

		highlightColor := tcell.ColorOlive
		if margin == 0 {
			highlightColor = tcell.ColorGray
		}

		// Print PnL information
		tview.Print(screen, pnLText, x, y+(i*lineHeight)+1, width, tview.AlignLeft, pnLColor)

		// Print trader nickname
		tview.Print(screen, trader["nickName"].(string), x+maxLenPnL+1, y+(i*lineHeight)+1, width, tview.AlignLeft, highlightColor)

		// Print today's PnL information
		tview.Print(screen, "Today:", x+3, y+(i*lineHeight)+2, width, tview.AlignLeft, highlightColor)
		tview.Print(screen, todayPnlText, x+3+6+todayPnLLen, y+(i*lineHeight)+2, width, tview.AlignLeft, todayPnlColor)

		// Print margin information
		tview.Print(screen, fmt.Sprintf("Inv. %s", showMoney(margin)), x+3+6+8, y+(i*lineHeight)+2, width, tview.AlignLeft, highlightColor)
	}
	return 0, 0, 0, 0
}
