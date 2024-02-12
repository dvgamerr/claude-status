#!/usr/bin/bash

echo Running \`aide-pi\`...

pushd /home/pi/lab > /dev/null

url=http://localhost:21280
rpi=raspi-lab

# Check and update the 'aide-pi' repository
check_updated () {
  echo Checking \`aide-pi\` latest version...
  /usr/bin/git pull origin
}

# build the '$rpi' executable
build_raspi_lab () {
  cd /home/pi/lab > /dev/null
  echo "``$rpi`` compiling..."
  go get ./raspi-lab/ > /dev/null
  go build -o /home/pi/lab/bin/$rpi ./raspi-lab/ > /dev/null
}

if [[ "$1" == "--update" ]]; then
  check_updated
  build_raspi_lab
  exit 0
fi

if [[ "$1" == "--reload" ]]; then
  git checkout .
  curl -s -o /dev/null -X PUT $url/_exit
  echo Signal exiting...
fi

if [[ "$1" == "--run" ]]; then
  build_raspi_lab
  /home/pi/lab/bin/$rpi --tty $(tty)
fi

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

# Function to handle CTRL+C and set exit flag
ctrl_c() {
  echo "CTRL+C detected. Exiting..."
  eof=true
}

# Set up trap to call ctrl_c function on CTRL+C
trap ctrl_c INT

# Flag to control loop exit
eof=false

while :
do
  # Conditional script to run '$rpi' on the /dev/tty1 terminal.
  if [[ $(tty) == /dev/tty1 && -f "/home/pi/lab/bin/$rpi" ]]; then
    check_updated
    build_raspi_lab
    /home/pi/lab/bin/$rpi --tty $(tty) --db
    reset

    # Check if the exit flag is set
    if [ "$eof" = true ]; then
      break  # Exit the loop
    fi
  else
    echo a "$(tty)" not is tty1 or not build raspi-lab.
    break
  fi
done
