scenario_realtime=1
scenario_name="PvP skills and the hardcore ear (two processes over TCP: a hostile hero's Fire Ball, Meteor and Blizzard hurt a hero at 17 percent; a hardcore kill drops an ear that is picked up and saved in the .d2s)"
# Host = the killer (softcore real save), joiner = the victim (a hardcore copy of the same save under another name).
# Both walk through a spawned portal to the Cold Plains first: skills do not work in town. Steps:
#   1 host casts at the joiner before declaring hostility: nothing is sent
#   2 host declares hostility, walks away and casts Fire Ball (missile + splash), Meteor (area) and Blizzard (storm)
#   3 the host keeps casting Fire Balls until the joiner (a full-life hero) dies: hardcore death, kill report, ear on the ground
#   4 the host picks the ear up and leaves; the exported .d2s has the ear
# Meteor's burning ground ticks (3.5 fire, under one point at 17 percent) hurt through the carried fraction
# (d2combat.PvPCarry), so the victim burns down while it stands in the fire.
# UNVERIFIED rules: the 17 percent scale (pvp.go), the defender applies its own resists after the scale, the ear
# drops where the victim fell. Curses and auras are in the same code path (d2core/d2skills/pvp.go) but need a
# paladin or necromancer hero; this scenario covers missiles, area hits and storms.
mp_second_name=NokkaEar
mp_join_log=$tmp/9f-join.log

_9f_name() { python3 -I -c 'import sys;b=open(sys.argv[1],"rb").read();print(b[0x14:0x24].split(b"\0")[0].decode())' "$1"; }

# _9f_make <src> <dst> <name>: a hardcore copy of a save under another hero name (checksum fixed up)
_9f_make() {
  python3 -I -c '
import sys
b = bytearray(open(sys.argv[1], "rb").read())
b[0x14:0x24] = sys.argv[3].encode().ljust(16, b"\0")
b[0x24] |= 0x04
b[0x0c:0x10] = b"\0\0\0\0"
s = 0
for x in b:
    s = ((((s << 1) | (s >> 31)) & 0xffffffff) + x) & 0xffffffff
b[0x0c:0x10] = s.to_bytes(4, "little")
open(sys.argv[2], "wb").write(b)' "$1" "$2" "$3"
}

_9f_casts() {
  local i out=""
  for i in {1..$2}; do out+="say:pvpcast $1 $mp_second_name;wait:2;"; done
  echo "${out%;}"
}

