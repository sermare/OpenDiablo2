scenario_name="skill bar (popup grid, F-key hotkeys while hovering, select + click, skill points with prerequisites, level-up, d2s header round trip)"
sb=$tmp/skillbar
wb=$tmp/writeback-skillbar
scenario_env() {
  mkdir -p $wb   # keeps the exported .d2s out of the Saves folder
  # OD2_AUTOMONSTER_HOLD: the monster test quits the game a few game seconds after its last monster died, and the scripted
  # Fire Ball kills the zombie; on a loaded machine (a long frame at the x4 clock) that ended the game in the middle of
  # the script, before the skill tree steps. With HOLD the script ends the run itself with its exit step.
  echo "export OD2_D2S_WRITEBACK=\"$wb\" OD2_AUTOMONSTER=\"zombie1,1\" OD2_AUTOMONSTER_PASSIVE=1 OD2_AUTOMONSTER_HOLD=1"
  # right popup: look at it, hover Fire Wall, F8 assigns it; click Meteor (right skill); left popup: click Ice Bolt;
  # F8 / F2 select the hotkeyed skills; use=right casts the right skill through the skill pipeline; the skill tree:
  # hover + F6 assigns, level-up grants points, spending needs the prerequisites (Thunder Storm is refused)
  echo "export OD2_AUTOSCRIPT='wait:1;skill:popup=right;wait:1;skill:hover=Fire Wall;wait:1;say:capframe $sb-right.png;press:F8;wait:1;say:capframe $sb-right-hotkey.png;skill:click=Meteor;wait:1;skill:popup=left;wait:1;skill:hover=Fire Bolt;wait:1;say:capframe $sb-left.png;skill:click=Ice Bolt;wait:1;press:F8;press:F2;skill:use=right;wait:2;hotkey:F7=Fire Bolt@left;press:F7;skill:left=Attack;say:levelup 4;panel:skills;wait:1;skill:hover=Inferno;wait:1;say:capframe $sb-tree.png;press:F6;wait:1;skill:spend=Charged Bolt;skill:spend=Lightning;skill:spend=Chain Lightning;skill:nospend=Thunder Storm;wait:1;say:capframe $sb-tree-spent.png;panel:close;exit'"
}
scenario_check() {
  grep -E "SKILLBAR|d2s: .* skills left|CAST do|D2S EXPORT reparse|AUTOSCRIPT RESULT" $log.txt | grep -v "click-sound\|state (" | cut -c1-330 | head -50

  # the real save: Fire Ball (47) on the right button, Attack on the left, seven hotkeys, the same swap set
  grep -qE "d2s: NokkaSorc skills left=0 right=47 swap=0/47 hotkeys=\[59 47 54 42 55 40 43 -1 -1 -1 -1 -1 -1 -1 -1 -1\]" $log.txt || { echo "FAIL: the assigned/left/right/swap skills of the save were not imported"; fail=1; }
  grep -qE "SKILLBAR state \(.*\) left=Attack\(0\) right=Fire Ball\(47\)" $log.txt || { echo "FAIL: the game did not start with Attack/Fire Ball"; fail=1; }

  # the popup lays out the skills of the page rows; Fire Wall / Meteor are in the right one, Teleport is not in the left one
  grep -qE "SKILLBAR popup hand=right icons=[1-9][0-9]* \[.*Attack@r0c0.*Fire Ball@r1c" $log.txt || { echo "FAIL: right popup grid"; fail=1; }
  grep -qE "SKILLBAR popup hand=left icons=[1-9][0-9]* \[.*Attack@r0c0" $log.txt || { echo "FAIL: left popup grid"; fail=1; }
  grep "SKILLBAR popup hand=left" $log.txt | grep -q "Teleport" && { echo "FAIL: a right-only skill is in the left popup"; fail=1; }

  # hotkeys: F-key over a hovered icon assigns; F-key without it selects; clicks select and click
  grep -qE "SKILLBAR hotkey key=F8 slot=7 skill=\"Fire Wall\" id=51 hand=right" $log.txt || { echo "FAIL: hover + F8 did not assign Fire Wall"; fail=1; }
  grep -qE "SKILLBAR select hand=right skill=\"Meteor\"" $log.txt || { echo "FAIL: click on Meteor did not select it"; fail=1; }
  grep -qE "SKILLBAR select hand=left skill=\"Ice Bolt\"" $log.txt || { echo "FAIL: click on Ice Bolt did not select it"; fail=1; }
  grep -qE "SKILLBAR press key=F8 slot=7 skill=\"Fire Wall\"" $log.txt || { echo "FAIL: pressing F8 did not find Fire Wall"; fail=1; }
  grep -qE "SKILLBAR press key=F2 slot=1 skill=\"Fire Ball\"" $log.txt || { echo "FAIL: pressing F2 did not find Fire Ball"; fail=1; }
  grep -qE "SKILLBAR select hand=right skill=\"Fire Ball\" id=47" $log.txt || { echo "FAIL: F2 did not make Fire Ball the right skill"; fail=1; }
  grep -qE "SKILLBAR click-sound" $log.txt || { echo "FAIL: no click sound on selection"; fail=1; }
  grep -qE "SKILLBAR use hand=right skill=\"Fire Ball\"" $log.txt || { echo "FAIL: use=right did not use Fire Ball"; fail=1; }
  grep -qE "CAST do skill=\"Fire Ball\" .*ok=true" $log.txt || { echo "FAIL: the right skill was not cast through the skill pipeline"; fail=1; }
  grep -qE "SKILLBAR hotkey key=F7 slot=6 skill=\"Fire Bolt\" id=36 hand=left" $log.txt || { echo "FAIL: hotkey step"; fail=1; }
  grep -qE "SKILLBAR select hand=left skill=\"Fire Bolt\" id=36" $log.txt || { echo "FAIL: F7 did not select Fire Bolt on the left"; fail=1; }
  grep -qE "SKILLBAR hotkey key=F6 slot=5 skill=\"Inferno\" id=41 hand=right" $log.txt || { echo "FAIL: skill tree hover + F6 did not assign Inferno"; fail=1; }

  # skill points: level-up grants them, the prerequisites decide
  grep -qE "SKILLBAR levelup \+4 level=98 unused_points=4 stat_points=20" $log.txt || { echo "FAIL: level-up did not grant 4 skill points"; fail=1; }
  grep -qE "SKILLBAR spend skill=\"Charged Bolt\" id=38 level=1 unused_points=3" $log.txt || { echo "FAIL: Charged Bolt not learned"; fail=1; }
  grep -qE "SKILLBAR spend skill=\"Lightning\" id=49 level=1 unused_points=2" $log.txt || { echo "FAIL: Lightning (needs Charged Bolt) not learned"; fail=1; }
  grep -qE "SKILLBAR spend skill=\"Chain Lightning\" id=53 level=1 unused_points=1" $log.txt || { echo "FAIL: Chain Lightning (needs Lightning) not learned"; fail=1; }
  grep -qE "SKILLBAR spend FAIL skill=\"Thunder Storm\" id=57: Thunder Storm needs a point in Nova" $log.txt || { echo "FAIL: Thunder Storm was not refused for the missing Nova"; fail=1; }

  # everything is in the exported .d2s header and skill block
  grep "D2S EXPORT reparse" $log.txt | tail -1 | grep -qE "skills=left0/right47 swap=0/47 hotkeys=\[59 47 54 42 55 41 36 51 -1 -1 -1 -1 -1 -1 -1 -1\] spent=[0-9]+ unused=1 " || { echo "FAIL: exported .d2s lacks the hotkeys/skills/points"; fail=1; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: skill bar script did not pass"; fail=1; }

  for s in right right-hotkey left tree tree-spent; do
    [ -s $sb-$s.png ] || { echo "FAIL: no screenshot $s"; fail=1; }
  done
  echo "screenshots: $sb-*.png"
}
