#!/bin/zsh
# Kill leftover test games: od2 processes started from a verify/scratch folder that are older than N minutes
# (default 20). Never touches the user's own game (those are not under /tmp/od2-verify.* or a scratchpad).
#   scripts/reap_games.sh [minutes]
max=${1:-20}
ps -axo pid=,etime=,command= | grep -E "(/tmp/od2-verify\.[A-Za-z0-9]+|/scratchpad/[^ ]*)/od2( |$)" | grep -v grep | while read pid et cmd; do
  # etime is [[dd-]hh:]mm:ss
  mins=$(echo $et | awk -F'[-:]' '{ n=NF; if (n==2) print $1; else if (n==3) print $1*60+$2; else print ($1*24+$2)*60+$3 }')
  if [ "${mins:-0}" -ge "$max" ]; then echo "reaping pid $pid (age ${mins}m): ${cmd[1,90]}"; kill $pid 2>/dev/null; fi
done
