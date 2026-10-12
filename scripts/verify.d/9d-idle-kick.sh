scenario_realtime=1
scenario_warnings_ok=1
scenario_name="idle kick (two processes over TCP with the d2gs protocol: the joiner falls silent, the host's server kicks it after the short test limit OD2_IDLE_KICK_MS and reports the departure)"
# N19 (Game.exe 0x52adc0): a client that sent no message for longer than the limit is removed. The exe's limit is 45 s
# and its sweep is gated by two globals of unknown meaning, so the Go sweep is opt-in; this scenario turns it on with
# the test-only limit OD2_IDLE_KICK_MS=6000 on the host. The joiner connects, then does nothing but wait.
mp_second_name=NokkaIdle
mp_join_log=$tmp/9d-idle-join.log

_idle_name() { python3 -I -c 'import sys;b=open(sys.argv[1],"rb").read();print(b[0x14:0x24].split(b"\0")[0].decode())' "$1"; }

_idle_rename() {
  python3 -I -c '
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
  mp_host_name=$(_idle_name $save)
  local jn=$mp_second_name jsave=$tmp/9d-idle-join.d2s jcmd=$tmp/9d-idle-join.command
  _idle_rename $save $jsave $jn

  local common="export OD2_PORT=$OD2_PORT OD2_PROTO=d2gs ${OD2_VERIFY_MUTE_ENV} OD2_AUTOEXIT=1 OD2_D2S_WRITEBACK=$tmp"
  # waitlog (not a fixed wait:20): the host waits for the roster notice of the departure, so a slow joiner or a stalled
  # frame loop on a loaded machine cannot make the script end before the kick (waitlog tests the whole captured log,
  # so a kick that landed before the step started is seen as well). The "IDLE KICK" line itself is a Game Server log
  # line, which the script's log capture does not see (an earlier waitlog:IDLE KICK always timed out); the notice
  # comes through the Game Screen logger. The scenario_check reads the kick line from the process log.
  local hscript="wait:2;waitlog:SOCIAL roster n=2;waitlog:notice=\"$jn left the game\";wait:1;say:roster;exit"
  local jscript="wait:1;waitlog:SOCIAL roster n=2;wait:40;exit"

  {
    echo '#!/bin/zsh'
    echo "$common OD2_JOIN=127.0.0.1:$OD2_PORT OD2_JOIN_RETRY=120"
    [ -n "${OD2_VERIFY_CONFIG_DIR:-}" ] && echo "export OD2_CONFIG_DIR=\"$OD2_VERIFY_CONFIG_DIR\""
    echo "export OD2_AUTOGAME=\"$jsave\""
    echo "export OD2_AUTOSCRIPT='$jscript'"
    # the host imports its save first (two imports at once pick the same hero file name): wait until the host's server is up
    # (its log has the line), not a fixed 25 s, which a loaded machine overran (the joiner then collided with the host's import
    # and never joined: "log did not contain SOCIAL roster n=2")
    echo "for i in {1..300}; do grep -q 'Starting Game Server' $log 2>/dev/null && break; sleep 1; done; sleep 3"
    echo "$tmp/od2 2>&1 | tee $mp_join_log"
  } > $jcmd
  chmod +x $jcmd; rm -f $mp_join_log
  launch_game $jcmd

  echo "$common OD2_HOST=1 OD2_BIND=127.0.0.1 OD2_IDLE_KICK_MS=6000"
  echo "export OD2_AUTOSCRIPT='$hscript'"
}

scenario_check() {
  local jn=$mp_second_name
  grep -E "IDLE KICK|PLAYER LEAVE|SOCIAL roster|AUTOSCRIPT RESULT" $log.txt | cut -c1-200

  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: host script did not pass"; fail=1; }
  grep -q "IDLE KICK name=\"$jn\"" $log.txt || { echo "FAIL: the silent joiner was not kicked"; fail=1; }
  grep -q "notice=\"$jn left the game\"" $log.txt || { echo "FAIL: the kick was not reported to the roster"; fail=1; }
  grep 'SOCIAL roster' $log.txt | tail -1 | grep -q 'n=1 ' || { echo "FAIL: host still lists the kicked joiner"; fail=1; }
  # the host (local client) is never kicked
  grep -q "IDLE KICK name=\"$mp_host_name\"" $log.txt && { echo "FAIL: the host was kicked"; fail=1; }
}
