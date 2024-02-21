#!/bin/bash
HOST=10-203-1-200
CRON=https://cronitor.link/p/41a6b95a358f4a64930be3aa6f1ed106/luEI0y
TEMP=$(/usr/bin/vcgencmd measure_temp | grep  -o -E '[[:digit:].]+')
/usr/bin/curl -fsS -m 10 --retry 5 -X GET "$CRON?host=$HOST&metric=count:1&metric=duration:$TEMP"