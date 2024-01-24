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
func boxDrawStatsValue(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
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

	k8sLabel := "HTTP"
	k8sColor := tcell.ColorGreen
	if !stats.IPK8S.IsOpened() {
		k8sColor = tcell.ColorRed
	}
	tview.Print(screen, k8sLabel, x, y+4, width, tview.AlignLeft, k8sColor)

	dnsLabel := "DNS"
	dnsColor := tcell.ColorGreen
	if !stats.IPDNS.IsOpened() {
		dnsColor = tcell.ColorRed
	}
	tview.Print(screen, dnsLabel, x+1+len(k8sLabel), y+4, width, tview.AlignLeft, dnsColor)

	sX := x + 3 + len(k8sLabel) + len(dnsLabel)

	for i := 0; i < len(stats.IPAide); i++ {
		e := stats.IPAide[i]
		if e.IsWait() {
			tview.Print(screen, "[ ]", sX+(3*i), y+4, 4, tview.AlignLeft, tcell.ColorYellow)
		} else if !e.IsOpened() {
			tview.Print(screen, e.DownIcon, sX+(3*i), y+4, 4, tview.AlignLeft, tcell.ColorRed)
		} else {
			tview.Print(screen, e.UpIcon, sX+(3*i), y+4, 4, tview.AlignLeft, tcell.ColorGreen)
		}
	}

	return 0, 0, 0, 0
}

// Define a draw function to draw a cross in the center of each cell.
func boxDrawStatsLabel(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	tview.Print(screen, "Time:", x-1, y, width, tview.AlignRight, tcell.ColorNavy)
	tview.Print(screen, "Uptime:", x-1, y+1, width, tview.AlignRight, tcell.ColorNavy)
	tview.Print(screen, "CPU:", x-1, y+2, width, tview.AlignRight, tcell.ColorNavy)
	tview.Print(screen, "MEM:", x-1, y+3, width, tview.AlignRight, tcell.ColorNavy)
	tview.Print(screen, "Healthy:", x-1, y+4, width, tview.AlignRight, tcell.ColorNavy)

	return 0, 0, 0, 0
}

var aNum accounting.Accounting

