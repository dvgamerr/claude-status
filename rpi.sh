#!/usr/bin/zsh

# Check and update the 'aide-pi' repository
check_updated () {
  cd /home/pi/lab > /dev/null
  echo Checking \`aide-pi\` latest version...
  /usr/bin/git pull origin --quiet
}

# build the 'raspi-lab' executable
build_raspi_lab () {
  echo "``raspi-lab`` compiling..."
  go get . > /dev/null
  go build -o /home/pi/lab/bin/raspi-lab . > /dev/null
}

# Adjusts Raspberry Pi backlight intensity using vcgencmd based on command-line arguments.
if [[ "$1" == "--blacklight" ]]; then
  if [[ "$2" == "off" ]]; then
    for v in 255 239 223 207 191 175 159 143 127 111 95 79 63 47 31 15 0; do
      vcgencmd set_backlight $v > /dev/null
      sleep 0.1;
    done;
  else
    for v in 0 15 31 47 63 79 95 111 127 143 159 175 191 207 223 239 255 ; do
      vcgencmd set_backlight $v > /dev/null
      sleep 0.2;
    done;
  fi
  exit 0
fi

if [[ "$1" == "--run" ]]; then
  build_raspi_lab
  /home/pi/lab/bin/raspi-lab --tty $(tty)
fi

# Conditional script to run 'raspi-lab' on the /dev/tty1 terminal.
if [[ $(tty) == /dev/tty1 && -f "/home/pi/lab/bin/raspi-lab" ]]; then
  check_updated
  build_raspi_lab
  /home/pi/lab/bin/raspi-lab --tty $(tty) --db
fi
