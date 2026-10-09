#!/bin/zsh
# Global cap on simultaneously running game windows (all sessions/agents share /tmp/od2-slots).
#   gameslot.sh acquire   blocks until a slot is free, prints the slot dir; holder = the caller's pid ($PPID)
#   gameslot.sh release <slot-dir>
# OD2_MAX_GAMES (default 99 = effectively no cap; cleanup is done by killing stuck games, see reap_games.sh) sets the cap. Slots whose owner process is gone are reclaimed.
root=/tmp/od2-slots; max=${OD2_MAX_GAMES:-99}; mkdir -p $root
case "$1" in
acquire)
  owner=${2:-$PPID}
  while true; do
    for i in $(seq 1 $max); do
      d=$root/$i
      if mkdir $d 2>/dev/null; then echo $owner > $d/owner; echo $d; exit 0; fi
      o=$(cat $d/owner 2>/dev/null)
      if [ -n "$o" ] && ! kill -0 $o 2>/dev/null; then rm -rf $d; fi
    done
    sleep 3
  done;;
release) rm -rf "$2";;
*) echo "usage: gameslot.sh acquire [owner-pid] | release <dir>" >&2; exit 2;;
esac