// Define a draw function to draw a cross in the center of each cell.
func boxDrawOverview(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	labelBalance := "OKX Est"
	moneyBalance := aNum.FormatMoney(okxAcc.TotalTrade + okxAcc.TotalFund)
	fundingText := aNum.FormatMoney(okxAcc.TotalFund)
	// fulfillText := aNum.FormatMoney(okxAcc.Fulfill)
	okxPnLMoney, okxPnLColor := getAmountUsdtColor(okxAcc.TotalTrade+okxAcc.TotalFund-okxAcc.Fulfill, true)

	lenBalance := 9 - len(moneyBalance)
	tview.Print(screen, labelBalance, x+5, y+1, width, tview.AlignLeft, tcell.ColorDarkSlateGray)
	tview.Print(screen, moneyBalance, x+5+len(labelBalance)+lenBalance, y+1, width, tview.AlignLeft, tcell.ColorWhite)

	pX := x + 5 + len(labelBalance) + lenBalance + len(moneyBalance) + 1
	tview.Print(screen, "[", pX, y+1, width, tview.AlignLeft, tcell.ColorGray)
	tview.Print(screen, fundingText, pX+2, y+1, width, tview.AlignLeft, tcell.ColorGreen)
	tview.Print(screen, "]", pX+len(fundingText)+3, y+1, width, tview.AlignLeft, tcell.ColorGray)

	pX = x + 5 + len(labelBalance) + lenBalance + len(moneyBalance) + 1 + len(okxPnLMoney) + 1
	tview.Print(screen, "[", pX, y+1, width, tview.AlignLeft, tcell.ColorGray)
	tview.Print(screen, okxPnLMoney, pX+2, y+1, width, tview.AlignLeft, okxPnLColor)
	tview.Print(screen, "]", pX+len(okxPnLMoney)+3, y+1, width, tview.AlignLeft, tcell.ColorGray)

	okxTodayPnLText, okxTodayPnLColor := getAmountUsdtColor(okxAcc.TodayPnL, true)
	tview.Print(screen, "Today's PnL", x+1, y+2, width, tview.AlignLeft, tcell.ColorDarkSlateGray)
	lenBalance = 10 - len(okxTodayPnLText)
	tview.Print(screen, fmt.Sprintf("%s (%s)", okxTodayPnLText, showPercent(okxAcc.TodayPercent)), x+5+6+lenBalance, y+2, width, tview.AlignLeft, okxTodayPnLColor)

	labelBalance = "Bitkub Est"

	an := accounting.Accounting{Precision: 2, Thousand: ","}

	moneyBalance = an.FormatMoney(btkAcc.Total)
	moneyAvailable := an.FormatMoney(btkAcc.Available)
	moneyFulfill := an.FormatMoney((btkAcc.Total + btkAcc.Available) - btkAcc.Fulfill)
	_, btkPnLColor := getAmountUsdtColor((btkAcc.Total+btkAcc.Available)-btkAcc.Fulfill, true)

	tview.Print(screen, labelBalance, x+2, y+4, width, tview.AlignLeft, tcell.ColorDarkSlateGray)
	tview.Print(screen, moneyBalance, x+4+len(labelBalance), y+4, width, tview.AlignLeft, tcell.ColorWhite)

	pX = x + 4 + len(labelBalance) + len(moneyBalance) + 1
	tview.Print(screen, "[", pX, y+4, width, tview.AlignLeft, tcell.ColorGray)
	tview.Print(screen, moneyAvailable, pX+2, y+4, width, tview.AlignLeft, tcell.ColorGreen)
	tview.Print(screen, "]", pX+len(moneyAvailable)+3, y+4, width, tview.AlignLeft, tcell.ColorGray)

	pX = x + 4 + len(labelBalance) + len(moneyBalance) + 1 + len(moneyAvailable) + 5
	tview.Print(screen, "[", pX, y+4, width, tview.AlignLeft, tcell.ColorGray)
	tview.Print(screen, moneyFulfill, pX+2, y+4, width, tview.AlignLeft, btkPnLColor)
	tview.Print(screen, "]", pX+len(moneyFulfill)+3, y+4, width, tview.AlignLeft, tcell.ColorGray)

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
func boxDrawCopyTrader(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	lineHeight := 2

	headerText := " COPY TRADERS "
	tview.Print(screen, getBorderTop(width), x, y, width, tview.AlignLeft, tcell.ColorGrey)
	tview.Print(screen, " COPY TRADERS ", x+(width/2)-(len(headerText)/2), y, width, tview.AlignLeft, tcell.ColorTeal)

	maxLenPnL := 0
	for _, trader := range okxAcc.Traders {
		pnl, _ := okx.ParseMoney(trader["copyTotalPnl"])
		pnLText, _ := getAmountUsdtColor(pnl, true)
		if len(pnLText) > maxLenPnL {
			maxLenPnL = len(pnLText)
		}
	}

	for i, trader := range okxAcc.Traders {
		pnl, _ := okx.ParseMoney(trader["copyTotalPnl"])
		todayPnl, _ := okx.ParseMoney(trader["todayPnl"])
		margin, _ := okx.ParseMoney(trader["margin"])

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

func checkfoundPositionOrder(posId string) bool {
	for _, o := range okxAcc.Ongoing {
		if posId == o["posId"].(string) {
			return true
		}
	}
	return false
}

// Define a draw function to draw a cross in the center of each cell.
func boxDrawOrderPosition(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	headerText := " ORDER POSITION "

	borderTop := getBorderTop(width)
	borderTop = "┬" + borderTop[3:]
	tview.Print(screen, borderTop, x, y, width, tview.AlignLeft, tcell.ColorGrey)
	tview.Print(screen, headerText, x+(width/2)-(len(headerText)/2), y, width, tview.AlignLeft, tcell.ColorTeal)

	for i := 0; i < height; i++ {
		tview.Print(screen, "│", x, y+i+1, width, tview.AlignLeft, tcell.ColorGrey)
	}

	totalOngoing := len(okxAcc.Ongoing)
	totalHistory := len(okxAcc.Historys)
	lineDate := ""

	for i := totalOngoing - 1; i >= 0; i-- {
		renderHistoryTable(screen, okxAcc.Ongoing[i], x, y+(totalOngoing-1)-i+1, 1, false)
	}

	tview.Print(screen, getBorderTop(int(float64(width)/1.5)), x, y+totalOngoing+1, width, tview.AlignCenter, tcell.ColorGrey)

	l := 0
	for i := totalOngoing; i < height; i++ {
		if i > totalHistory {
			continue
		}

		e := okxAcc.Historys[totalHistory-(i-l)-1]

		// if checkfoundPositionOrder(e["posId"].(string)) {
		// 	continue
		// }

		dateText := okx.ParseUnixDate(e["uTime"]).Format(okx.YYYYMMDD)
		if dateText != lineDate {
			if i < height-2 {
				lineDate = dateText
				tview.Print(screen, dateText, x+2, y+i+2, width, tview.AlignLeft, tcell.ColorAqua)
				l++
			}
			continue
		}

		renderHistoryTable(screen, e, x, y+i+2, 1, true)
	}

	return 0, 0, 0, 0
}

func renderHistoryTable(screen tcell.Screen, e map[string]interface{}, x int, y int, p int, realizedPnL bool) {

	pnl := okx.CalcRealizedPnL(e)

	timeText := okx.ParseUnixDate(e["uTime"]).Format(okx.HHmm)
	direction := ""
	if e["direction"] == nil {
		direction = e["posSide"].(string)
	} else {
		direction = e["direction"].(string)
	}
	instId := e["instId"].(string)
	ccy := instId[:len(instId)-10]

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
	if realizedPnL {
		pnlText, pnlColor = getAmountUsdtColor(pnl.PnLRealized, true)
	}

	const (
		colTime   = 5
		colCcy    = 5
		colDirect = 5
		colClosed = 9
		colPnL    = 8
		colPnLPer = 8
		colFee    = 6
	)

	const (
		labelPnL = "PnL:"
		labelFee = "Fee:"
	)

	sX := x + 3
	tview.Print(screen, timeText, sX, y, colTime, tview.AlignLeft, tcell.ColorGrey)
	tview.Print(screen, ccy, (p*1)+sX+colTime, y, colCcy, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, direction, (p*2)+sX+colTime+colCcy, y, colDirect, tview.AlignLeft, tcell.ColorGrey)
	tview.Print(screen, closedText, (p*3)+sX+colTime+colCcy+colDirect, y, colClosed, tview.AlignRight, tcell.ColorAqua)
	tview.Print(screen, labelPnL, (p*4)+sX+colTime+colCcy+colDirect+colClosed, y, len(labelPnL), tview.AlignLeft, tcell.ColorGrey)
	tview.Print(screen, pnlText, (p*4)+sX+colTime+colCcy+colDirect+colClosed+len(labelPnL), y, colPnL, tview.AlignRight, pnlColor)
	tview.Print(screen, "(", (p*5)+sX+colTime+colCcy+colDirect+colClosed+len(labelPnL)+colPnL, y, 1, tview.AlignLeft, tcell.ColorGrey)
	tview.Print(screen, percentText, (p*5)+sX+colTime+colCcy+colDirect+colClosed+len(labelPnL)+colPnL, y, colPnLPer, tview.AlignRight, pnlColor)
	tview.Print(screen, ")", (p*5)+sX+colTime+colCcy+colDirect+colClosed+len(labelPnL)+colPnL+colPnLPer, y, 1, tview.AlignLeft, tcell.ColorGrey)
	tview.Print(screen, labelFee, (p*6)+sX+colTime+colCcy+colDirect+colClosed+len(labelPnL)+colPnL+colPnLPer+1, y, len(labelFee), tview.AlignLeft, tcell.ColorGrey)
	tview.Print(screen, feeText, (p*7)+sX+colTime+colCcy+colDirect+colClosed+len(labelPnL)+colPnL+colPnLPer+1+len(labelFee), y, colFee, tview.AlignLeft, tcell.ColorWhite)

}
