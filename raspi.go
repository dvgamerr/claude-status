package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func Clean(str string, args ...string) string {
	for _, arg := range args {
		str = strings.Replace(str, arg, "", -1)
	}
	str = strings.TrimSpace(str)
	return str
}

// Exec execute program and return stdout
func Exec(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// CPUTemp return cpu temp.
func GPUTemp() string {
	cpuTemp := Clean(Exec("vcgencmd", "measure_temp"), "temp=", "'C")
	if cpuTemp == "" {
		return "n/a"
	}
	return cpuTemp
}

// CPUMemory return cpu memory.
func GPUMemory() string {
	cpuMem := Clean(Exec("vcgencmd", "get_mem", "gpu"), "gpu=", "M")
	if cpuMem == "" {
		return "n/a"
	}
	return cpuMem
}

// CPUCoreVolt return core volt.
func CPUCoreVolt() string {
	cpuCoreVolt := Clean(Exec("vcgencmd", "measure_volts", "core"), "volt=", "V")
	if cpuCoreVolt == "" {
		return "n/a"
	}
	return cpuCoreVolt
}

type Memory struct {
	Total string
	Used  string
}

// vcgencmd measure_volts core
func MemoryInfo() (*Memory, error) {
	output := Exec("free", "-b")
	if output == "" {
		return &Memory{Total: "n/a", Used: "n/a"}, nil
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
		Total: BytesToText(total),
		Used:  BytesToText(used),
	}, nil
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
