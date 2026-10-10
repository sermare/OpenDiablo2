scenario_name="mac input (click: left=move/left skill, right and Control+click = right skill, Shift = stand still; press: Tab/I/C/T/Q/Esc; bindkey saved in config.json)"
cfg=$tmp/input-config
scenario_env() {
  mkdir -p $cfg
  # an isolated config dir (a copy of the real config: it holds the MPQ path), so the run never edits the real one
  cp "$HOME/Library/Application Support/OpenDiablo2/config.json" $cfg/config.json 2>/dev/null
  echo "export OD2_CONFIG_DIR=\"$cfg\""
  # clicks land on the upper left of the screen (no NPC or object there); the hero starts with Attack on the left
  # button and Fire Ball on the right. Control+click is the right button on macOS (the game is run on macOS here).
  local s='wait:2;click:left@120,120;click:right@120,120;click:left+ctrl@120,120;click:left+shift@120,120'
  s="$s;skill:left=Fire Bolt;click:left@120,120;click:left+shift@120,120;skill:left=Attack"
  s="$s;automap:off;press:Tab;press:Tab"
  s="$s;press:I;press:I;press:C;press:C;press:T;press:T;press:Q;press:Q;press:Escape;press:Escape"
  s="$s;say:bindkey ToggleInventoryPanel X;press:X;press:X;exit"
  echo "export OD2_AUTOSCRIPT='$s'"
}
scenario_check() {
  grep -E "INPUT|KEYS|AUTOMAP state|AUTOSCRIPT RESULT" $log.txt | cut -c1-200 | head -50
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: input scenario did not pass"; fail=1; }

  # the mouse: in order of the steps (Attack on the left: walk; right; Control+click = right; Shift = stand still; Fire Bolt: cast / stand still)
  actions=$(grep "INPUT world-click" $log.txt | sed 's/.*action=\([a-z-]*\) left_skill=\([0-9]*\)/\1:\2/' | tr -d '\r' | tr '\n' ' ')
  # a plain left click with a spell on the left button: in town (where this scenario runs) a spell that cannot be used there just walks, so move:36 (outside town it is cast-left:36)
  want="move:0 cast-right:0 cast-right:0 stand-still:0 move:36 stand-still:36 "
  [ "$actions" = "$want" ] || { echo "FAIL: world click actions '$actions' want '$want'"; fail=1; }

  # the keyboard
  grep -q "INPUT panels after Tab: .* automap=true" $log.txt || { echo "FAIL: Tab did not open the automap"; fail=1; }
  grep -q "INPUT panels after I: inventory=true" $log.txt || { echo "FAIL: I did not open the inventory"; fail=1; }
  grep -q "INPUT panels after C: .* character=true" $log.txt || { echo "FAIL: C did not open the character panel"; fail=1; }
  grep -q "INPUT panels after T: .* skills=true" $log.txt || { echo "FAIL: T did not open the skill tree"; fail=1; }
  grep -q "INPUT panels after Q: .* quest=true" $log.txt || { echo "FAIL: Q did not open the quest log"; fail=1; }
  grep -q "INPUT panels after Escape: .* menu=true" $log.txt || { echo "FAIL: Esc did not open the game menu"; fail=1; }

  # rebinding: the new key works and is in config.json
  grep -q "KEYS bound event=ToggleInventoryPanel key=X" $log.txt || { echo "FAIL: bindkey"; fail=1; }
  grep -q "INPUT panels after X: inventory=true" $log.txt || { echo "FAIL: the rebound key X did not open the inventory"; fail=1; }
  grep -q '"ToggleInventoryPanel"' $cfg/config.json && grep -A2 '"ToggleInventoryPanel"' $cfg/config.json | grep -q '"X"' \
    || { echo "FAIL: key binding not saved in config.json"; fail=1; }
}
