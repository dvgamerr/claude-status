#!/bin/bash

# Push directory to /home/pi/lab silently
pushd /home/pi/lab > /dev/null

# Define constants
URL="http://localhost:21280"
RPI="raspi-lab"

# Function to check and update the 'aide-pi' repository
check_updated() {
  echo "Checking \`aide-pi\` latest version..."
  git checkout .
  git pull origin
}

# Function to build the '$rpi' executable
build_raspi_lab() {
  cd /home/pi/lab > /dev/null
  echo "Compiling $RPI..."
  go get ./raspi-lab/ > /dev/null
  go build -o /home/pi/lab/bin/$RPI -buildvcs=false ./raspi-lab/ > /dev/null
}

# Handle different actions based on arguments
case "$1" in
  "--update")
    check_updated
    build_raspi_lab
    exit 0
    ;;
  "--reload")
    git checkout .
    curl -s -o /dev/null -X PUT "$URL/_exit"
    echo "Signal exiting..."
    ;;
  "--run")
    build_raspi_lab
    /home/pi/lab/bin/$RPI --tty $(tty)
    ;;
  "--blacklight")
    if [[ "$2" == "off" ]]; then
      for v in $(seq 255 15 -0); do
        vcgencmd set_backlight $v > /dev/null
        sleep 0.1
      done
    else
      for v in $(seq 0 15 255); do
        vcgencmd set_backlight $v > /dev/null
        sleep 0.2
      done
    fi
    exit 0
    ;;
  *)
    echo "Invalid argument. Usage: $0 [--update, --reload, --run, --blacklight [on|off]]"
    exit 1
    ;;
esac

# Function to handle CTRL+C and set exit flag
ctrl_c() {
  echo "CTRL+C detected. Exiting..."
  eof=true
}

# Exit on errors
if_err_exit() {
  if [ $? -ne 0 ]; then
    exit 1
  fi
}

# Set trap to call ctrl_c function on CTRL+C
trap ctrl_c INT

# Flag to control loop exit
eof=false

while true; do
  # Run script on /dev/tty1 if executable exists
  if [[ $(tty) == /dev/tty1 && -f "/home/pi/lab/bin/$RPI" ]]; then
    check_updated
    build_raspi_lab
    if_err_exit

    /home/pi/lab/bin/$RPI --tty $(tty) --db
    if_err_exit

    reset

    # Check if exit flag is set
    if [ "$eof" = true ]; then
      break
    fi
  else
    echo "$(tty)" is not /dev/tty1 or raspi-lab not built.
    break
  fi
done

popd  # Pop directory back
