package main

import (
	"log"
	"os/exec"
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
		log.Println(err)
		return ""
	}
	return string(out)
}

// CPUTemp return cpu temp.
func CPUTemp(data string) string {
	cpuTemp := Clean(Exec("vcgencmd", "measure_temp"), "temp=", "'C")
	if cpuTemp == "" {
		return "n/a"
	}
	return cpuTemp
}

// CPUMemory return cpu memory.
func CPUMemory(data string) string {

	cpuMem := Clean(Exec("vcgencmd", "get_mem", "arm"), "arm=", "M")
	if cpuMem == "" {
		return "n/a"
	}
	return cpuMem
}

// CPUCoreVolt return core volt.
func CPUCoreVolt(data string) string {
	cpuCoreVolt := Clean(Exec("vcgencmd", "measure_volts", "core"), "volt=", "V")
	if cpuCoreVolt == "" {
		return "n/a"
	}
	return cpuCoreVolt
}
