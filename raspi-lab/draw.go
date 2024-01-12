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
	tview.Print(screen, "GPU:", x+9, y+2, width, tview.AlignLeft, tcell.ColorDarkSlateGray)
	tview.Print(screen, rpiOs.GPUTemp, x+14, y+2, width, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, fmt.Sprintf("%s (%s)", mem.Percent(), mem.UsedText()), x, y+3, width, tview.AlignLeft, tcell.ColorWhite)
	return 0, 0, 0, 0
}

// Define a draw function to draw a cross in the center of each cell.
func drawOSHeaderText(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	tview.Print(screen, "Time:", x-1, y, width, tview.AlignRight, tcell.ColorDarkSlateGray)
	tview.Print(screen, "Uptime:", x-1, y+1, width, tview.AlignRight, tcell.ColorDarkSlateGray)
	tview.Print(screen, "CPU:", x-1, y+2, width, tview.AlignRight, tcell.ColorDarkSlateGray)
	tview.Print(screen, "MEM:", x-1, y+3, width, tview.AlignRight, tcell.ColorDarkSlateGray)
	tview.Print(screen, "K8S:", x-1, y+4, width, tview.AlignRight, tcell.ColorDarkSlateGray)
	tview.Print(screen, "DNS:", x-1, y+5, width, tview.AlignRight, tcell.ColorDarkSlateGray)
	return 0, 0, 0, 0
}

var aNum accounting.Accounting

// Define a draw function to draw a cross in the center of each cell.
func drawOverviewHeader(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	okxTodayPnLText, okxTodayPnLColor := showUSD(okxAcc.TodayPnL, true)
	balanceName := "OKX Est:"
	balanceText := aNum.FormatMoney(okxAcc.Total)
	fulfillText := aNum.FormatMoney(okxAcc.Fulfill)
	okxPnLText, okxPnLColor := showUSD(okxAcc.Total-okxAcc.Fulfill, true)

	tview.Print(screen, fmt.Sprintf("(Capital: %s)", fulfillText), x-2, y+1, width, tview.AlignRight, tcell.ColorWhite)

	tview.Print(screen, balanceName, x+3, y+1, width, tview.AlignLeft, tcell.ColorDarkSlateGray)
	tview.Print(screen, balanceText, x+5+len(balanceName), y+1, width, tview.AlignLeft, tcell.ColorWhite)
	tview.Print(screen, fmt.Sprintf("(%s)", okxPnLText), x+6+len(balanceName)+len(fulfillText), y+1, width, tview.AlignLeft, okxPnLColor)

	tview.Print(screen, "Today's PnL", x+1, y+2, width, tview.AlignLeft, tcell.ColorDarkSlateGray)
	tview.Print(screen, fmt.Sprintf("%s (%s)", okxTodayPnLText, showPercent(okxAcc.TodayPercent)), x+13+(len(balanceText)-len(okxTodayPnLText)), y+2, width, tview.AlignLeft, okxTodayPnLColor)

	// btkFulfill := 90000
	// btkPnLText, btkPnLColor := showUSD(1000)
	// tview.Print(screen, "BTK PnL:", x+vw, y+1, vw, tview.AlignLeft, tcell.ColorGreen)
	// tview.Print(screen, aNum.FormatMoney(btkFulfill), x+vw, y+1, vw, tview.AlignLeft, tcell.ColorWhite)
	// tview.Print(screen, fmt.Sprintf("(%s)", btkPnLText), x+vw, y+1, vw, tview.AlignLeft, btkPnLColor)

	return 0, 0, 0, 0
}

//	{
//	  "beginCopyTime": "1704935878159",
//	  "ccy": "USDT",
//	  "copyTotalAmt": "600",
//	  "copyTotalPnl": "0",
//	  "leadMode": "public",
//	  "margin": "0",
//	  "nickName": "Cultivated-DYDX-Lock",
//	  "portLink": "https://static.okx.com/cdn/okex/users/headimages/20230805/b5da2f821a10472e949f8a7fd7af35f2",
//	  "profitSharingRatio": "0.1",
//	  "todayPnl": "0",
//	  "uniqueCode": "764530FE18217877",
//	  "upl": "0"
//	}
const ColorBlack = 0
const ColorRed = 1
const ColorWhiteGray = 7
const ColorDrakGray = 8

// Define a draw function to draw a cross in the center of each cell.
func drawTraderList(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
	line := 2
	tview.Print(screen, "MY TRADERS", x, y, width, tview.AlignLeft, tcell.ColorWhiteSmoke)
	for i, e := range okxAcc.Traders {
		pnl, _ := toFloat64(e["copyTotalPnl"])
		todayPnl, _ := toFloat64(e["todayPnl"])
		margin, _ := toFloat64(e["margin"])
		pnLText, pnLColor := showUSD(pnl, true)
		todayPnlText, todayPnlColor := showUSD(todayPnl, false)
		pnlLen := 7 - len(todayPnlText)
		// copyLen := 7 - len(showMoney(copyTotalPnl))
		if margin == 0 {
			tview.Print(screen, e["nickName"].(string), x, y+(i*line)+1, width, tview.AlignLeft, ColorDrakGray)
			tview.Print(screen, pnLText, x, y+(i*line)+1, width, tview.AlignRight, ColorDrakGray)
			tview.Print(screen, "Today:", x+2, y+(i*line)+2, width, tview.AlignLeft, ColorDrakGray)
			tview.Print(screen, todayPnlText, x+2+6+pnlLen, y+(i*line)+2, width, tview.AlignLeft, ColorDrakGray)
			tview.Print(screen, fmt.Sprintf("Inv. %s", showMoney(margin)), x+2+6+8, y+(i*line)+2, width, tview.AlignLeft, ColorDrakGray)
		} else {
			tview.Print(screen, e["nickName"].(string), x, y+(i*line)+1, width, tview.AlignLeft, tcell.ColorDarkSlateGray)
			tview.Print(screen, pnLText, x, y+(i*line)+1, width, tview.AlignRight, pnLColor)
			tview.Print(screen, "Today:", x+2, y+(i*line)+2, width, tview.AlignLeft, tcell.ColorGrey)
			tview.Print(screen, todayPnlText, x+2+6+pnlLen, y+(i*line)+2, width, tview.AlignLeft, todayPnlColor)
			tview.Print(screen, fmt.Sprintf("Inv. %s", showMoney(margin)), x+2+6+8, y+(i*line)+2, width, tview.AlignLeft, tcell.ColorGrey)
		}

	}
	return 0, 0, 0, 0
}