scenario_env() {
  mp_host_name=$(_9f_name $save)
  local jn=$mp_second_name hn=$mp_host_name jsave=$tmp/9f-join.d2s jcmd=$tmp/9f-join.command
  _9f_make $save $jsave $jn

  local common="export OD2_PORT=$OD2_PORT OD2_PROTO=d2gs OD2_AUTOPARTY=1 ${OD2_VERIFY_MUTE_ENV} OD2_AUTOEXIT=1 OD2_D2S_WRITEBACK=$tmp"

  local hscript="wait:2;say:dropinv cm1;say:dropinv cm1;waitlog:SOCIAL roster n=2;say:spawnportal 3;use:Portal;waitlog:LEVEL built: level 3"
  hscript+=";wait:15;say:pvpwalk 6 0;wait:4;say:players"
  hscript+=";say:pvpcast Fire_Ball $jn;wait:3"
  hscript+=";say:hostile $jn 1;waitlog:$hn is now hostile toward $jn;wait:1"
  hscript+=";$(_9f_casts Fire_Ball 4);say:pvpcast Meteor $jn;wait:6;say:pvpcast Blizzard $jn;wait:8"
  # The victim heals (its worn items have Replenish Life, and the natural regeneration applies to it): the 12 Fire
  # Balls that once just killed it (43 of 1241 life left before the last one without regeneration) leave it alive
  # with it. 24 casts is plenty; casts after the kill are skipped by pvpcast (the victim left the game).
  hscript+=";say:chat nowdie;wait:3;$(_9f_casts Fire_Ball 24)"
  hscript+=";waitlog:PVP EAR dropped;wait:1;loot:8,15;wait:2;say:chat eartaken;wait:4;exit"

  local jscript="wait:1;waitlog:SOCIAL roster n=2;say:spawnportal 3;use:Portal;waitlog:LEVEL built: level 3"
  # the victim may die before the host's last Fire Ball (Meteor's burning ground keeps hurting): it stays in the
  # game until the host has picked the ear up, so the host's casts and the loot still find it
  jscript+=";waitlog:PVP HIT attacker;waitlog:PVP KILLED;waitlog:eartaken;wait:1;exit"

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

scenario_check() {
  local j=$mp_join_log.txt hn=$mp_host_name jn=$mp_second_name
  [ -f "$mp_join_log" ] && sed 's/\x1b\[[0-9;]*m//g' $mp_join_log > $j
  if [ ! -f "$j" ]; then echo "FAIL: no joiner log"; fail=1; return; fi

  local pat="PVP |PORTAL|D2S EXPORT reparse|AUTOSCRIPT RESULT|DEATH hero|hostile"
  grep -E "$pat" $log.txt | sed 's/^/host   /' | cut -c1-300
  grep -E "$pat" $j | sed 's/^/joiner /' | cut -c1-300

  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: host script did not pass"; fail=1; }
  grep -q "AUTOSCRIPT RESULT PASS" $j || { echo "FAIL: joiner script did not pass"; fail=1; }

  # 1 a cast at a hero who is not hostile hurts nobody
  local hostile_line first_skill
  hostile_line=$(grep -n "$hn is now hostile toward $jn" $log.txt | head -1 | cut -d: -f1)
  first_skill=$(grep -n "PVP SKILL target=" $log.txt | head -1 | cut -d: -f1)
  { [ -n "$hostile_line" ] && [ -n "$first_skill" ] && [ "$first_skill" -gt "$hostile_line" ]; } || { echo "FAIL: a skill hit a hero who was not hostile (hostile@$hostile_line first skill hit@$first_skill)"; fail=1; }

  # 2 missiles, areas and storms hit, at the PvP scale, and the defender received them
  local sk
  for sk in "Fire Ball" "Meteor" "Blizzard"; do
    grep -q "PVP SKILL target=\"$jn\" skill=\"$sk" $log.txt || { echo "FAIL: no $sk hit on the hostile hero"; fail=1; }
    grep -q "PVP HIT attacker=\"$hn\" .*skill=\"$sk" $j || { echo "FAIL: the joiner never took a $sk hit"; fail=1; }
  done

  local bad
  bad=$(sed -nE 's/.*PVP SKILL target="[^"]*" skill="[^"]*" raw=([0-9]+) scaled=([0-9]+) pct=([0-9]+).*/\3/p' $log.txt | awk '$1 != 17 {n++} END {print n + 0}')
  [ "$bad" -eq 0 ] || { echo "FAIL: $bad skill hits are not scaled to 17 percent"; fail=1; }

  # The exact scale, with no slack beyond the carry: raw= is rounded to whole points, so the check uses the unrounded
  # 8.8 damage of every hit (exact=[phys fire light magic cold], one PVPSKILL line per hit, including the ticks that
  # only add to the carry). Per damage type PvPCarry keeps t = the sum of floor(v*17/100) (units of 1/256 point) and
  # pays out whole points, leaving a remainder under one point, so the points charged satisfy EXACTLY
  # 256*charged <= t < 256*charged + 256 (no slack; one hit alone is at most floor((v*17/100 + 255)/256), the first
  # line of the awk). d2combat.PvPScaledBounds is the closed form of the same bound, unit-tested.
  bad=$(sed -nE 's/.*PVPSKILL skill="[^"]*" target=[^ ]* raw=[0-9]+ scaled=[0-9]+ pct=17 parts=\[([0-9 ]+)\] exact=\[([0-9 ]+)\].*/\1 \2/p' $log.txt | awk '
    { for (i = 1; i <= 5; i++) { s = $i; v = $(i + 5); sum[i] += s; ex[i] += v; t[i] += int(v * 17 / 100)
        if (s * 25600 > 17 * v + 25500) { bad++; print "  hit above the exact bound: type " i " scaled " s " exact88 " v > "/dev/stderr" } }
      lines++ }
    END { for (i = 1; i <= 5; i++) {
            if (sum[i] * 256 > t[i]) { bad++; print "  type " i ": charged " sum[i] " above the exact 17 percent (" t[i] "/256) of " ex[i] "/256" > "/dev/stderr" }
            if (sum[i] * 256 + 256 <= t[i]) { bad++; print "  type " i ": charged " sum[i] " is a whole point or more below the exact " t[i] "/256" > "/dev/stderr" } }
          print bad + 0 " " lines + 0 }')
  if [ "${bad#* }" -eq 0 ]; then echo "FAIL: no PVPSKILL line with exact= in the host log"; fail=1
  elif [ "${bad%% *}" -ne 0 ]; then echo "FAIL: ${bad%% *} PvP scale bound violations (17 percent of the unrounded damage plus the carry)"; fail=1; fi
  bad=$(sed -nE 's/.*PVP HIT attacker="[^"]*" raw=([0-9]+) scaled=([0-9]+) taken=([0-9]+).*skill="[^"]+".*/\2 \3/p' $j | awk '$2 > $1 {n++} END {print n + 0}')
  [ "$bad" -eq 0 ] || { echo "FAIL: $bad hits took more than the scaled damage"; fail=1; }

  # 3 hardcore death, kill report to the killer, ear
  grep -q "PVP KILLED by=\"$hn\" ear=true hardcore=true" $j || { echo "FAIL: the hardcore victim did not report an ear kill"; fail=1; }
  grep -q "PVP KILL victim=\"$jn\" killer=\"$hn\" level=[0-9]* class=1 hardcore=true" $log.txt || { echo "FAIL: server did not relay the kill"; fail=1; }
  grep -q "PVP EAR dropped victim=\"$jn\" level=94 class=1" $log.txt || { echo "FAIL: no ear dropped for the killer"; fail=1; }
  grep -q "DEATH hardcore: the character is dead" $j || { echo "FAIL: the hardcore victim did not die for good"; fail=1; }

  # 4 the ear was picked up and is in the exported save
  grep -qE "D2S EXPORT reparse: .*ears=\[$jn/L94/c1\]" $log.txt || { echo "FAIL: the exported .d2s of the killer has no ear of $jn"; fail=1; }
}
