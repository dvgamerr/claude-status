package sys

import (
	"go.uber.org/zap"
)

var sugar *zap.SugaredLogger

type StatsOnline struct {
	IPK8S  *IPAddr
	IPDNS  *IPAddr
	IPAide []*IPAddr
}

type IPAddr struct {
	Addr string
	Port int
	Err  error
}

func (ip *IPAddr) IsOpened() bool {
	return ip.Err == nil
}

func (s *StatsOnline) Initializer(zp *zap.SugaredLogger) {
	sugar = zp

	s.IPK8S = &IPAddr{Addr: "103.206.205.154", Port: 443}
	s.IPDNS = &IPAddr{Addr: "10.203.1.202", Port: 53}
	s.IPAide = []*IPAddr{
		{Addr: "10.203.1.201", Port: 0},
		{Addr: "10.203.1.202", Port: 0},
		{Addr: "10.203.1.203", Port: 0},
	}
	s.CheckAll()
}

func (s *StatsOnline) CheckAll() {
	if s.IPK8S.Err = PingTCP(s.IPK8S.Addr, s.IPK8S.Port); s.IPK8S.Err != nil {
		sugar.Errorln(s.IPK8S.Err)
	}

	if s.IPDNS.Err = PingTCP(s.IPDNS.Addr, s.IPDNS.Port); s.IPDNS.Err != nil {
		sugar.Errorln(s.IPDNS.Err)
	}

	for _, e := range s.IPAide {
		if e.Err = PingIP(e.Addr); e.Err != nil {
			sugar.Errorln(e.Err)
		}
	}
}
