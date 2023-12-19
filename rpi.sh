#!/bin/bash

check_updated () {
  echo Checking version latest...
  /usr/bin/git pull origin --quiet
  echo "----------------------------------"
}

cd /home/pi/lab > /dev/null
if [ -f "/usr/bin/git" ]; then
  if [[ "$1" != "-w" ]]; then
    check_updated
  fi
fi

if [ -f "/home/pi/lab/bin/raspi" ]; then
  /home/pi/lab/bin/raspi --tty $(tty)
fi

if [[ $(tty) != /dev/tty1 ]]; then
  # Purpose: Display the ARM CPU and GPU  temperature of Raspberry Pi 2/3 
  # -------------------------------------------------------
  cpu=$(</sys/class/thermal/thermal_zone0/temp)
  gpu=$(/usr/bin/vcgencmd measure_temp | grep -o -E '[[:digit:]].*')
  echo "CPU => $((cpu/1000))'C GPU => $gpu"
  echo "----------------------------------"
fi

