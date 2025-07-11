package sys

import (
	"runtime"
	"time"

	"github.com/go-ping/ping"
	"github.com/rs/zerolog"
)

var logger *zerolog.Logger

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
	Pinger   *ping.Pinger
	UpIcon   string
	DownIcon string
	Addr     string
	Err      error
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

func (s *StatsOnline) Initializer(log *zerolog.Logger) {
	logger = log

	s.IPK8S = &IPAddr{Addr: "103.206.205.154", Port: 443}
	s.IPDNS = &IPAddr{Addr: "10.203.1.202", Port: 53}
	s.IPAide = []*IPPinger{
		{Addr: "103.206.205.254", UpIcon: " → ", DownIcon: " × "},
		{Addr: "10.203.1.201", UpIcon: "[1]", DownIcon: "[×]"},
		{Addr: "10.203.1.202", UpIcon: "[2]", DownIcon: "[×]"},
		{Addr: "10.203.1.203", UpIcon: "[3]", DownIcon: "[×]"},
	}

	go func() {
		s.CheckAll()
		for _, e := range s.IPAide {
			if e.Pinger, e.Err = ping.NewPinger(e.Addr); e.Err != nil {
				logger.Error().Err(e.Err).Msg("Error creating pinger")
			}
			e.Pinger.Timeout = 500 * time.Millisecond
			e.Pinger.Count = 3
			if runtime.GOOS == "windows" {
				e.Pinger.SetPrivileged(true)
			}
			// Blocks until finished.
			if err := e.Pinger.Run(); err != nil {
				logger.Error().Err(err).Msg("Error running ping")
				if e.Err == nil {
					e.Err = err
				}
			}
		}
	}()
}

func (s *StatsOnline) CheckAll() {
	if s.IPK8S.Err = PingTCP(s.IPK8S.Addr, s.IPK8S.Port); s.IPK8S.Err != nil {
		logger.Error().Err(s.IPK8S.Err).Msg("Error pinging K8S")
	}

	if s.IPDNS.Err = PingTCP(s.IPDNS.Addr, s.IPDNS.Port); s.IPDNS.Err != nil {
		logger.Error().Err(s.IPDNS.Err).Msg("Error pinging DNS")
	}
}
