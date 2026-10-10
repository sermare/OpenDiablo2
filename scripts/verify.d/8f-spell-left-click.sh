scenario_name="spell on the left button outside town (Blood Moor, Fire Bolt left / Fire Ball right: left click on ground walks, on a monster casts, Shift+click casts standing still, Cmd+click and right click use the right skill)"
scenario_timeout=240
sp=$tmp/spellleft
# Regression for control bugs the other scenarios missed. Clicks go through GameControls.OnMouseButtonDown like the mouse;
# click:left@monster targets the nearest living monster. INPUT world-click lines carry the WorldAction, HERO pos lines (heropos)
# the position, CAST start lines the skill that was used.
scenario_env() {
  mkdir -p $sp/s94 $sp/wb94; rm -f $sp/s94/*.d2s(N) $sp/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $sp/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$sp/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$sp/wb94\" OD2_AUTOSPEED=3"
  else
    echo "# no revived sample hero, using the default save" >&2
  fi
  echo 'export OD2_REALMAPS=1 OD2_AUTOLEVEL=2 OD2_AUTOMONSTER="zombie1,1" OD2_AUTOMONSTER_PASSIVE=1 OD2_AUTOMONSTER_HOLD=1 OD2_AUTOMONSTER_DIFF=0 OD2_AUTOMONSTER_SECONDS=120'
  local s='wait:2;expect:level=2;wait:4;say:learnskillid 36;say:learnskillid 47;skill:left=Fire Bolt;skill:right=Fire Ball;say:heropos'
  s="$s;click:left@monster;wait:5"                                  # 1 left click on a monster: casts the left skill
  s="$s;click:left@${SPELL_GROUND:-hero:200,0};wait:6;say:heropos"     # 2 left click on the ground: walks
  s="$s;click:left+shift@${SPELL_GROUND:-hero:200,0};wait:4;say:heropos"  # 3 Shift+click: casts the left skill standing still
  s="$s;click:left+cmd@${SPELL_GROUND:-hero:200,0};wait:4;say:heropos"    # 4 Cmd+click: the right skill
  s="$s;click:right@${SPELL_GROUND:-hero:200,0};wait:4;say:heropos;exit"  # 5 right click: the right skill
  echo "export OD2_AUTOSCRIPT='$s'"
}
scenario_check() {
  grep -E "INPUT world-click|HERO pos|CAST start|CAST do|AUTOSCRIPT RESULT|SKILLBAR select|MONSTER spawn|AUTOSCRIPT step [0-9]* FAIL" $log.txt | cut -c1-220 | head -40
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: spell-left-click script did not pass"; fail=1; }
  grep -q "HERO pos=.*town=false" $log.txt || { echo "FAIL: hero not outside town"; fail=1; }
  actions=$(grep "INPUT world-click" $log.txt | sed 's/.*action=\([a-z-]*\) left_skill=\([0-9]*\)/\1:\2/' | tr -d '\r' | tr '\n' ' ')
  want="cast-left:36 move:36 stand-still:36 cast-right:36 cast-right:36 "
  [ "$actions" = "$want" ] || { echo "FAIL: world click actions '$actions' want '$want'"; fail=1; }
  # the skills that were cast, in order: Fire Bolt for the monster click and Shift+click, Fire Ball for Cmd+click and right click
  casts=$(grep "CAST start" $log.txt | sed 's/.*skill="\([^"]*\)".*/\1/' | tr -d '\r' | tr '\n' ',')
  want="Fire Bolt,Fire Bolt,Fire Ball,Fire Ball,"
  [ "$casts" = "$want" ] || { echo "FAIL: skills cast '$casts' want '$want'"; fail=1; }
  grep "CAST start" $log.txt | grep -q "ok=false" && { echo "FAIL: a cast was refused"; fail=1; }
  # positions: 1 before the clicks, 2 after the ground click (moved), 3 after Shift+click, 4 after Cmd+click, 5 after the right click (all standing)
  pos=("${(@f)$(grep -o 'HERO pos=([-0-9.]*,[-0-9.]*)' $log.txt | sed 's/HERO pos=//')}")
  if [ ${#pos} -ne 5 ]; then echo "FAIL: want 5 HERO pos lines, got ${#pos}"; fail=1; else
    walked=$(echo "$pos[1] $pos[2]" | tr -d '()' | tr ',' ' ' | awk '{print sqrt(($3-$1)^2+($4-$2)^2)}')
    awk "BEGIN{exit !($walked > 0.5)}" || { echo "FAIL: the ground click walked only $walked tiles"; fail=1; }
    [ "$pos[1]" != "$pos[2]" ] || { echo "FAIL: left click on the ground with Fire Bolt did not move the hero ($pos[1])"; fail=1; }
    [ "$pos[2]" = "$pos[3]" ] || { echo "FAIL: Shift+click moved the hero ($pos[2] -> $pos[3])"; fail=1; }
    [ "$pos[3]" = "$pos[4]" ] || { echo "FAIL: Cmd+click moved the hero ($pos[3] -> $pos[4])"; fail=1; }
    [ "$pos[4]" = "$pos[5]" ] || { echo "FAIL: right click moved the hero ($pos[4] -> $pos[5])"; fail=1; }
  fi
}
