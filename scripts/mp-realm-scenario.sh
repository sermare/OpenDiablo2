#!/bin/bash
# Multiplayer gameplay through the realm, as separate processes, without game windows:
# cmd/od2server hosts the game (authoritative simulation, package d2mp) and N cmd/od2mpbot processes
# join it over TCP with the d2gs protocol: party, walk, exit portal, a fight against the level's
# monsters, drops, experience. Each bot prints a digest of its replica of the level; the scenario
# passes when all digests are equal and every bot saw the same deaths and drops.
#
#   scripts/mp-realm-scenario.sh [bots]        (default 3, at most 8)
#
# Needs no game files and opens no window (so it does not count against the 2-window cap).
set -u
n=${1:-3}
here=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d "${TMPDIR:-/tmp}/od2-mp-realm.XXXXXX")
cleanup() { for p in $pids; do kill "$p" 2>/dev/null; done; rm -rf "$tmp"; }
pids=""
trap cleanup EXIT

(cd "$here" && go build -o "$tmp/od2server" ./cmd/od2server && go build -o "$tmp/od2mpbot" ./cmd/od2mpbot) 2>&1 | grep -v "duplicate libraries"
[ -x "$tmp/od2server" ] && [ -x "$tmp/od2mpbot" ] || { echo "FAIL: build"; exit 1; }

port=$((20000 + RANDOM % 20000))
"$tmp/od2server" -listen 127.0.0.1:$port -saves "$tmp/saves" -hero-life 3000 >"$tmp/server.log" 2>&1 &
pids="$!"
sleep 1

names=(Alice Bobby Carol Dwarf Edgar Frank Grace Helen)
"$tmp/od2mpbot" -addr 127.0.0.1:$port -name ${names[0]} -game bots -host -expect "$n" >"$tmp/bot0.log" 2>&1 &
pids="$pids $!"
i=1
while [ $i -lt "$n" ]; do
  "$tmp/od2mpbot" -addr 127.0.0.1:$port -name ${names[$i]} -game bots -expect "$n" >"$tmp/bot$i.log" 2>&1 &
  pids="$pids $!"
  i=$((i + 1))
done

# wait for the bots (not the server)
for p in $(echo $pids | cut -d' ' -f2-); do wait "$p"; done

fail=0
digests=""
for i in $(seq 0 $((n - 1))); do
  cat "$tmp/bot$i.log"
  r=$(grep RESULT "$tmp/bot$i.log")
  [ -n "$r" ] || { echo "FAIL: bot $i printed no result"; fail=1; continue; }
  digests="$digests
$(echo "$r" | sed 's/.*digest=\([0-9a-f]*\).*/\1/')"
  echo "$r" | grep -q "level=2" || { echo "FAIL: bot $i not in level 2"; fail=1; }
  echo "$r" | grep -q "party=[1-9]" || { echo "FAIL: bot $i not in a party"; fail=1; }
  echo "$r" | grep -q "heroes=$n" || { echo "FAIL: bot $i does not see $n heroes"; fail=1; }
done

[ "$(echo "$digests" | grep -v '^$' | sort -u | wc -l | tr -d ' ')" = 1 ] || { echo "FAIL: the bots' worlds differ:$digests"; fail=1; }
[ "$(grep -h RESULT "$tmp"/bot*.log | sed 's/.*deaths_seen=\([0-9]*\).*/\1/' | sort -u | wc -l | tr -d ' ')" = 1 ] || { echo "FAIL: bots saw different numbers of deaths"; fail=1; }
grep -h RESULT "$tmp"/bot*.log | grep -q "deaths_seen=0 " && { echo "FAIL: no monster died"; fail=1; }

if [ $fail = 0 ]; then echo "mp-realm-scenario: PASS ($n processes, same world)"; else echo "--- server log"; tail -20 "$tmp/server.log"; exit 1; fi
