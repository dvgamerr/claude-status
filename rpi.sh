#!/bin/bash

if [[ "$1" != "--blacklight" ]]; then
  if [[ "$2" == "off" ]]; then
    for v in 255 239 223 207 191 175 159 143 127 111 95 79 63 47 31 15 0; do
      vcgencmd set_backlight $v;
      sleep 0.04;
    done;
  else
    for v in 0 15 31 47 63 79 95 111 127 143 159 175 191 207 223 239 255 ; do
      vcgencmd set_backlight $v;
      sleep 0.04;
    done;
  fi
fi

check_updated () {
  echo Checking \`aide-pi\` version latest...
  /usr/bin/git pull origin --quiet
  echo "----------------------------------"
}

cd /home/pi/lab > /dev/null
if [ -f "/usr/bin/git" ]; then
  if [[ "$1" != "-w" ]]; then
    check_updated
  fi
fi

if [[ $(tty) != /dev/tty1 ]]; then
  # Purpose: Display the ARM CPU and GPU  temperature of Raspberry Pi 2/3 
  # -------------------------------------------------------
  cpu=$(</sys/class/thermal/thermal_zone0/temp)
  gpu=$(/usr/bin/vcgencmd measure_temp | grep -o -E '[[:digit:]].*')
  echo "CPU => $((cpu/1000))'C GPU => $gpu"
  echo "----------------------------------"
fi

if [ -f "/home/pi/lab/bin/raspi" ]; then
  /home/pi/lab/bin/raspi --tty $(tty)
fi