scenario_name="town portals (two processes over TCP: a portal pair opens in the field and in town, outsiders are refused, party members and the owner use it, it survives level changes, recasting replaces it, leaving closes it)"
# The runner starts the host with $save (a real level 94 hero); this scenario starts the joiner itself, with a copy
# of the same save under another hero name. Both scripts run in step with `waitlog:` and chat markers:
#   1 host walks through a spawned portal to the Cold Plains and casts a town portal (a tome charge): the pair is
#     one portal beside the hero and one in the Rogue Encampment, where the joiner (still in town) sees it
#   2 the joiner is not in the host's party: its use of the portal is refused
#   3 party formed; the joiner uses the host's portal and arrives beside the field end
#   4 the host walks back to town through its portal (5 s cooldown refuses an immediate second jump) and returns
#     to the Cold Plains through the town end: the pair outlived both level changes
#   5 the host casts again: the first pair is replaced (one pair per owner); the joiner casts its own, then leaves,
#     which closes its pair
# UNVERIFIED rules (d2common/d2portal): one pair per owner, replacement on recast, closing when the owner leaves,
# the town end next to the waypoint. VERIFIED (session-core.md): 5 s cooldown, owner/party rule, arrival with start
# type 3 beside the other portal, both ends belong to the same pair.
mp_second_name=NokkaTwin
mp_join_log=$tmp/9g-join.log

_9g_name() { python3 -I -c 'import sys;b=open(sys.argv[1],"rb").read();print(b[0x14:0x24].split(b"\0")[0].decode())' "$1"; }

_9g_rename() {
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
  mp_host_name=$(_9g_name $save)
  local jn=$mp_second_name hn=$mp_host_name jsave=$tmp/9g-join.d2s jcmd=$tmp/9g-join.command
  _9g_rename $save $jsave $jn

  local common="export OD2_REALMAPS=1 OD2_PORT=$OD2_PORT OD2_PROTO=d2gs OD2_AUTOPARTY=1 ${OD2_VERIFY_MUTE_ENV} OD2_AUTOEXIT=1 OD2_D2S_WRITEBACK=$tmp"

  local hscript="wait:2;waitlog:SOCIAL roster n=2;say:spawnportal 3;use:Portal;waitlog:LEVEL built: level 3"
  hscript+=";wait:2;say:townportal;waitlog:PORTAL object level=3;say:chat tp_open"
  hscript+=";waitlog:CHAT <$jn> refused_ok"
  hscript+=";say:party invite $jn;waitlog:$jn joined the party"
  hscript+=";waitlog:CHAT <$jn> arrived"
  hscript+=";wait:6;use:Portal;waitlog:LEVEL built: level 1;use:Portal;wait:7;say:chat host_in_town;use:Portal;waitlog:PORTAL used owner=\"$hn\" from=1 dest=3"
  hscript+=";until:LEVEL built: level 3,20;wait:3;say:pvpwalk 4 0;wait:3;say:townportal;waitlog:replaced=true;say:chat host_recast"
  hscript+=";waitlog:PORTAL closed owner=\"$jn\";wait:2;say:portals;exit"

  local jscript="wait:1;waitlog:SOCIAL roster n=2;waitlog:CHAT <$hn> tp_open;wait:1;say:portals;use:Portal;wait:2;say:chat refused_ok"
  jscript+=";waitlog:$hn invites you to a party;say:party accept -;waitlog:$jn joined the party"
  jscript+=";use:Portal;waitlog:LEVEL built: level 3;wait:2;say:chat arrived"
  jscript+=";waitlog:CHAT <$hn> host_recast;wait:2;say:portals;say:townportal free;wait:3;say:portals;exit"

  {
    echo '#!/bin/zsh'
    echo "$common OD2_JOIN=127.0.0.1:$OD2_PORT OD2_JOIN_RETRY=120"
    echo "export OD2_AUTOGAME=\"$jsave\""
    echo "export OD2_AUTOSCRIPT='$jscript'"
    echo "sleep 25  # the host imports its save first (two imports at once pick the same hero file name)"
    echo "$tmp/od2 2>&1 | tee $mp_join_log"
  } > $jcmd
  chmod +x $jcmd; rm -f $mp_join_log
  launch_game $jcmd

  echo "$common OD2_HOST=1 OD2_BIND=127.0.0.1"
  echo "export OD2_AUTOSCRIPT='$hscript'"
}

# _9g_dist <ax> <ay> <bx> <by>: the distance in sub-tiles
_9g_dist() { awk -v ax=$1 -v ay=$2 -v bx=$3 -v by=$4 'BEGIN { dx = ax - bx; dy = ay - by; printf "%d", sqrt(dx * dx + dy * dy) }'; }

