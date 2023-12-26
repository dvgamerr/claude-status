package main

import (
	"fmt"
	"log"
	"time"

	"github.com/dvgamerr/aide-lab/rpi"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

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
	tview.Print(screen, "Time:", x-1, y, width, tview.AlignRight, tcell.ColorTeal)
	tview.Print(screen, "Uptime:", x-1, y+1, width, tview.AlignRight, tcell.ColorTeal)
	tview.Print(screen, "GPU:", x-1, y+2, width, tview.AlignRight, tcell.ColorTeal)
	tview.Print(screen, "CPU:", x-1, y+3, width, tview.AlignRight, tcell.ColorTeal)
	tview.Print(screen, "MEM:", x-1, y+4, width, tview.AlignRight, tcell.ColorTeal)
	return 0, 0, 0, 0
}

// Define a draw function to draw a cross in the center of each cell.
func drawOverviewHeader(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {

	// percent := (totalEqual * 100 / fulfill) - 100
	// _, _, day := time.Now().Date()
	// fmt.Printf("%s %s USD | %s (%s) %s",
	// 	bold("OKX Total PnL:"),
	// 	info(fmt.Sprintf("$%.2f", totalEqual)),
	// 	printfProfit("$%.2f", totalEqual-fulfill),
	// 	printfProfit("%.2f%%", percent),
	// 	bold(fmt.Sprintf("%dDay", day)),
	// )

	tview.Print(screen, "OKX PnL:", x+2, y+1, width, tview.AlignLeft, tcell.ColorLightSeaGreen)
	tview.Print(screen, "BTK PnL:", x+2, y+2, width, tview.AlignLeft, tcell.ColorLightGrey)
	tview.Print(screen, "Uptime:", x+2, y+3, width, tview.AlignLeft, tcell.ColorDarkOliveGreen)
	tview.Print(screen, "GPU:", x+2, y+4, width, tview.AlignLeft, tcell.ColorMediumSpringGreen)
	return 0, 0, 0, 0
}
