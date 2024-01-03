package rpi

import "fmt"

func GPUTemp() string {
	cpuTemp := clean(execPi("vcgencmd", "measure_temp"), "temp=", "'C")
	if cpuTemp == "" {
		return "00.0°C"
	}
	return fmt.Sprintf("%s°C", cpuTemp)
}
