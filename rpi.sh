#!/bin/bash

USR=dvgamerr
# Push directory to /home/$USR/lab silently
pushd /home/$USR/lab > /dev/null

# Define constants
URL="http://localhost:21280"
RPI="raspi-lab"

# Function to check and update the 'aide-lab' repository
check_updated() {
  echo "Checking \`aide-lab\` latest version..."
  git checkout . > /dev/null
  git pull origin
}

# # Function to build the '$rpi' executable
# build_raspi_lab() {
#   cd /home/$USR/lab > /dev/null
#   echo "Compiling $RPI..."
#   go get ./raspi-lab/ > /dev/null
#   go build -o /home/$USR/lab/bin/$RPI -buildvcs=false ./raspi-lab/ > /dev/null
# }

# # Function to handle CTRL+C and set exit flag
# ctrl_c() {
#   echo "CTRL+C detected. Exiting..."
#   eof=true
# }

# # Exit on errors
if_err_exit() {
  if [ $? -ne 0 ]; then
    exit 1
  fi
}

# # Set trap to call ctrl_c function on CTRL+C
# trap ctrl_c INT

# # Flag to control loop exit
# eof=false

# while true; do
# Check if exit flag is set
if [ "$eof" = true ]; then
  break
fi

# Run script on /dev/tty1 if executable exists
if [[ $(tty) == /dev/tty1 && -f "/home/$USR/lab/bin/$RPI" ]]; then
  check_updated
  # build_raspi_lab
  if_err_exit

  # /home/$USR/lab/bin/$RPI --tty $(tty) --db
  # if_err_exit

  reset
fi
# done

popd  # Pop directory back

# # Handle different actions based on arguments
# case "$1" in
#   "--update")
#     check_updated
#     build_raspi_lab
#     exit 0
#     ;;
#   "--reload")
#     curl -s -o /dev/null -X PUT "$URL/_exit"
#     echo "Signal exiting..."
#     ;;
#   "--run")
#     build_raspi_lab
#     /home/$USR/lab/bin/$RPI --tty $(tty)
#     ;;
#   "--blacklight")
#     if [[ "$2" == "off" ]]; then
#       for v in $(seq 255 15 -0); do
#         vcgencmd set_backlight $v > /dev/null
#         sleep 0.1
#       done
#     else
#       for v in $(seq 0 15 255); do
#         vcgencmd set_backlight $v > /dev/null
#         sleep 0.2
#       done
#     fi
#     exit 0
#     ;;
#   *)
#     echo "Invalid argument. Usage: $0 [--update, --reload, --run, --blacklight [on|off]]"
#     exit 1
#     ;;
# esac
