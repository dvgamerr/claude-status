package main

import (
	"crypto_lab/crypto-lab/btk"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/joho/godotenv"
	"github.com/leekchan/accounting"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	args ApplicationSettings
)

func init() {
	arg.MustParse(&args)

	if args.Tty == "" {
		godotenv.Load()
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// Check that all required environment variables are set
	if err := checkEnvList("BTK_APIKEY", "BTK_SECRETKEY"); err != nil {
		log.Fatal().Err(err).Send()
		os.Exit(1)
	}
}

// var (
// 	okxAcc okx.Account
// 	btkAcc btk.Account
// 	rpiOs  rpi.StatsOS
// 	stats  sys.StatsOnline
// )

func main() {
	aNum := accounting.Accounting{Precision: 2, Thousand: ","}

	log.Info().Str("name", "crypto-lab").Msg("Starting...")
	acc, err := btk.GetBalancer()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get Bitkub balances")
		os.Exit(1)
	}
	btk := acc.Account.(*btk.BitkubBalances)

	log.Info().
		Str("Total", aNum.FormatMoney(btk.Total)).
		Str("Available", aNum.FormatMoney(btk.Available)).
		Str("In-Order", aNum.FormatMoney(btk.InOrder)).
		Str("In-Withdraw", aNum.FormatMoney(btk.InWithdraw)).
		Msg("Bitkub.com Initialized")
	// if args.Develop {
	// 	// 	pinger, err := ping.NewPinger("10.203.1.202")
	// 	// 	if runtime.GOOS == "windows" {
	// 	// 		pinger.SetPrivileged(true)
	// 	// 	}
	// 	// 	if err != nil {
	// 	// 		sugar.Panicln(err)
	// 	// 	}
	// 	// 	pinger.Timeout = 500 * time.Millisecond
	// 	// 	pinger.Count = 3
	// 	// 	// if err := pinger.Run(); err != nil { // Blocks until finished.
	// 	// 	// 	sugar.Panicln(err)
	// 	// 	// }
	// 	// 	stats := pinger.Statistics()
	// 	// 	fmt.Printf("%.1fms\n", float64(stats.AvgRtt)/float64(time.Millisecond))

	// 	// 	// var res map[string]btk.Ticker
	// 	// 	// if err := btk.FetchNonSecure("GET", "/market/ticker?sym=THB_USDT", nil, &res); err != nil {
	// 	// 	// 	sugar.Errorln(err)
	// 	// 	// }
	// 	// 	// sugar.Debugf(" %+v\n", res["THB_USDT"].Last)

	// 	// 	// bal := btk.GetBalances()
	// 	// 	// fmt.Printf(" Total: %.2f\n", bal.Total)
	// 	// 	// checkResponseOKX()

	// 	accList, err := okx.GETAccountPositions()
	// 	if err != nil {
	// 		log.Fatal().Msg(err.Error())
	// 	}

	// 	for _, e := range accList {
	// 		timeText := okx.ParseUnixDate(e["uTime"]).Format(okx.HHmm)
	// 		instId := e["instId"].(string)
	// 		posSide := e["posSide"].(string)

	// 		fmt.Printf("%s %s %s\n", timeText, instId[:len(instId)-10], posSide)
	// 	}

	// 	// data, err := PrettyStruct(accList[len(accList)-1])
	// 	// if err != nil {
	// 	// 	sugar.Fatalln(err)
	// 	// }

	// 	// fmt.Print(data)

	// 	// accHis, err := okx.GETAccountPositionsHistory()
	// 	// if err != nil {
	// 	// 	sugar.Fatalln(err)
	// 	// }

	// 	// data, err = PrettyStruct(accHis[len(accHis)-1])
	// 	// if err != nil {
	// 	// 	sugar.Fatalln(err)
	// 	// }
	// 	// fmt.Print(data)
	// 	fmt.Print("done")

	// 	os.Exit(0)
	// }
	// go httpController()
	// defer loggingSync()

	// app := tview.NewApplication()

	// log.Info().Msg("OKX prettyplare initializing...")
	// rpiOs.Initializer(&log)
	// stats.Initializer(&log)
	// btkAcc.Initializer(&log)
	// okxAcc.Initializer(&log)

	// okxIntervel := 3 * time.Second
	// if os.Getenv("ENV") == "development" {
	// 	okxIntervel = 10 * time.Second
	// }

	// log.Info().Msg("Ticker interval setting...")
	// go setTickerInterval(500*time.Millisecond, func() { app.Draw() })()
	// if args.Tty != "" {
	// 	go setTickerInterval(1*time.Second, stats.CheckAll)()
	// 	go setTickerInterval(1*time.Second, rpiOs.GetOSStats)()
	// 	go setTickerInterval(okxIntervel, okxAcc.GetBalances)()
	// 	go setTickerInterval(okxIntervel, okxAcc.GetTreaders)()
	// 	go setTickerInterval(okxIntervel, btkAcc.GetTotalBalance)()
	// 	go setTickerInterval(okxIntervel, okxAcc.GetHistoryPositions)()
	// }

	// if args.Debug {
	// 	okxAcc.GetBalances()
	// 	okxAcc.GetTreaders()
	// 	okxAcc.GetHistoryPositions()

	// 	btkAcc.GetTotalBalance()
	// 	return
	// }

	// log.Info().Msg("Dashboard Renderer...")

	// flex := tview.NewFlex().
	// 	AddItem(tview.NewBox().SetDrawFunc(boxDrawOverview), 0, 1, false).
	// 	AddItem(tview.NewBox().SetDrawFunc(boxDrawStatsLabel), 9, 0, false).
	// 	AddItem(tview.NewBox().SetDrawFunc(boxDrawStatsValue), 22, 0, false)

	// grid := tview.NewGrid().SetRows(6, 0).SetColumns(32, 0).AddItem(flex, 0, 0, 1, 2, 0, 0, false)

	// // Layout for screens wider than 100 cells.
	// grid.
	// 	AddItem(tview.NewBox().SetDrawFunc(boxDrawCopyTrader), 1, 0, 1, 1, 0, 100, false).
	// 	AddItem(tview.NewBox().SetDrawFunc(boxDrawOrderPosition), 1, 1, 1, 1, 0, 100, false)

	// if err := app.SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
	// 	panic(err)
	// }

}
