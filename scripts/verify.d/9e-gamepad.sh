scenario_name="gamepad (synthetic controller: hot-plug, panels, automap, skill cycling, potion keys, left stick walking, right stick cursor)"
pd=$tmp/gamepad
scenario_env() {
  # a virtual controller is driven through the pad: steps (no hardware needed); pad:state logs the panels,
  # automap and active skills. X/Y/L3 toggle inventory/character/skill tree, BACK the automap, RB/LB cycle the
  # skills, D-pad up is potion 1, the left stick walks (positions come from expect:level), the right stick moves
  # the cursor, unplugging mid-game releases everything and plugging in again works.
  echo 'export OD2_SUBTITLE_LOG=1'
  echo "export OD2_AUTOSCRIPT='wait:1;automap:off;wait:1;expect:level=1;pad:connect;wait:1;pad:state;\
pad:press=X;wait:1;pad:state;pad:press=X;wait:1;pad:state;\
pad:press=Y;wait:1;pad:state;pad:press=Y;wait:1;pad:state;\
pad:press=L3;wait:1;pad:state;pad:press=L3;wait:1;pad:state;\
pad:press=BACK;wait:1;pad:state;pad:press=BACK;wait:1;pad:state;\
pad:press=RB;wait:1;pad:state;pad:press=RB;wait:1;pad:state;pad:press=LB;wait:1;pad:state;\
pad:press=DUP;wait:1;pad:press=DLEFT;wait:1;\
expect:level=1;pad:stick=left,1,0;wait:3;pad:stick=left,0,0;wait:1;expect:level=1;pad:stick=left,-1,0;wait:2;pad:stick=left,0,0;wait:1;\
pad:press=X;wait:1;pad:state;pad:stick=right,1,0;wait:1;pad:stick=right,0,0;wait:1;say:capframe $pd-inv.png;pad:press=X;wait:1;pad:state;pad:press=A;wait:1;\
pad:disconnect;wait:1;pad:press=X;wait:1;pad:state;pad:connect;wait:1;pad:press=X;wait:1;pad:state;pad:press=X;wait:1;pad:state;exit'"
}
scenario_check() {
  grep -E "GAMEPAD|PAD state|AUTOSCRIPT level=|AUTOSCRIPT RESULT" $log.txt | cut -c1-260
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: gamepad script did not pass"; fail=1; }

  # hot-plug: plugged in twice, unplugged once, the synthetic pad is id 1000
  [ "$(grep -c 'GAMEPAD connected id=1000' $log.txt)" = 2 ] || { echo "FAIL: expected two connects of the synthetic pad"; fail=1; }
  [ "$(grep -c 'GAMEPAD disconnected id=1000' $log.txt)" = 1 ] || { echo "FAIL: expected one disconnect"; fail=1; }

  # the default mapping reaches the game through the key bindings
  grep -qE "GAMEPAD button=X action=INVENTORY" $log.txt || { echo "FAIL: X is not INVENTORY"; fail=1; }
  grep -qE "GAMEPAD button=DUP action=POTION 1" $log.txt || { echo "FAIL: D-pad up is not POTION 1"; fail=1; }
  grep -qE "GAMEPAD button=DLEFT action=POTION 4" $log.txt || { echo "FAIL: D-pad left is not POTION 4"; fail=1; }

  st=$(grep "PAD state" $log.txt)
  # state lines in script order: 1 start, 2-3 X, 4-5 Y, 6-7 L3, 8-9 BACK, 10-12 RB RB LB, 13 inventory+cursor,
  # 14 after the second X, 15 unplugged, 16-17 re-plugged
  line() { echo "$st" | sed -n "${1}p"; }
  echo "$(line 1)" | grep -q "inventory=false character=false skilltree=false quests=false automap=false ui=false" || { echo "FAIL: start state not clean: $(line 1)"; fail=1; }
  echo "$(line 2)" | grep -q "inventory=true" || { echo "FAIL: X did not open the inventory"; fail=1; }
  echo "$(line 3)" | grep -q "inventory=false" || { echo "FAIL: X did not close the inventory"; fail=1; }
  echo "$(line 4)" | grep -q "character=true" || { echo "FAIL: Y did not open the character panel"; fail=1; }
  echo "$(line 5)" | grep -q "character=false" || { echo "FAIL: Y did not close the character panel"; fail=1; }
  echo "$(line 6)" | grep -q "skilltree=true" || { echo "FAIL: L3 did not open the skill tree"; fail=1; }
  echo "$(line 7)" | grep -q "skilltree=false" || { echo "FAIL: L3 did not close the skill tree"; fail=1; }
  echo "$(line 8)" | grep -q "automap=true" || { echo "FAIL: BACK did not show the automap"; fail=1; }
  echo "$(line 9)" | grep -q "automap=false" || { echo "FAIL: BACK did not hide the automap"; fail=1; }

  # skill cycling: RB steps the right skill twice (different each time), LB steps the left skill
  r1=$(line 10 | sed 's/.*right=\("[^"]*"\).*/\1/'); r2=$(line 11 | sed 's/.*right=\("[^"]*"\).*/\1/'); r0=$(line 1 | sed 's/.*right=\("[^"]*"\).*/\1/')
  [ "$r1" != "$r0" ] && [ "$r2" != "$r1" ] || { echo "FAIL: RB did not cycle the right skill ($r0 -> $r1 -> $r2)"; fail=1; }
  l0=$(line 1 | sed 's/.* left=\("[^"]*"\) right=.*/\1/'); l1=$(line 12 | sed 's/.* left=\("[^"]*"\) right=.*/\1/')
  [ "$l1" != "$l0" ] || { echo "FAIL: LB did not cycle the left skill ($l0 -> $l1)"; fail=1; }
  grep -qE "GAMEPAD cycle right skill dir=1" $log.txt || { echo "FAIL: no cycle log line"; fail=1; }

  # left stick: the hero is further right on the screen after walking (world x/y both grow), then comes back
  p=$(grep "AUTOSCRIPT level=1 hero=" $log.txt | sed 's/.*hero=(\([0-9.]*\),\([0-9.]*\)).*/\1 \2/')
  p1=$(echo "$p" | sed -n 2p); p2=$(echo "$p" | sed -n 3p)
  echo "hero before/after the left stick: ($p1) -> ($p2)"
  awk -v a="$p1" -v b="$p2" 'BEGIN{split(a,A," ");split(b,B," ");d=sqrt((A[1]-B[1])^2+(A[2]-B[2])^2); exit !(d>1.5)}' || { echo "FAIL: the left stick did not walk the hero"; fail=1; }
  grep -q "GAMEPAD walk=true" $log.txt || { echo "FAIL: no walk start logged"; fail=1; }
  grep -q "GAMEPAD walk=false" $log.txt || { echo "FAIL: walking did not stop with the stick"; fail=1; }

  # right stick: with the inventory open the stick moves the cursor right of the screen centre, A clicks
  line 13 | grep -q "inventory=true.*ui=true" || { echo "FAIL: inventory not open with ui mode for the cursor test"; fail=1; }
  cx=$(grep "GAMEPAD cursor stopped at" $log.txt | head -1 | sed 's/.*at (\([0-9]*\),.*/\1/')
  [ -n "$cx" ] && [ "$cx" -gt 450 ] || { echo "FAIL: the right stick did not move the cursor right (x=$cx)"; fail=1; }
  grep -qE "GAMEPAD button=A action=CLICK" $log.txt || { echo "FAIL: A did not click"; fail=1; }
  [ -s $pd-inv.png ] || { echo "FAIL: no screenshot"; fail=1; }

  # unplugged: the gamepad no longer opens panels; plugged in again it does
  echo "$(line 14)" | grep -q "inventory=false" || { echo "FAIL: second X did not close the inventory"; fail=1; }
  echo "$(line 15)" | grep -q "inventory=false" || { echo "FAIL: a disconnected pad opened the inventory"; fail=1; }
  echo "$(line 16)" | grep -q "inventory=true" || { echo "FAIL: the re-plugged pad did not open the inventory"; fail=1; }
  echo "$(line 17)" | grep -q "inventory=false" || { echo "FAIL: the re-plugged pad did not close the inventory"; fail=1; }

  echo "screenshots: $pd-*.png"
}
