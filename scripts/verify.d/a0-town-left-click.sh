scenario_name="town left click (Glacial Spike on the left button still walks in town; right click does not move; Attack walks)"
tl=$tmp/townleft
# Regression for "left clicks did nothing in town with a left skill that cannot be used there"
# (INPUT world-click action=cast-left ... CAST start ok=false reason=town). Clicks go through GameControls.OnMouseButtonDown
# -> worldClick like the mouse; heropos is a test-only console command that logs the hero position.
scenario_env() {
  mkdir -p $tl/s94 $tl/wb94; rm -f $tl/s94/*.d2s(N) $tl/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $tl/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$tl/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$tl/wb94\" OD2_AUTOSPEED=3"
  else
    echo "# no revived sample hero, using the default save" >&2
  fi
  # warm-up: the first walk loads the walk animations lazily (a stall of seconds on a busy machine), so walk once and wait
  # for the hero to have moved and stopped (heromoved/herostill log lines, see hero_watch.go) instead of a fixed time
  local s='wait:2;say:learnskillid 55;skill:left=Glacial Spike;say:heromoved;click:left@hero:-100,0;until:HERO moved,120;say:herostill;until:HERO still,120;say:heropos'
  s="$s;say:heromoved;click:left@hero:160,40;until:HERO moved,120;say:herostill;until:HERO still,120;say:heropos"  # 1 left click, Glacial Spike: walks
  s="$s;click:right@hero:-100,60;wait:1;click:left+cmd@hero:-100,60;wait:3;say:heropos"  # 2 right click and Cmd+click: cast the right skill (refused in town), no move
  s="$s;skill:left=Attack;say:heromoved;click:left@hero:-100,0;until:HERO moved,120;say:herostill;until:HERO still,120;say:heropos;exit"  # 3 Attack: walks
  echo "export OD2_AUTOSCRIPT='$s'"
}
scenario_check() {
  grep -E "INPUT world-click|HERO pos|CAST start|AUTOSCRIPT RESULT|SKILLBAR select" $log.txt | cut -c1-200 | head -30
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: town left click script did not pass"; fail=1; }
  actions=$(grep "INPUT world-click" $log.txt | sed 's/.*action=\([a-z-]*\) left_skill=\([0-9]*\)/\1:\2/' | tr -d '\r' | tr '\n' ' ')
  want="move:55 move:55 cast-right:55 cast-right:55 move:0 "
  [ "$actions" = "$want" ] || { echo "FAIL: world click actions '$actions' want '$want'"; fail=1; }
  grep -E "CAST start.*reason=town" $log.txt | grep -q "Glacial Spike" && { echo "FAIL: Glacial Spike cast was attempted in town on a left click"; fail=1; }
  grep -q "HERO pos=.*town=true" $log.txt || { echo "FAIL: hero not in town"; fail=1; }
  # positions: 1 start, 2 after the left click (moved), 3 after the right click (same as 2), 4 after the Attack click (moved)
  pos=("${(@f)$(grep -o 'HERO pos=([-0-9.]*,[-0-9.]*)' $log.txt | sed 's/HERO pos=//')}")
  if [ ${#pos} -ne 4 ]; then echo "FAIL: want 4 HERO pos lines, got ${#pos}"; fail=1; else
    [ "$pos[1]" != "$pos[2]" ] || { echo "FAIL: left click with Glacial Spike did not move the hero ($pos[1])"; fail=1; }
    [ "$pos[2]" = "$pos[3]" ] || { echo "FAIL: right click moved the hero ($pos[2] -> $pos[3])"; fail=1; }
    [ "$pos[3]" != "$pos[4]" ] || { echo "FAIL: left click with Attack did not move the hero ($pos[3])"; fail=1; }
  fi
}
