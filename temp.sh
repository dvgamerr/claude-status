#!/bin/bash
# Purpose: Display the ARM CPU and GPU  temperature of Raspberry Pi 2/3 
# -------------------------------------------------------
cpu=$(</sys/class/thermal/thermal_zone0/temp)
echo "$(date) @ $(hostname)"
echo "-------------------------------------------"
echo "GPU => $(/usr/bin/vcgencmd measure_temp | grep -o -E '[[:digit:]].*')"
echo "CPU => $((cpu/1000))'C"
