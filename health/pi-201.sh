#!/bin/bash
HOST=10-203-1-201
CRON=https://cronitor.link/p/41a6b95a358f4a64930be3aa6f1ed106/3wzIAF
TEMP=$(/bin/sensors -Aj bigcore0_thermal-virtual-0 | jq '.["bigcore0_thermal-virtual-0"].temp1.temp1_input')
/usr/bin/curl -fsS -m 10 --retry 5 -X GET "$CRON?host=$HOST&metric=count:1&metric=duration:$TEMP" -o /dev/null