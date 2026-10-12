scenario_realtime=1
scenario_name="server-owned ground items (two processes over TCP: a drop is shared with an id, both heroes race for the item and for a gold pile, exactly one wins each)"
# The runner starts the host with $save; this scenario starts the joiner (a copy of the save under another hero
# name; chat underscores show as spaces in the log). Nothing is clicked: `groundrace` sends the pickup request at once, so both processes ask the server for
# the same item at the same moment (no walking, no timing luck in the walk). Sync points are chat lines.
#   host: spawnground hp1 -> joiner shows it (GROUND shown) -> joiner chats ready_item -> both race
#   host: spawnground gold 300 -> the same with ready_gold
# The server (in the host's log) keeps one winner per id: one "GROUND pickup id=" per pile, one "AUTOGROUND pickup"
# and one "PARTYGOLD award of=300" across both logs; the loser either gets "GROUND pickup denied" or already saw the
# item removed ("GROUNDRACE none"). The dedup of a drop both clients roll themselves is covered by the d2server tests.
mp_second_name=NokkaGround
mp_join_log=$tmp/9j-join.log

_d2s_name() { python3 -c 'import sys;b=open(sys.argv[1],"rb").read();print(b[0x14:0x24].split(b"\0")[0].decode())' "$1"; }

_d2s_rename() {
  python3 -c '
import sys
b = bytearray(open(sys.argv[1], "rb").read())
b[0x14:0x24] = sys.argv[3].encode().ljust(16, b"\0")
b[0x0c:0x10] = b"\0\0\0\0"
s = 0
for x in b:
    s = ((((s << 1) | (s >> 31)) & 0xffffffff) + x) & 0xffffffff
b[0x0c:0x10] = s.to_bytes(4, "little")
open(sys.argv[2], "wb").write(b)' "$1" "$2" "$3"
}

scenario_env() {
  mp_host_name=$(_d2s_name $save)
  local jn=$mp_second_name hn=$mp_host_name jsave=$tmp/9j-join.d2s jcmd=$tmp/9j-join.command
  _d2s_rename $save $jsave $jn

  local common="export OD2_PORT=$OD2_PORT OD2_PROTO=d2gs ${OD2_VERIFY_MUTE_ENV} OD2_AUTOEXIT=1 OD2_D2S_WRITEBACK=$tmp"

  local hscript="wait:2;waitlog:SOCIAL roster n=2;say:spawnground hp1 4 4;until:AUTOGROUND spawn source=command,30"
  hscript+=";waitlog:CHAT <$jn> ready item;say:groundrace item;wait:3"
  hscript+=";say:spawnground gold 300 5;until:AUTOGROUND spawn source=command name=\"300,30"
  hscript+=";waitlog:CHAT <$jn> ready gold;say:groundrace gold;wait:3;say:chat done;waitlog:CHAT <$jn> done too;wait:4;exit"

  local jscript="wait:1;waitlog:SOCIAL roster n=2;waitlog:GROUND shown id=;say:chat ready_item;say:groundrace item;wait:3"
  jscript+=";waitlog:kind=gold;say:chat ready_gold;say:groundrace gold;wait:3"
  jscript+=";waitlog:CHAT <$hn> done;say:chat done_too;exit"

  {
    echo '#!/bin/zsh'
    echo "$common OD2_JOIN=127.0.0.1:$OD2_PORT OD2_JOIN_RETRY=120"
    [ -n "${OD2_VERIFY_CONFIG_DIR:-}" ] && echo "export OD2_CONFIG_DIR=\"$OD2_VERIFY_CONFIG_DIR\""
    echo "export OD2_AUTOGAME=\"$jsave\""
    echo "export OD2_AUTOSCRIPT='$jscript'"
    echo "for i in {1..300}; do grep -q 'Starting Game Server' $log 2>/dev/null && break; sleep 1; done; sleep 3"
    echo "$tmp/od2 2>&1 | tee $mp_join_log"
  } > $jcmd
  chmod +x $jcmd; rm -f $mp_join_log
  launch_game $jcmd

  echo "$common OD2_HOST=1 OD2_BIND=127.0.0.1"
  echo "export OD2_AUTOSCRIPT='$hscript'"
}

scenario_check() {
  local j=$mp_join_log.txt hn=$mp_host_name jn=$mp_second_name
  [ -f "$mp_join_log" ] && sed 's/\x1b\[[0-9;]*m//g' $mp_join_log > $j
  if [ ! -f "$j" ]; then echo "FAIL: no joiner log"; fail=1; return; fi

  grep -E "GROUND|PARTYGOLD|AUTOSCRIPT RESULT" $log.txt | sed 's/^/host   /' | cut -c1-200
  grep -E "GROUND|PARTYGOLD|AUTOSCRIPT RESULT" $j | sed 's/^/joiner /' | cut -c1-200

  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: host script did not pass"; fail=1; }
  grep -q "AUTOSCRIPT RESULT PASS" $j || { echo "FAIL: joiner script did not pass"; fail=1; }

  # the server registered the two drops once each and the joiner saw both with the server's id
  [ "$(grep -c 'GROUND drop id=' $log.txt)" = 2 ] || { echo "FAIL: server should hold exactly 2 drops"; fail=1; }
  grep -q "GROUND shown id=.* kind=item" $j || { echo "FAIL: joiner never saw the item"; fail=1; }
  grep -q "GROUND shown id=.* kind=gold" $j || { echo "FAIL: joiner never saw the gold"; fail=1; }

  # one winner per pile, at the server and across the clients
  [ "$(grep -c 'GROUND pickup id=' $log.txt)" = 2 ] || { echo "FAIL: server approved $(grep -c 'GROUND pickup id=' $log.txt) pickups, want 2 (one per pile)"; fail=1; }
  local won; won=$(cat $log.txt $j | grep -c 'GROUND won id=')
  [ "$won" = 2 ] || { echo "FAIL: $won clients won a pickup, want 2 (item and gold, once each)"; fail=1; }
  [ "$(cat $log.txt $j | grep -c 'AUTOGROUND pickup name=')" = 1 ] || { echo "FAIL: the item must reach exactly one cursor"; fail=1; }
  [ "$(cat $log.txt $j | grep -c 'PARTYGOLD award add=300 of=300')" = 1 ] || { echo "FAIL: the 300 gold must be paid exactly once"; fail=1; }
  if grep -E "\[(ERROR|WARNING)\]|panic" $j | grep -v "skipping missing"; then echo "FAIL: errors in the joiner log"; fail=1; fi
}
