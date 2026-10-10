scenario_realtime=1
scenario_name="party, trade and PvP (two processes over TCP with the d2gs protocol: party, roster panel, party experience, trade both ways, hostile hits, leave)"
# The runner starts the host with $save (a real level 94 hero); this scenario starts the joiner itself, with a
# copy of the same save under another hero name (the roster tells players apart by id, the scripts by name).
# Both scripts run in step with `waitlog:` steps (they wait for a line the other process caused in their own
# log), so the timeline does not depend on how fast either process starts:
#   1 joiner arrives; host invites; joiner accepts            (party)
#   2 roster panel + automap markers; a swing at a party member is blocked; hostility inside a party is refused
#   3 host kills a monster: the experience is split between both (PARTYXP)
#   4 trade: host gives a large charm + 1000 gold, joiner gives a Town Portal Book + 400 gold (both 1x2 cells:
#     the real hero's inventory is full)
#   5 both leave the party, declare hostility, swing at each other, host makes peace, a swing is blocked
#   6 joiner leaves; the host sees it
# The gold step waits for the server's echo of the added item: an offer update of the other side that arrives in
# between replaces the local offer (PlayerTradeWindow.Update), and the gold packet would then carry no item.
# Protocol details with no verified counterpart (party/trade/PvP packets ride the 0xAE/0x6c tunnel, the
# trade rules, the XP weights) are marked UNVERIFIED in d2common/d2party, d2core/d2playertrade and
# d2networking/d2netpacket/packet_social.go.
mp_second_name=NokkaTwin
mp_join_log=$tmp/9d-join.log

# d2s_name <file>: the hero name of a save
_d2s_name() { python3 -c 'import sys;b=open(sys.argv[1],"rb").read();print(b[0x14:0x24].split(b"\0")[0].decode())' "$1"; }

# _d2s_rename <src> <dst> <name>: a copy of a save under another hero name (checksum fixed up)
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

# _swings <name>: 24 melee swings at a player, a second apart (a level 94 sorceress hits a level 94
# sorceress about one time in five: the hero's attack rating against her defense)
_swings() {
  local i out=""
  for i in {1..24}; do out+="say:pvp $1;wait:1;"; done
  echo "${out%;}"
}

