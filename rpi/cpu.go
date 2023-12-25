package rpi

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// CPUTemp return cpu temp.
func CPUTemp() string {
	// Read CPU temperature from file
	cpuTemp, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		return "00.0°C"
	}

	// Convert content to integer
	cpuInt, err := strconv.Atoi(strings.TrimSpace(string(cpuTemp)))
	if err != nil {
		log.Fatal(err)
	}

	// Convert the temperature in millidegrees Celsius
	return fmt.Sprintf("%.1f°C", float64(cpuInt)/1000)
}

// CPUCoreVolt return core volt.
func CPUCoreVolt() string {
	cpuCoreVolt := clean(execPi("vcgencmd", "measure_volts", "core"), "volt=", "V")
	f, err := strconv.ParseFloat(cpuCoreVolt, 64)
	if err != nil {
		return "0.00V"
	}
	return fmt.Sprintf("%.2fV", f)
}

// CPUMemory return cpu memory.
func GPUMemory() string {
	cpuMem := clean(execPi("vcgencmd", "get_mem", "gpu"), "gpu=", "M")
	if cpuMem == "" {
		return "n/a"
	}
	return cpuMem
}
