scenario_name="hold past NPCs and objects (Rogue Encampment: a hold that starts on the ground keeps walking when an NPC or the waypoint passes under the cursor and interacts with neither; a hold that starts on an NPC or the waypoint interacts exactly once)"
scenario_realtime=1
scenario_timeout=300
source scripts/verify.d/lib/segments.sh
hn=$tmp/holdnpc
# The owner found the control bugs of the hold by hand; this is the scripted version. The cursor stays at a fixed screen spot while the
# button is down, and the world under it moves with the hero (the camera follows him), so a cursor set half way to an NPC reaches that
# NPC's place when the hero has walked the other half. @npc:<name>*0.5 / @object:<id>*0.5 put the cursor there, on the ground.
# The log shows what happened: HOLD hover lines (what is under the cursor, from the renderer), the one line each interaction writes
# (INTERACT_RE in lib/segments.sh), HOLD pos samples (distance walked). Waits are on log lines (until:), not on time.
scenario_env() {
  mkdir -p $hn/s94 $hn/wb94; rm -f $hn/s94/*.d2s(N) $hn/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $hn/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$hn/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$hn/wb94\""
  else
    echo "# no revived sample hero, using the default save" >&2
  fi
  local still='say:herostill;until:HERO still,120'
  # warm-up: the first walk loads the walk animations lazily; walk once and wait for it to end
  local s='wait:2;skill:left=Attack;say:heromoved;click:left@hero:60,20;until:HERO moved,120;'"$still"';say:heropos'
  # the straight, unobstructed routes of the camp start at the hero's starting place (31.6,13.8): the holds toward the waypoint and
  # toward Akara each start there (a path that has to bend round a hut would not pass under the cursor)
  local back="say:heromoved;move:31.6,13.8;until:HERO moved,120;$still"
  s="$s;hold:6,left@npc:Akara*0.5;$still;say:heropos"       # 1 ground hold towards Akara: she passes under the cursor, no interaction
  s="$s;$back;hold:3,left@object:119*0.5;$still;say:heropos"  # 2 ground hold towards the waypoint: it passes under the cursor
  s="$s;hold:3,left@npc:Akara;until:NPC menu opened,60;press:Escape;$still"  # 3 hold that starts on Akara: one interaction, the menu opens once
  s="$s;hold:3,left@object:119;$still;press:Escape;say:heropos;exit"  # 4 hold that starts on the waypoint: one interaction
  echo "export OD2_AUTOSCRIPT='$s'"
}
scenario_check() {
  grep -E "CLICK target|HOLD start|HOLD hover|HERO pos|interacting with|OBJECT walking|NPC menu|AUTOSCRIPT RESULT|AUTOSCRIPT step [0-9]* FAIL" $log.txt | cut -c1-200 | head -80
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: hold-npc script did not pass"; fail=1; }
  grep -q "HERO pos=.*town=true" $log.txt || { echo "FAIL: hero not in town"; fail=1; }
  seg=("${(@f)$(seg_events $log.txt 'HOLD start')}")
  hold=("${(@f)$(hold_segments $log.txt)}")
  printf '%s\n' $seg $hold
  [ ${#seg} -eq 4 ] || { echo "FAIL: want 4 hold segments, got ${#seg}"; fail=1; return; }
  # 1 (first hold): ground hold past Akara
  [ "$(seg_field $seg[1] interact)" = 0 ] || { echo "FAIL: the ground hold interacted with something ($seg[1])"; fail=1; }
  [[ "$(seg_field $seg[1] hovers)" == *Akara* ]] || { echo "FAIL: Akara never was under the cursor during the ground hold: nothing was tested ($seg[1])"; fail=1; }
  awk -v h="$hold[1]" 'BEGIN { split(h, a, " "); w=a[4]; sub(/walked=/, "", w); s=a[5]; sub(/longest_still=/, "", s); exit !(w+0 >= 6 && s+0 <= 1.0) }' \
    || { echo "FAIL: the ground hold did not keep the hero walking ($hold[1]; want walked >= 6, longest_still <= 1)"; fail=1; }
  # 2 (second hold): ground hold past the waypoint
  [ "$(seg_field $seg[2] interact)" = 0 ] || { echo "FAIL: the ground hold toward the waypoint interacted ($seg[2])"; fail=1; }
  [[ "$(seg_field $seg[2] hovers)" == *Waypoint* ]] || { echo "FAIL: the waypoint never was under the cursor during the ground hold ($seg[2])"; fail=1; }
  awk -v h="$hold[2]" 'BEGIN { split(h, a, " "); w=a[4]; sub(/walked=/, "", w); exit !(w+0 >= 2) }' \
    || { echo "FAIL: the hold toward the waypoint walked too little ($hold[2])"; fail=1; }
  # 3 and 4: holds that start on an NPC / object interact once, whatever passes under the cursor later
  [ "$(seg_field $seg[3] interact)" = 1 ] || { echo "FAIL: a hold on Akara must interact exactly once ($seg[3])"; fail=1; }
  [ "$(grep -c 'NPC menu opened: npc="Akara"' $log.txt)" = 1 ] || { echo "FAIL: Akara's menu must open exactly once"; fail=1; }
  [ "$(seg_field $seg[4] interact)" = 1 ] || { echo "FAIL: a hold on the waypoint must interact exactly once ($seg[4])"; fail=1; }
  grep -q "interacting with \"Akara\"" $log.txt || { echo "FAIL: no interaction with Akara logged"; fail=1; }
}
