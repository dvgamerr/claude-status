package rpi

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

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

func clean(str string, args ...string) string {
	for _, arg := range args {
		str = strings.Replace(str, arg, "", -1)
	}
	str = strings.TrimSpace(str)
	return str
}

// execPi execute program and return stdout
func execPi(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}
