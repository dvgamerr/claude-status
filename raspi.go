package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
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

func CPUTemp() string {
	// Open the file
	file, err := os.Open("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		return "n/a"
	}
	defer file.Close()

	cpuTemp, err := io.ReadAll(file)
	if err != nil {
		return "n/a"
	}

	// Convert the CPU temperature to an integer
	tempInt, err := strconv.Atoi(strings.TrimSpace(string(cpuTemp)))
	if err != nil {
		log.Fatal(err)
	}

	// Convert the temperature to Celsius
	return fmt.Sprintf("%d°C", tempInt/1000)
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
	Total uint64
	Used  uint64
}

// vcgencmd measure_volts core
func MemoryInfo() (*Memory, error) {
	output := Exec("free", "-b")
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
		return fmt.Sprintf("%d", m.Used*100.0/m.Total)
	}
	return "0%"
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

func GetUptime() (string, error) {
	// Read the /proc/uptime file
	content, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "n/a", nil
	}

	// Extract the uptime value from the file
	fields := strings.Fields(string(content))
	if len(fields) < 1 {
		return "0", fmt.Errorf("unexpected format in /proc/uptime")
	}

	uptimeSeconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return "0", err
	}

	// Convert uptime to time.Duration
	uptimeDuration := time.Duration(int64(uptimeSeconds)) * time.Second

	return formatUptime(uptimeDuration), nil
}

func formatUptime(uptime time.Duration) string {
	years := uptime / (365 * 24 * time.Hour)
	uptime %= 365 * 24 * time.Hour

	months := uptime / (30 * 24 * time.Hour)
	uptime %= 30 * 24 * time.Hour

	days := uptime / (24 * time.Hour)
	uptime %= 24 * time.Hour

	hours := uptime / time.Hour
	uptime %= time.Hour

	minutes := uptime / time.Minute
	uptime %= time.Minute

	seconds := uptime / time.Second

	var result string

	if years > 0 {
		result += fmt.Sprintf("%d year, ", years)
	}

	if months > 0 {
		result += fmt.Sprintf("%d month, ", months)
	}

	if days > 0 {
		result += fmt.Sprintf("%d day, ", days)
	}

	result += fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)

	return strings.TrimSuffix(result, ", ")
}