scenario_env() {
  mp_host_name=$(_d2s_name $save)
  local jn=$mp_second_name hn=$mp_host_name jsave=$tmp/9d-join.d2s jcmd=$tmp/9d-join.command
  _d2s_rename $save $jsave $jn

  local common="export OD2_PORT=$OD2_PORT OD2_PROTO=d2gs OD2_AUTOPARTY=1 ${OD2_VERIFY_MUTE_ENV} OD2_AUTOEXIT=1 OD2_D2S_WRITEBACK=$tmp"

  local hscript="wait:2;waitlog:SOCIAL roster n=2;say:roster"
  hscript+=";say:party invite $jn;waitlog:$jn joined the party"
  hscript+=";panel:party;wait:2;say:roster;say:capframe $tmp/9d-host-party.png;automap:stats;panel:close"
  hscript+=";say:pvp $jn;say:hostile $jn 1;wait:1"
  hscript+=";say:spawnmon zombie1;wait:1;say:killnear;waitlog:PARTYXP award"
  hscript+=";say:trade request $jn;waitlog:TRADE open with"
  hscript+=";say:trade add cm2;waitlog:yours=[Hellfire Torch];say:trade gold 1000;waitlog:gold=400 you_accepted;wait:1;say:capframe $tmp/9d-host-trade.png;wait:2;say:trade accept -"
  hscript+=";waitlog:TRADE done;wait:2;say:roster"
  hscript+=";say:party leave -;waitlog:$hn left the party;say:hostile $jn 1;waitlog:$jn is now hostile toward $hn;wait:1"
  hscript+=";$(_swings $jn);wait:2"
  hscript+=";say:hostile $jn 0;waitlog:$hn is now at peace toward $jn;say:pvp $jn;wait:1"
  hscript+=";waitlog:$jn left the game;say:roster;exit"

  local jscript="wait:1;waitlog:SOCIAL roster n=2;waitlog:$hn invites you to a party;say:party accept -;waitlog:$jn joined the party"
  jscript+=";panel:party;wait:2;say:roster;say:capframe $tmp/9d-joiner-party.png;automap:stats;panel:close"
  jscript+=";say:pvp $hn;waitlog:PARTYXP award"
  jscript+=";waitlog:TRADE request from;say:trade yes -;waitlog:TRADE open with"
  jscript+=";say:trade add tbk;waitlog:yours=[Tome of Town Portal];say:trade gold 400;waitlog:gold=1000 you_accepted;wait:1;say:capframe $tmp/9d-joiner-trade.png;wait:2;say:trade accept -"
  jscript+=";waitlog:TRADE done;wait:2;say:roster"
  jscript+=";waitlog:$hn left the party;say:hostile $hn 1;waitlog:$hn is now hostile toward $jn;wait:1"
  jscript+=";$(_swings $hn);wait:2"
  jscript+=";waitlog:$hn is now at peace toward $jn;wait:1;exit"

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

# _nth_num <file> <regex-with-one-group>: the numbers the group captured, one per line
_9d_nums() { sed -nE "s/.*$2.*/\1/p" $1; }

scenario_check() {
  local j=$mp_join_log.txt hn=$mp_host_name jn=$mp_second_name
  [ -f "$mp_join_log" ] && sed 's/\x1b\[[0-9;]*m//g' $mp_join_log > $j
  if [ ! -f "$j" ]; then echo "FAIL: no joiner log"; fail=1; return; fi

  local pat="PARTY (invite|join|leave|hostile|peace|refused)|SOCIAL roster|ROSTER PANEL|AUTOMAP marker other|PARTYXP|PVP |TRADE |AUTOSCRIPT RESULT"
  grep -E "$pat" $log.txt | sed 's/^/host   /' | cut -c1-260
  grep -E "$pat" $j | sed 's/^/joiner /' | cut -c1-260

  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: host script did not pass"; fail=1; }
  grep -q "AUTOSCRIPT RESULT PASS" $j || { echo "FAIL: joiner script did not pass"; fail=1; }

  # 1/2 party formed; both roster panels list the other with name, class, level and the party state
  grep -q "PARTY join name=\"$jn\"" $log.txt || { echo "FAIL: server did not log $jn joining the party"; fail=1; }
  grep -qE "ROSTER PANEL rows open=true \[$jn [A-Za-z]+ L94 party\]" $log.txt || { echo "FAIL: host roster panel does not list $jn as party member"; fail=1; }
  grep -qE "ROSTER PANEL rows open=true \[$hn [A-Za-z]+ L94 party\]" $j || { echo "FAIL: joiner roster panel does not list $hn as party member"; fail=1; }
  grep -qE "AUTOMAP marker other-player .*name=\"$jn\" party=true" $log.txt || { echo "FAIL: automap does not mark $jn as a party member (host)"; fail=1; }
  grep -qE "AUTOMAP marker other-player .*name=\"$hn\" party=true" $j || { echo "FAIL: automap does not mark $hn as a party member (joiner)"; fail=1; }

  # no friendly fire, no hostility inside a party
  grep -q "PVP BLOCKED target=\"$jn\" reason=\"party members do not hurt each other\"" $log.txt || { echo "FAIL: swing at a party member was not blocked (host)"; fail=1; }
  grep -q "PVP BLOCKED target=\"$hn\" reason=\"party members do not hurt each other\"" $j || { echo "FAIL: swing at a party member was not blocked (joiner)"; fail=1; }
  grep -q "PARTY refused op=hostile name=\"$hn\"" $log.txt || { echo "FAIL: hostility inside a party was not refused"; fail=1; }

  # 3 party experience: the server scales each member's share by the member's own level against the monster's
  # (VERIFIED), so the shares no longer add up to the kill's raw experience: each is positive and the sum does not exceed it
  local xp hxp jxp
  xp=$(_9d_nums $log.txt 'PARTYXP kill monster="[^"]*" xp=([0-9]+) sent' | head -1)
  hxp=$(_9d_nums $log.txt 'PARTYXP award amount=([0-9]+) of=' | head -1)
  jxp=$(_9d_nums $j 'PARTYXP award amount=([0-9]+) of=' | head -1)
  { [ -n "$xp" ] && [ "$xp" -gt 0 ] && [ -n "$hxp" ] && [ -n "$jxp" ] && [ "$hxp" -gt 0 ] && [ "$jxp" -gt 0 ] && [ $((hxp + jxp)) -le "$xp" ]; } || { echo "FAIL: party experience: kill=$xp host=$hxp joiner=$jxp"; fail=1; }
  grep -q "PARTYXP kill monster=.*shares=\[$hn=$hxp $jn=$jxp\]\|PARTYXP kill monster=.*shares=\[$jn=$jxp $hn=$hxp\]" $log.txt || { echo "FAIL: server share log differs"; fail=1; }

  # 4 trade: item and gold both ways, with the gold totals before and after
  local hline jline hb ha jb ja
  hline=$(grep "TRADE done with=\"$jn\"" $log.txt | head -1); jline=$(grep "TRADE done with=\"$hn\"" $j | head -1)
  echo "host   $hline" | cut -c1-260; echo "joiner $jline" | cut -c1-260
  hb=$(echo "$hline" | sed -nE 's/.*before=([0-9]+) after=([0-9]+).*/\1/p'); ha=$(echo "$hline" | sed -nE 's/.*before=([0-9]+) after=([0-9]+).*/\2/p')
  jb=$(echo "$jline" | sed -nE 's/.*before=([0-9]+) after=([0-9]+).*/\1/p'); ja=$(echo "$jline" | sed -nE 's/.*before=([0-9]+) after=([0-9]+).*/\2/p')
  { [ -n "$hb" ] && [ -n "$jb" ] && [ $((hb - 1000 + 400)) -eq "$ha" ] && [ $((jb + 1000 - 400)) -eq "$ja" ]; } || { echo "FAIL: trade gold host $hb->$ha joiner $jb->$ja"; fail=1; }
  echo "$hline" | grep -qE 'gave=\[[^]]+\] got=\[[^]]+\]' || { echo "FAIL: host trade line has no items both ways"; fail=1; }
  echo "$jline" | grep -qE 'gave=\[[^]]+\] got=\[[^]]+\]' || { echo "FAIL: joiner trade line has no items both ways"; fail=1; }
  grep -q "TRADE done a=" $log.txt || { echo "FAIL: server did not log the trade"; fail=1; }
  # both heroes are saved (and exported to a .d2s) after the trade
  local done_line n_exp
  done_line=$(grep -n "TRADE done a=" $log.txt | head -1 | cut -d: -f1)
  n_exp=0
  [ -n "$done_line" ] && n_exp=$(tail -n +$done_line $log.txt | grep -c "D2S EXPORT path=")
  [ "$n_exp" -ge 2 ] || { echo "FAIL: heroes not exported after the trade ($n_exp)"; fail=1; }

  # 5 PvP: hits both ways at the 17 percent scale, a peaceful swing is blocked
  local side f swings hits bad
  for side in host joiner; do
    f=$log.txt; [ $side = joiner ] && f=$j
    hits=$(grep -c "PVP SWING target=.* hit=true" $f)
    [ "$hits" -ge 1 ] || { echo "FAIL: $side landed no PvP hit"; fail=1; }
    grep -q "PVP HIT attacker=" $f || { echo "FAIL: $side was never hit"; fail=1; }
    bad=$(sed -nE 's/.*PVP SWING target="[^"]*" hit=true raw=([0-9]+) scaled=([0-9]+) pct=([0-9]+).*/\1 \2 \3/p' $f | awk '$3 != 17 || $2 != int($1 * 17 / 100) {n++} END {print n + 0}')
    [ "$bad" -eq 0 ] || { echo "FAIL: $side PvP damage is not 17 percent of the swing"; fail=1; }
  done
  grep -q "PVP BLOCKED target=\"$jn\" reason=\"not hostile\"" $log.txt || { echo "FAIL: swing after making peace was not blocked"; fail=1; }
  grep -q "PVP HIT attacker=\"$hn\"" $j || { echo "FAIL: joiner did not see the host's hit"; fail=1; }
  grep -q "PVP HIT attacker=\"$jn\"" $log.txt || { echo "FAIL: host did not see the joiner's hit"; fail=1; }
  grep -q "PVP KILLED" $log.txt $j && { echo "FAIL: a hero died in the PvP exchange"; fail=1; }

  # 6 clean leave
  grep -q "notice=\"$jn left the game\"" $log.txt || { echo "FAIL: host did not see the joiner leave"; fail=1; }
  grep 'SOCIAL roster' $log.txt | tail -1 | grep -q 'n=1 ' || { echo "FAIL: host still lists the joiner after it left"; fail=1; }
  if grep -E "\[(ERROR|WARNING)\]|panic" $j | grep -v "skipping missing"; then echo "FAIL: errors in the joiner log"; fail=1; fi
}
