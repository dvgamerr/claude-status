package rpi

import (
	"sync"

	"github.com/rs/zerolog"
)

type StatsOS struct {
	CPUTemp string
	GPUTemp string
}

var logger *zerolog.Logger

func (s *StatsOS) Initializer(log *zerolog.Logger) {
	logger = log
	s.GetOSStats()
}

func (s *StatsOS) GetOSStats() {
	w := &sync.WaitGroup{}
	w.Add(2)
	go func() {
		defer w.Done()
		s.CPUTemp = CPUTemp()
	}()
	go func() {
		defer w.Done()
		s.GPUTemp = GPUTemp()
	}()
	w.Wait()
}
