scenario_name="click matrix in a dungeon (Jail Level 1: a left click opens a door once, activates a waypoint, opens a chest, picks up gold; a ground click walks, Shift+click casts the left skill standing still, right click casts the right skill)"
scenario_timeout=300
cd_=$tmp/clickjail
# The same input path as 8g-click-bloodmoor (injected events through the input manager, waits on log lines) in a maze level of the real
# DRLG maps. The door (objects.txt 15, Door Wooden Left), the waypoint (288, the cellar waypoint of the Act 1 dungeons) and the chest
# (5) are put on free cells next to the hero by spawnchest (three of a kind: the click takes the nearest, the others stay), the gold pile
# by spawnground. A door toggles, so "opens once" is one OBJECT door line with open=true changed=true and no second one.
scenario_env() {
  mkdir -p $cd_/s94 $cd_/wb94; rm -f $cd_/s94/*.d2s(N) $cd_/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $cd_/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$cd_/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$cd_/wb94\" OD2_AUTOSPEED=3"
  else
    echo "# no revived sample hero, using the default save" >&2
  fi
  echo 'export OD2_REALMAPS=1 OD2_AUTOLEVEL=29'
  local still='say:herostill;until:HERO still,120'
  local s='expect:level=29;say:learnskillid 36;say:learnskillid 47;skill:left=Fire Bolt;skill:right=Fire Ball;say:heropos'
  # 1 door
  s="$s;say:spawnchest 15 15 15;until:AUTOGROUND chest-spawn id=15,30;click:left@object:15;until:OBJECT door,120;$still;say:heropos"
  # 2 waypoint: activates and opens the panel; Escape closes it
  s="$s;say:spawnchest 288 288 288;until:AUTOGROUND chest-spawn id=288,30;click:left@object:288;until:WAYPOINT activated level=29,120;$still;press:Escape;say:heropos"
  # 3 chest
  s="$s;say:spawnchest 5 5 5;until:AUTOGROUND chest-spawn id=5,30;click:left@object:5;until:AUTOGROUND chest id=5,120;$still;say:heropos"
  # 4 gold
  s="$s;say:spawnground gold 555 6;until:AUTOGROUND spawn source=command,30;click:left@item:555;until:AUTOGROUND pickup gold amount=555,120;$still;say:heropos"
  # 5 ground walk, Shift+click, right click (the free cell next to the hero in the direction of the click is not known: the walk
  # target is only a point on the screen, the hero stops where the walls stop him; a ground click must still start a walk)
  s="$s;say:heromoved;click:left@hero:60,30;until:HERO moved,120;$still;say:heropos"
  s="$s;click:left+shift@hero:60,30;until:WATCH cast-start,60;$still;say:heropos"
  s="$s;click:right@hero:60,30;until:WATCH cast-start,60;$still;say:heropos;exit"
  echo "export OD2_AUTOSCRIPT='$s'"
}
scenario_check() {
  grep -E "CLICK target|INPUT world-click|WATCH|HERO pos|AUTOGROUND (chest|pickup)|OBJECT door|WAYPOINT|walking to|OBJECT walking|AUTOSCRIPT RESULT|AUTOSCRIPT step [0-9]* FAIL" $log.txt | cut -c1-220 | head -80
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: dungeon click matrix script did not pass"; fail=1; }
  grep -q "HERO pos=.*town=false" $log.txt || { echo "FAIL: hero not outside town"; fail=1; }
  body=$(grep -v AUTOSCRIPT $log.txt)
  [ "$(echo "$body" | grep -c 'OBJECT door .*id=15 ')" = 1 ] || { echo "FAIL: the door must be operated exactly once"; fail=1; }
  echo "$body" | grep -q 'OBJECT door .*id=15 open=true changed=true' || { echo "FAIL: the door did not open"; fail=1; }
  [ "$(echo "$body" | grep -c 'WAYPOINT activated level=29')" = 1 ] || { echo "FAIL: the waypoint must be activated exactly once"; fail=1; }
  [ "$(echo "$body" | grep -c 'AUTOGROUND chest id=5')" = 1 ] || { echo "FAIL: the chest must open exactly once"; fail=1; }
  [ "$(echo "$body" | grep -c 'AUTOGROUND pickup gold amount=555')" = 1 ] || { echo "FAIL: the gold pile must be picked up exactly once"; fail=1; }
  # one walk-and-use line per object click: door, waypoint, chest (open), gold (pick up)
  [ "$(echo "$body" | grep -c 'OBJECT walking to use')" = 2 ] || { echo "FAIL: want two 'OBJECT walking to use' (door, waypoint)"; fail=1; }
  [ "$(echo "$body" | grep -c 'walking to open')" = 1 ] || { echo "FAIL: want one 'walking to open' (chest)"; fail=1; }
  [ "$(echo "$body" | grep -c 'walking to pick up')" = 1 ] || { echo "FAIL: want one 'walking to pick up' (gold)"; fail=1; }
  actions=$(echo "$body" | grep "INPUT world-click" | sed 's/.*action=\([a-z-]*\) left_skill=\([0-9]*\)/\1:\2/' | tr -d '\r' | tr '\n' ' ')
  want="move:36 stand-still:36 cast-right:36 "
  [ "$actions" = "$want" ] || { echo "FAIL: world click actions '$actions' want '$want'"; fail=1; }
  casts=$(echo "$body" | grep "CAST start" | sed 's/.*skill="\([^"]*\)".*/\1/' | tr -d '\r' | tr '\n' ',')
  want="Fire Bolt,Fire Ball,"
  [ "$casts" = "$want" ] || { echo "FAIL: skills cast '$casts' want '$want'"; fail=1; }
  # positions: 1 start, 2 door, 3 waypoint, 4 chest, 5 gold, 6 ground walk, 7 Shift, 8 right
  pos=("${(@f)$(grep -o 'HERO pos=([-0-9.]*,[-0-9.]*)' $log.txt | sed 's/HERO pos=//')}")
  if [ ${#pos} -ne 8 ]; then echo "FAIL: want 8 HERO pos lines, got ${#pos}"; fail=1; else
    [ "$pos[5]" != "$pos[6]" ] || { echo "FAIL: the ground click did not move the hero"; fail=1; }
    [ "$pos[6]" = "$pos[7]" ] || { echo "FAIL: Shift+click moved the hero"; fail=1; }
    [ "$pos[7]" = "$pos[8]" ] || { echo "FAIL: right click moved the hero"; fail=1; }
    for i in 2 3 4 5; do [ "$pos[$((i-1))]" != "$pos[$i]" ] || { echo "FAIL: click $i (door, waypoint, chest, gold) did not move the hero"; fail=1; }; done
  fi
}
