scenario_name="click matrix outside town (Blood Moor: a chest opens on a left click, gold and an item on the ground are picked up, a ground click walks, Shift+click casts the left skill standing still, right / Ctrl / Cmd click cast the right skill, a click on a monster casts the left skill)"
scenario_timeout=300
cm=$tmp/clickmoor
# The hero (Fire Bolt left, Fire Ball right) clicks through the whole matrix with the input manager's injected events. Every wait is on a log
# line (until:), none on a clock, so a loaded machine only makes the run slower. The chest, the gold pile and the item are placed on free
# cells next to the hero by the test commands spawnchest / spawnground (the cell numbers keep the three apart).
# The mirrors in the Game Screen log that the script waits for: "WATCH cast-start" (a cast began) and "WATCH world-click action=" (the
# decision of a click on the ground); the scenario counts the original "CAST start" / "INPUT world-click" lines.
scenario_env() {
  mkdir -p $cm/s94 $cm/wb94; rm -f $cm/s94/*.d2s(N) $cm/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $cm/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$cm/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$cm/wb94\" OD2_AUTOSPEED=3"
  else
    echo "# no revived sample hero, using the default save" >&2
  fi
  echo 'export OD2_REALMAPS=1 OD2_AUTOLEVEL=2 OD2_AUTOMONSTER="zombie1,1" OD2_AUTOMONSTER_PASSIVE=1 OD2_AUTOMONSTER_HOLD=1 OD2_AUTOMONSTER_DIFF=0 OD2_AUTOMONSTER_SECONDS=240'
  local still='say:herostill;until:HERO still,120'
  local g='hero:200,0'   # open ground to the right of the hero
  local s='expect:level=2;say:learnskillid 36;say:learnskillid 47;skill:left=Fire Bolt;skill:right=Fire Ball'
  # warm-up walk (animations load lazily), then the baseline position
  s="$s;say:heromoved;click:left@$g;until:HERO moved,120;$still;say:heropos"
  # 1 chest: a left click walks to it and opens it
  s="$s;say:spawnchest 5 5 5;until:AUTOGROUND chest-spawn id=5,30;click:left@object:5;until:AUTOGROUND chest id=5,120;$still;say:heropos"
  # 2 gold pile: a left click walks to it and picks it up
  s="$s;say:spawnground gold 777 6;until:AUTOGROUND spawn source=command,30;click:left@item:777;until:AUTOGROUND pickup gold amount=777,120;$still;say:heropos"
  # 3 ground click walks; Shift+click casts the left skill standing still; Cmd, Ctrl and right click cast the right skill
  s="$s;say:heromoved;click:left@$g;until:HERO moved,120;$still;say:heropos"
  s="$s;click:left+shift@$g;until:WATCH cast-start,60;$still;say:heropos"
  s="$s;click:left+cmd@$g;until:WATCH cast-start,60;$still;say:heropos"
  s="$s;click:left+ctrl@$g;until:WATCH cast-start,60;$still;say:heropos"
  s="$s;click:right@$g;until:WATCH cast-start,60;$still;say:heropos"
  # 4 a click on a monster casts the left skill
  s="$s;click:left@monster;until:WATCH cast-start,60;$still;say:heropos"
  # 5 an item on the ground: the click walks to it and picks it up onto the cursor (last: the cursor then holds it)
  s="$s;say:spawnground tbk 8 8;until:AUTOGROUND spawn source=command name=,30;click:left@item:Tome;until:AUTOGROUND pickup name=,120;$still;say:heropos;exit"
  echo "export OD2_AUTOSCRIPT='$s'"
}
scenario_check() {
  grep -E "CLICK target|INPUT world-click|WATCH|HERO pos|AUTOGROUND (chest|pickup|spawn)|walking to|AUTOSCRIPT RESULT|AUTOSCRIPT step [0-9]* FAIL" $log.txt | cut -c1-220 | head -80
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: click matrix script did not pass"; fail=1; }
  grep -q "HERO pos=.*town=false" $log.txt || { echo "FAIL: hero not outside town"; fail=1; }
  # interactions: exactly one walk-and-use line for the chest, the gold pile and the item
  [ "$(grep -c 'walking to open' $log.txt)" = 1 ] || { echo "FAIL: want exactly one 'walking to open' (the chest)"; fail=1; }
  [ "$(grep -c 'walking to pick up' $log.txt)" = 2 ] || { echo "FAIL: want exactly two 'walking to pick up' (gold, item)"; fail=1; }
  [ "$(grep -v AUTOSCRIPT $log.txt | grep -c 'AUTOGROUND chest id=5')" = 1 ] || { echo "FAIL: the chest must open exactly once"; fail=1; }
  [ "$(grep -v AUTOSCRIPT $log.txt | grep -c 'AUTOGROUND pickup gold amount=777')" = 1 ] || { echo "FAIL: the gold pile must be picked up exactly once"; fail=1; }
  grep -q 'AUTOGROUND pickup name=.*-> cursor' $log.txt || { echo "FAIL: the item was not picked up onto the cursor"; fail=1; }
  # the clicks on the ground, in order: walk (warm-up), walk, Shift = stand-still, Cmd, Ctrl, right = cast-right, the monster click = cast-left
  actions=$(grep "INPUT world-click" $log.txt | sed 's/.*action=\([a-z-]*\) left_skill=\([0-9]*\)/\1:\2/' | tr -d '\r' | tr '\n' ' ')
  want="move:36 move:36 stand-still:36 cast-right:36 cast-right:36 cast-right:36 cast-left:36 "
  [ "$actions" = "$want" ] || { echo "FAIL: world click actions '$actions' want '$want'"; fail=1; }
  casts=$(grep "CAST start" $log.txt | sed 's/.*skill="\([^"]*\)".*/\1/' | tr -d '\r' | tr '\n' ',')
  want="Fire Bolt,Fire Ball,Fire Ball,Fire Ball,Fire Bolt,"
  [ "$casts" = "$want" ] || { echo "FAIL: skills cast '$casts' want '$want'"; fail=1; }
  grep "CAST start" $log.txt | grep -q "ok=false" && { echo "FAIL: a cast was refused"; fail=1; }
  # positions (heropos): 1 baseline after the warm-up walk, 2 chest, 3 gold, 4 after the ground walk, 5 Shift, 6 Cmd, 7 Ctrl, 8 right, 9 monster, 10 item
  pos=("${(@f)$(grep -o 'HERO pos=([-0-9.]*,[-0-9.]*)' $log.txt | sed 's/HERO pos=//')}")
  if [ ${#pos} -ne 10 ]; then echo "FAIL: want 10 HERO pos lines, got ${#pos}"; fail=1; else
    [ "$pos[1]" != "$pos[2]" ] || { echo "FAIL: the click on the chest did not move the hero"; fail=1; }
    [ "$pos[2]" != "$pos[3]" ] || { echo "FAIL: the click on the gold did not move the hero"; fail=1; }
    [ "$pos[3]" != "$pos[4]" ] || { echo "FAIL: the ground click did not move the hero"; fail=1; }
    for i in 5 6 7 8; do
      [ "$pos[$((i-1))]" = "$pos[$i]" ] || { echo "FAIL: click $i (Shift / Cmd / Ctrl / right) moved the hero ($pos[$((i-1))] -> $pos[$i])"; fail=1; }
    done
    [ "$pos[9]" != "$pos[10]" ] || { echo "FAIL: the click on the item did not move the hero"; fail=1; }
  fi
}
