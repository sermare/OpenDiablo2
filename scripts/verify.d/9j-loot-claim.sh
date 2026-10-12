scenario_realtime=1
scenario_name="single-roll loot (two processes claim one loot source at once: the server grants one roll, the other hero is shown the result)"
# Same two-process layout as 9j-ground-items. Both heroes run `lootroll k1 500` after a chat sync; the server grants the
# claim to one of them only ("LOOT granted"), the other gets "LOOT elsewhere" and is shown the pile by broadcast
# ("GROUND shown ... kind=gold"). Exactly one pile reaches the server (one "GROUND drop id=").
mp_second_name=NokkaLoot
mp_join_log=$tmp/9j-loot-join.log

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
  local jn=$mp_second_name hn=$mp_host_name jsave=$tmp/9j-loot-join.d2s jcmd=$tmp/9j-loot-join.command
  _d2s_rename $save $jsave $jn

  local common="export OD2_PORT=$OD2_PORT OD2_PROTO=d2gs ${OD2_VERIFY_MUTE_ENV} OD2_AUTOEXIT=1 OD2_D2S_WRITEBACK=$tmp"

  local hscript="wait:2;waitlog:SOCIAL roster n=2;waitlog:CHAT <$jn> ready;say:chat go;say:lootroll k1 500;wait:4"
  hscript+=";say:chat done;waitlog:CHAT <$jn> done too;wait:3;exit"

  local jscript="wait:1;waitlog:SOCIAL roster n=2;say:chat ready;waitlog:CHAT <$hn> go;say:lootroll k1 500;wait:4"
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
  local j=$mp_join_log.txt
  [ -f "$mp_join_log" ] && sed 's/\x1b\[[0-9;]*m//g' $mp_join_log > $j
  if [ ! -f "$j" ]; then echo "FAIL: no joiner log"; fail=1; return; fi

  grep -E "LOOT|GROUND|AUTOSCRIPT RESULT" $log.txt | sed 's/^/host   /' | cut -c1-200
  grep -E "LOOT|GROUND|AUTOSCRIPT RESULT" $j | sed 's/^/joiner /' | cut -c1-200

  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: host script did not pass"; fail=1; }
  grep -q "AUTOSCRIPT RESULT PASS" $j || { echo "FAIL: joiner script did not pass"; fail=1; }

  [ "$(cat $log.txt $j | grep -c 'LOOT granted')" = 1 ] || { echo "FAIL: exactly one client must be granted the roll"; fail=1; }
  [ "$(cat $log.txt $j | grep -c 'LOOT elsewhere')" = 1 ] || { echo "FAIL: exactly one client must be told another one rolls"; fail=1; }
  [ "$(grep -c 'GROUND drop id=' $log.txt)" = 1 ] || { echo "FAIL: the server should hold exactly one pile"; fail=1; }
  [ "$(cat $log.txt $j | grep -c 'GROUND shown id=.* kind=gold')" = 1 ] || { echo "FAIL: the non-roller must be shown the pile once"; fail=1; }
  [ "$(cat $log.txt $j | grep -c 'AUTOGROUND spawn source=command')" = 1 ] || { echo "FAIL: the pile must be rolled once"; fail=1; }
  if grep -E "\[(ERROR|WARNING)\]|panic" $j | grep -v "skipping missing"; then echo "FAIL: errors in the joiner log"; fail=1; fi
}
