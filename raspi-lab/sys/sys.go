package sys

import (
	"runtime"
	"time"

	"github.com/go-ping/ping"
	"go.uber.org/zap"
)

var sugar *zap.SugaredLogger

type StatsOnline struct {
	IPK8S  *IPAddr
	IPDNS  *IPAddr
	IPAide []*IPPinger
}

type IPAddr struct {
	Addr  string
	Port  int
	Err   error
	State byte
}

type IPPinger struct {
	Pinger *ping.Pinger
	Addr   string
	Err    error
}

func (ip *IPPinger) IsOpened() bool {
	return ip.Err == nil && ip.Pinger.Statistics().AvgRtt > 0
}
func (ip *IPPinger) IsWait() bool {
	return ip.Pinger == nil
}

func (ip *IPAddr) IsOpened() bool {
	return ip.Err == nil
}

func (s *StatsOnline) Initializer(zp *zap.SugaredLogger) {
	sugar = zp

	s.IPK8S = &IPAddr{Addr: "103.206.205.154", Port: 443}
	s.IPDNS = &IPAddr{Addr: "10.203.1.202", Port: 53}
	s.IPAide = []*IPPinger{
		{Addr: "10.203.1.201"},
		{Addr: "10.203.1.202"},
		{Addr: "10.203.1.203"},
	}

	go func() {
		for _, e := range s.IPAide {
			if e.Pinger, e.Err = ping.NewPinger(e.Addr); e.Err != nil {
				sugar.Errorln(e.Err)
			}
			e.Pinger.Timeout = 500 * time.Millisecond
			e.Pinger.Count = 3
			if runtime.GOOS == "windows" {
				e.Pinger.SetPrivileged(true)
			}
			// Blocks until finished.
			if err := e.Pinger.Run(); err != nil {
				sugar.Errorln(err)
				if e.Err == nil {
					e.Err = err
				}
			}
		}
	}()
	s.CheckAll()
}

func (s *StatsOnline) CheckAll() {
	if s.IPK8S.Err = PingTCP(s.IPK8S.Addr, s.IPK8S.Port); s.IPK8S.Err != nil {
		sugar.Errorln(s.IPK8S.Err)
	}

	if s.IPDNS.Err = PingTCP(s.IPDNS.Addr, s.IPDNS.Port); s.IPDNS.Err != nil {
		sugar.Errorln(s.IPDNS.Err)
	}
}
