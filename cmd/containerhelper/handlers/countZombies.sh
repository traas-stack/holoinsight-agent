#!/bin/sh
set -e

# docs: count zombies processes
COUNT_LIMIT=100

count=0
for i in /proc/*; do
  b=`basename $i`
  if [ -e "$i/status" ]; then
    if grep 'State:	Z (zombie)' "$i/status" >/dev/null 2>&1; then
      count=`expr $count + 1`
      if [ "$count" -gt "$COUNT_LIMIT" ]; then
        break
      fi
    fi
  fi
done

echo $count
