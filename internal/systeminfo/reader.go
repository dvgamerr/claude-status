package systeminfo

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Stats struct {
	CPUPercent       *float64
	MemoryUsedBytes  *uint64
	MemoryTotalBytes *uint64
	Load1            *float64
	TemperatureC     *float64
	Uptime           *time.Duration
}

type Reader struct {
	root    string
	mu      sync.Mutex
	lastCPU cpuSample
	hasCPU  bool
}

type cpuSample struct {
	total uint64
	idle  uint64
}

func NewReader(root string) *Reader {
	if strings.TrimSpace(root) == "" {
		root = string(filepath.Separator)
	}
	return &Reader{root: filepath.Clean(root)}
}

// Read collects best-effort Linux metrics. Missing files are represented as
// nil fields so the dashboard remains usable on macOS, Windows, and unusual Pi
// images instead of failing as a whole.
func (r *Reader) Read() Stats {
	r.mu.Lock()
	defer r.mu.Unlock()

	stats := Stats{}

	if data, err := os.ReadFile(r.path("proc", "stat")); err == nil {
		if sample, err := parseCPUSample(string(data)); err == nil {
			if r.hasCPU && sample.total > r.lastCPU.total && sample.idle >= r.lastCPU.idle {
				totalDelta := sample.total - r.lastCPU.total
				idleDelta := sample.idle - r.lastCPU.idle
				if idleDelta <= totalDelta {
					value := 100 * float64(totalDelta-idleDelta) / float64(totalDelta)
					stats.CPUPercent = &value
				}
			}
			r.lastCPU = sample
			r.hasCPU = true
		}
	}

	if data, err := os.ReadFile(r.path("proc", "meminfo")); err == nil {
		if used, total, err := parseMemory(string(data)); err == nil {
			stats.MemoryUsedBytes = &used
			stats.MemoryTotalBytes = &total
		}
	}

	if data, err := os.ReadFile(r.path("proc", "loadavg")); err == nil {
		if value, err := parseLoad(string(data)); err == nil {
			stats.Load1 = &value
		}
	}

	if data, err := os.ReadFile(r.path("proc", "uptime")); err == nil {
		if value, err := parseUptime(string(data)); err == nil {
			stats.Uptime = &value
		}
	}

	for _, path := range r.temperaturePaths() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if value, err := parseTemperature(string(data)); err == nil {
			stats.TemperatureC = &value
			break
		}
	}
	return stats
}

func (r *Reader) path(parts ...string) string {
	all := append([]string{r.root}, parts...)
	return filepath.Join(all...)
}

func (r *Reader) temperaturePaths() []string {
	paths := []string{r.path("sys", "class", "thermal", "thermal_zone0", "temp")}
	matches, _ := filepath.Glob(r.path("sys", "class", "hwmon", "hwmon*", "temp1_input"))
	return append(paths, matches...)
}

func parseCPUSample(data string) (cpuSample, error) {
	line, _, _ := strings.Cut(data, "\n")
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuSample{}, fmt.Errorf("invalid /proc/stat cpu line")
	}

	values := make([]uint64, 0, len(fields)-1)
	for _, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return cpuSample{}, fmt.Errorf("invalid cpu counter %q: %w", field, err)
		}
		values = append(values, value)
	}
	// guest and guest_nice are already included in user and nice, respectively.
	limit := min(len(values), 8)
	var total uint64
	for _, value := range values[:limit] {
		total += value
	}
	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}
	return cpuSample{total: total, idle: idle}, nil
}

func parseMemory(data string) (used, total uint64, err error) {
	values := make(map[string]uint64)
	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimSuffix(fields[0], ":")
		value, parseErr := strconv.ParseUint(fields[1], 10, 64)
		if parseErr != nil {
			continue
		}
		values[name] = value * 1024
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}
	total = values["MemTotal"]
	if total == 0 {
		return 0, 0, fmt.Errorf("MemTotal is missing")
	}
	available, found := values["MemAvailable"]
	if !found {
		available = values["MemFree"] + values["Buffers"] + values["Cached"]
	}
	if available > total {
		available = total
	}
	return total - available, total, nil
}

func parseLoad(data string) (float64, error) {
	fields := strings.Fields(data)
	if len(fields) == 0 {
		return 0, fmt.Errorf("load average is empty")
	}
	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0, fmt.Errorf("invalid load average %q", fields[0])
	}
	return value, nil
}

func parseUptime(data string) (time.Duration, error) {
	fields := strings.Fields(data)
	if len(fields) == 0 {
		return 0, fmt.Errorf("uptime is empty")
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < 0 {
		return 0, fmt.Errorf("invalid uptime %q", fields[0])
	}
	return time.Duration(seconds * float64(time.Second)), nil
}

func parseTemperature(data string) (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(data), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("invalid temperature %q", strings.TrimSpace(data))
	}
	if math.Abs(value) >= 1000 {
		value /= 1000
	}
	if value < -40 || value > 150 {
		return 0, fmt.Errorf("temperature %.1f is outside the supported range", value)
	}
	return value, nil
}
