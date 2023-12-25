package rpi

import (
	"fmt"
	"strconv"
	"strings"
)

type Memory struct {
	Total uint64
	Used  uint64
}

// vcgencmd measure_volts core
func MemoryInfo() (*Memory, error) {
	output := execPi("free", "-b")
	if output == "" {
		return &Memory{Total: 0, Used: 0}, nil
	}

	lines := strings.Split(output, "\n")

	if len(lines) < 2 {
		return nil, fmt.Errorf("unexpected output format")
	}

	fields := strings.Fields(lines[1]) // Assuming the relevant information is in the second line

	if len(fields) < 3 {
		return nil, fmt.Errorf("unexpected output format")
	}

	total, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return nil, err
	}

	used, err := strconv.ParseUint(fields[2], 10, 64)
	if err != nil {
		return nil, err
	}

	return &Memory{
		Total: total,
		Used:  used,
	}, nil
}

func (m *Memory) TotalText() string {
	return BytesToText(m.Total)
}
func (m *Memory) UsedText() string {
	return BytesToText(m.Used)
}
func (m *Memory) Percent() string {
	if m.Total > 0 {
		return fmt.Sprintf("%.1f", float64(m.Used)*100.0/float64(m.Total))
	}
	return "0.0%"
}

func BytesToText(size uint64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
	)

	switch {
	case size < KB:
		return fmt.Sprintf("%d bytes", size)
	case size < MB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	case size < GB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	default:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	}
}