scenario_check() {
  local j=$mp_join_log.txt hn=$mp_host_name jn=$mp_second_name
  [ -f "$mp_join_log" ] && sed 's/\x1b\[[0-9;]*m//g' $mp_join_log > $j
  if [ ! -f "$j" ]; then echo "FAIL: no joiner log"; fail=1; return; fi

  local pat="PORTAL|LEVEL CHANGE|AUTOSCRIPT RESULT"
  grep -E "$pat" $log.txt | sed 's/^/host   /' | cut -c1-260
  grep -E "$pat" $j | sed 's/^/joiner /' | cut -c1-260

  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: host script did not pass"; fail=1; }
  grep -q "AUTOSCRIPT RESULT PASS" $j || { echo "FAIL: joiner script did not pass"; fail=1; }

  # 1 casting opens a pair, tied to the creator: a field end beside the hero and a town end
  grep -qE "PORTAL cast level=3 .* town=1 " $log.txt || { echo "FAIL: host did not cast a town portal in level 3"; fail=1; }
  grep -qE "PORTAL paid with (tbk|tsc)" $log.txt || { echo "FAIL: the cast was not paid with a scroll or a tome charge"; fail=1; }
  grep -qE "PORTAL opened owner=\"$hn\" id=1 field=3 town=1 replaced=false" $log.txt || { echo "FAIL: server did not store the host's pair"; fail=1; }
  grep -qE "PORTAL object level=3 .* owner=\"$hn\" dest=1 .* town=false at=" $log.txt || { echo "FAIL: host sees no field portal in level 3"; fail=1; }
  grep -qE "PORTAL object level=1 .* owner=\"$hn\" dest=3 .* town=true at=" $j || { echo "FAIL: joiner sees no town portal in the Rogue Encampment"; fail=1; }

  # 2 an outsider is refused
  grep -q "PORTAL refused: portal: it belongs to a player outside your party" $j || { echo "FAIL: a non-party hero was not refused"; fail=1; }

  # 3 a party member uses it and arrives beside the field end
  grep -qE "PORTAL used owner=\"$hn\" from=1 dest=3 .* mine=false" $j || { echo "FAIL: the party member did not use the host's portal"; fail=1; }
  grep -qE "LEVEL CHANGE from=1 to=3 .* via=portal " $j || { echo "FAIL: no portal level change for the joiner"; fail=1; }
  local ob hero d
  ob=$(sed -nE 's/.*PORTAL object level=3 .* owner="[^"]*" dest=1 .* at=\(([0-9]+),([0-9]+)\).*/\1 \2/p' $j | head -1)
  hero=$(sed -nE 's/.*LEVEL CHANGE from=1 to=3 .* hero=\(([0-9.]+),([0-9.]+)\).*/\1 \2/p' $j | head -1)
  if [ -n "$ob" ] && [ -n "$hero" ]; then
    d=$(awk -v o="$ob" -v h="$hero" 'BEGIN { split(o, a, " "); split(h, b, " "); dx = a[1] / 5 - b[1]; dy = a[2] / 5 - b[2]; printf "%.1f", sqrt(dx * dx + dy * dy) }')
    awk -v d=$d 'BEGIN { exit !(d <= 4) }' || { echo "FAIL: the joiner arrived $d tiles from the portal"; fail=1; }
  else
    echo "FAIL: cannot compare the arrival with the portal (object '$ob' hero '$hero')"; fail=1
  fi

  # 4 the owner's cooldown, and the pair outlives level changes (both ends used, the objects rebuilt)
  grep -q "PORTAL refused: portal: less than 5 s since the last level change" $log.txt || { echo "FAIL: no 5 s cooldown after a portal jump"; fail=1; }
  grep -qE "PORTAL used owner=\"$hn\" from=3 dest=1 .* mine=true" $log.txt || { echo "FAIL: host did not go to town through its portal"; fail=1; }
  grep -qE "PORTAL used owner=\"$hn\" from=1 dest=3 .* mine=true" $log.txt || { echo "FAIL: host did not return through the town end"; fail=1; }
  [ "$(grep -cE "PORTAL object level=(1|3) .* owner=\"$hn\"" $log.txt)" -ge 4 ] || { echo "FAIL: the portal objects were not rebuilt after the level changes"; fail=1; }
  grep -q "LEVEL portal state" $log.txt || { echo "FAIL: no portal arrival state"; fail=1; }

  # 5 recasting replaces the pair; the joiner sees the new one; leaving closes the joiner's pair
  grep -qE "PORTAL opened owner=\"$hn\" id=[0-9]+ field=3 town=1 replaced=true" $log.txt || { echo "FAIL: recast did not replace the old pair"; fail=1; }
  grep -qE "PORTAL opened owner=\"$jn\" " $log.txt || { echo "FAIL: the joiner's own portal was not stored"; fail=1; }
  grep -q "PORTAL closed owner=\"$jn\" (left the game)" $log.txt || { echo "FAIL: the joiner's pair stayed open after it left"; fail=1; }
  [ "$(grep -c "PORTAL pair id=" $log.txt)" -ge 1 ] && grep "PORTAL list level" $log.txt | tail -1 | grep -q "pairs=1" || { echo "FAIL: host does not end with exactly its own pair"; fail=1; }
  if grep -E "\[(ERROR|WARNING)\]|panic" $j | grep -v "skipping missing"; then echo "FAIL: errors in the joiner log"; fail=1; fi
}
