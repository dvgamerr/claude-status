package main

import (
	"fmt"
	"log"
	"time"

	"github.com/dvgamerr/aide-lab/raspi-lab/okx"
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
	okxTodayPnLText, okxTodayPnLColor := getAmountUsdtColor(okxAcc.TodayPnL, true)
	balanceName := "OKX Est"
	balanceText := aNum.FormatMoney(okxAcc.Total)
	fulfillText := aNum.FormatMoney(okxAcc.Fulfill)
	okxPnLText, okxPnLColor := getAmountUsdtColor(okxAcc.Total-okxAcc.Fulfill, true)
	posBalanceText := 11 - len(balanceText)

	tview.Print(screen, fmt.Sprintf("(Capital: %s)", fulfillText), x-2, y+1, width, tview.AlignRight, tcell.ColorWhite)

	tview.Print(screen, balanceName, x+5, y+1, width, tview.AlignLeft, tcell.ColorDarkSlateGray)
	tview.Print(screen, balanceText, x+4+len(balanceName)+posBalanceText, y+1, width, tview.AlignLeft, tcell.ColorWhite)

	totalPnLPos := x + 5 + len(balanceName) + len(fulfillText)
	tview.Print(screen, "[", totalPnLPos+posBalanceText, y+1, width, tview.AlignLeft, tcell.ColorGray)
	tview.Print(screen, okxPnLText, totalPnLPos+posBalanceText+2, y+1, width, tview.AlignLeft, okxPnLColor)
	tview.Print(screen, "]", totalPnLPos+len(okxPnLText)+posBalanceText+3, y+1, width, tview.AlignLeft, tcell.ColorGray)

	tview.Print(screen, "Today's PnL", x+1, y+2, width, tview.AlignLeft, tcell.ColorDarkSlateGray)
	tview.Print(screen, fmt.Sprintf("%s (%s)", okxTodayPnLText, showPercent(okxAcc.TodayPercent)), x+15+(len(balanceText)-len(okxTodayPnLText)), y+2, width, tview.AlignLeft, okxTodayPnLColor)

	balanceName = "BTK Est"
	btkFulfill := 300.0
	btkBalance := 302.0
	btkPnLText, btkPnLColor := getAmountUsdtColor(btkBalance-btkFulfill, true)
	balanceText = aNum.FormatMoney(btkBalance)
	posBalanceText = 11 - len(balanceText)
	tview.Print(screen, balanceName, x+5, y+4, width, tview.AlignLeft, tcell.ColorDarkSlateGray)
	tview.Print(screen, balanceText, x+4+len(balanceName)+posBalanceText, y+4, width, tview.AlignLeft, tcell.ColorWhite)

	totalPnLPos = x + 5 + len(balanceName) + len(btkPnLText)
	tview.Print(screen, "[", totalPnLPos+posBalanceText+1, y+4, width, tview.AlignLeft, tcell.ColorGray)
	tview.Print(screen, btkPnLText, totalPnLPos+posBalanceText+3, y+4, width, tview.AlignLeft, btkPnLColor)
	tview.Print(screen, "]", totalPnLPos+posBalanceText+len(btkPnLText)+4, y+4, width, tview.AlignLeft, tcell.ColorGray)

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

func getBorderTop(w int) (txt string) {
	for i := 0; i < w; i++ {
		txt += "─"
	}
	return
}

// Define a draw function to draw a cross in the center of each cell.
func drawCopyTrader(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	lineHeight := 2

	headerText := " COPY TRADERS "
	tview.Print(screen, getBorderTop(width), x, y, width, tview.AlignLeft, tcell.ColorGrey)
	tview.Print(screen, " COPY TRADERS ", x+(width/2)-(len(headerText)/2), y, width, tview.AlignLeft, tcell.ColorTeal)

	maxLenPnL := 0
	for _, trader := range okxAcc.Traders {
		pnl, _ := toFloat64(trader["copyTotalPnl"])
		pnLText, _ := getAmountUsdtColor(pnl, true)
		if len(pnLText) > maxLenPnL {
			maxLenPnL = len(pnLText)
		}
	}

	for i, trader := range okxAcc.Traders {
		pnl, _ := toFloat64(trader["copyTotalPnl"])
		todayPnl, _ := toFloat64(trader["todayPnl"])
		margin, _ := toFloat64(trader["margin"])

		pnLText, _ := getAmountUsdtColor(pnl, true)
		todayPnlText, todayPnlColor := getAmountUsdtColor(todayPnl, false)
		todayPnLLen := 7 - len(todayPnlText)

		highlightColor := tcell.ColorOlive
		if margin == 0 {
			highlightColor = tcell.ColorGrey
		}

		// Print PnL information
		tview.Print(screen, pnLText, x, y+(i*lineHeight)+1, width, tview.AlignLeft, highlightColor)

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

// Define a draw function to draw a cross in the center of each cell.
func drawOrderPositionHistory(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	headerText := " ORDER POSITION "

	borderTop := getBorderTop(width)
	borderTop = "┬" + borderTop[3:]
	tview.Print(screen, borderTop, x, y, width, tview.AlignLeft, tcell.ColorGrey)
	tview.Print(screen, headerText, x+(width/2)-(len(headerText)/2), y, width, tview.AlignLeft, tcell.ColorTeal)

	for i := 0; i < height; i++ {
		tview.Print(screen, "│", x, y+i+1, width, tview.AlignLeft, tcell.ColorGrey)
	}

	totalHistory := len(okxAcc.Historys)
	lineDate := ""

	l := 0
	for i := 0; i < height; i++ {
		if i > totalHistory {
			continue
		}
		e := okxAcc.Historys[totalHistory-(i-l)-1]
		dateText := okx.ParseUnixDate(e["uTime"]).Format(okx.YYYYMMDD)
		if dateText != lineDate && dateText != time.Now().Format(okx.YYYYMMDD) {
			if i < height-2 {
				lineDate = dateText
				tview.Print(screen, dateText, x+2, y+i+1, width, tview.AlignLeft, tcell.ColorAqua)
				l++
			}
			continue
		}

		pnl := okx.CalcRealizedPnL(e)

		timeText := okx.ParseUnixDate(e["uTime"]).Format(okx.HHmm)
		mgnMode := e["mgnMode"].(string)
		uly := e["uly"].(string)
		ccy := uly[:len(uly)-5]

		// percent := pnl.PnL * 100 / (pnl.Closed / pnl.Lever)

		feeText := aNum.FormatMoney(pnl.Fee)
		closedText := aNum.FormatMoney(pnl.Closed)
		percentText := showPercent(pnl.PnLPercent)

		// closeTag := ")"
		// if pnl.PnLPercent-percent > 0.01 {
		// 	orderType := e["type"].(string)
		// 	closeTag = "|" + orderType + "|" + showPercent(percent) + ")"
		// }
		pnlText, pnlColor := getAmountUsdtColor(pnl.PnL, true)

		const (
			colTime   = 5
			colCcy    = 4
			colType   = 6
			colClosed = 9
			colPnL    = 8
			colPnLPer = 8
			colFee    = 6
		)

		const (
			labelPnL = "PnL:"
			labelFee = "Fee:"
		)

		func(x int, y int, p int) {
			sX := x + 3
			tview.Print(screen, timeText, sX, y, colTime, tview.AlignLeft, tcell.ColorGrey)
			tview.Print(screen, ccy, (p*1)+sX+colTime, y, colCcy, tview.AlignLeft, tcell.ColorWhite)
			tview.Print(screen, mgnMode, (p*2)+sX+colTime+colCcy, y, colType, tview.AlignLeft, tcell.ColorGrey)
			tview.Print(screen, closedText, (p*3)+sX+colTime+colCcy+colType, y, colClosed, tview.AlignRight, tcell.ColorAqua)
			tview.Print(screen, labelPnL, (p*4)+sX+colTime+colCcy+colType+colClosed, y, len(labelPnL), tview.AlignLeft, tcell.ColorGrey)
			tview.Print(screen, pnlText, (p*4)+sX+colTime+colCcy+colType+colClosed+len(labelPnL), y, colPnL, tview.AlignRight, pnlColor)
			tview.Print(screen, "(", (p*5)+sX+colTime+colCcy+colType+colClosed+len(labelPnL)+colPnL, y, 1, tview.AlignLeft, tcell.ColorGrey)
			tview.Print(screen, percentText, (p*5)+sX+colTime+colCcy+colType+colClosed+len(labelPnL)+colPnL, y, colPnLPer, tview.AlignRight, pnlColor)
			tview.Print(screen, ")", (p*5)+sX+colTime+colCcy+colType+colClosed+len(labelPnL)+colPnL+colPnLPer, y, 1, tview.AlignLeft, tcell.ColorGrey)
			tview.Print(screen, labelFee, (p*6)+sX+colTime+colCcy+colType+colClosed+len(labelPnL)+colPnL+colPnLPer+1, y, len(labelFee), tview.AlignLeft, tcell.ColorGrey)
			tview.Print(screen, feeText, (p*7)+sX+colTime+colCcy+colType+colClosed+len(labelPnL)+colPnL+colPnLPer+1+len(labelFee), y, colFee, tview.AlignLeft, tcell.ColorWhite)

		}(x, y+i+1, 1)
	}

	return 0, 0, 0, 0
}
