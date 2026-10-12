scenario_name="Assassin kick sound and footsteps on animation frames (SOUNDLOG: Dragon Talon plays weapon_kick_1, walk steps fall twice per cycle; w48)"
scenario_realtime=1
scenario_timeout=420
scenario_warnings_ok=1
# w48. A fresh Assassin (OD2_AUTOCAST_CLASS, level 10, own temp save) walks, then kicks a zombie with Dragon Talon.
# Footsteps: the exe steps on animation frames (UNITSND_PlayModeEndSound 0x4c7a20: 2 per walk cycle for a hero). The kick mode (KK) plays Sounds row 0xff
# (weapon_kick_1) with a delay of 4 ticks for the Assassin (UNITSND_HandleUnitModeSound 0x4c9030).
scenario_env() {
  local s
  s='wait:2;expect:level=2;wait:3;skill:spend=Dragon Talon;skill:left=Attack;say:heromoved;click:left@hero:140,30;wait:4;click:left@hero:-140,-30;wait:4;click:left@hero:140,30;wait:4;click:left@hero:-140,-30;wait:4'
  s="$s;skill:left=Dragon Talon;click:left@monster;wait:4;click:left@monster;wait:4;click:left@monster;wait:4;exit"
  echo "export OD2_SOUNDLOG=1 OD2_AUTOCAST_MANA=400 OD2_AUTOGAME=\"$tmp/w48.od2\" OD2_AUTOCAST_CLASS=assassin OD2_AUTOCAST_CLVL=20 OD2_AUTOCAST_SKILLPOINTS=8"
  echo 'export OD2_REALMAPS=1 OD2_AUTOLEVEL=2 OD2_AUTOMONSTER="zombie1,10" OD2_AUTOMONSTER_PASSIVE=1 OD2_AUTOMONSTER_HOLD=1 OD2_AUTOMONSTER_DIFF=0 OD2_AUTOMONSTER_SECONDS=300'
  echo "export OD2_AUTOSCRIPT='$s'"
}
scenario_check() {
  grep -E "SOUNDLOG .* kind=(hero-footstep|hero-kick|hero-swing) " $log.txt | sed -E 's/t=[0-9.]+ //' | cut -c1-120 | sort | uniq -c | sort -rn | head -10
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the script did not pass"; fail=1; }
  grep -qE "SOUNDLOG .* kind=hero-footstep .*handle=light_(walk|run)_[a-z]+_[0-9] " $log.txt || { echo "FAIL: no light footstep for the Assassin"; fail=1; }
  grep -qE "SOUNDLOG .* kind=hero-kick .*handle=weapon_kick_1 " $log.txt || { echo "FAIL: Dragon Talon (KK) played no kick sound"; fail=1; }
  # several steps over the long walks (two per cycle on the animation frames)
  [ "$(grep -cE 'SOUNDLOG .* kind=hero-footstep ' $log.txt)" -ge 4 ] || { echo "FAIL: fewer than 4 footsteps over two long walks"; fail=1; }
}
