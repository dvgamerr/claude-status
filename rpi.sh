#!/bin/bash
if [ -f "/usr/bin/git" ]; then
  pushd /home/pi/lab > /dev/null
  /usr/bin/git pull origin
fi

if [[ $(tty) = /dev/tty1 ]]; then
  echo " -- First run of the script. Performing some actions --"
fi
