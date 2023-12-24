#!/bin/bash
check_updated () {
  echo Checking \`aide-pi\` latest version...
  /usr/bin/git pull origin --quiet
  echo "----------------------------------"
}

if [[ "$1" == "--blacklight" ]]; then
  if [[ "$2" == "off" ]]; then
    for v in 255 239 223 207 191 175 159 143 127 111 95 79 63 47 31 15 0; do
      vcgencmd set_backlight $v > /dev/null
      sleep 0.02;
    done;
  else
    for v in 0 15 31 47 63 79 95 111 127 143 159 175 191 207 223 239 255 ; do
      vcgencmd set_backlight $v > /dev/null
      sleep 0.04;
    done;
  fi
  exit 0
fi

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
else if [ -f "/home/pi/lab/bin/raspi-lab" ]; then
  echo "``raspi-lab`` compiling..."
  go get . > /dev/null
  go build -o /home/pi/lab/bin/raspi-lab . > /dev/null
  /home/pi/lab/bin/raspi-lab --tty $(tty)
fi

