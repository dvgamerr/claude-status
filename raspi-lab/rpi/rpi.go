package rpi

import (
	"sync"

	"go.uber.org/zap"
)

type StatsOS struct {
	CPUTemp string
	GPUTemp string
}

// var sugar *zap.SugaredLogger

func (s *StatsOS) Initializer(zp *zap.SugaredLogger) {
	// sugar = l
	s.GetOSStats()
}

func (s *StatsOS) GetOSStats() {
	w := &sync.WaitGroup{}
	w.Add(2)
	go func() {
		s.CPUTemp = CPUTemp()
		w.Done()
	}()
	go func() {
		s.GPUTemp = GPUTemp()
		w.Done()
	}()
	w.Wait()
}
