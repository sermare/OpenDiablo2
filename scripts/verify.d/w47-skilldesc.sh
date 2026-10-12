scenario_name="w47 skilldesc (character screen hand skill boxes run the skill's descdam function: Fire Bolt / Meteor damage ranges, the skill popup and tree tooltips; frames)"
scenario_turbo=1
wb=$tmp/writeback-w47
# Frames (read by eye, not committed): $tmp/w47-char.png (damage boxes), $tmp/w47-popup.png (skill popup hover),
# $tmp/w47-tree.png (skill tree hover). The HANDBOX lines carry the numbers: descdam=true means the ported descdam 5 / 7
# function (d2statlist.DescDamBox) produced the damage text.
scenario_env() {
  mkdir -p $wb
  local s="wait:2;skill:left=Fire Bolt;skill:right=Meteor;panel:character;wait:1;say:capframe $tmp/w47-char.png;panel:close"
  s="$s;skill:popup=right;wait:1;skill:hover=Meteor;wait:1;say:capframe $tmp/w47-popup.png;skill:popup=left"
  s="$s;panel:skills;wait:2;skill:hover=Fire Ball;wait:1;say:capframe $tmp/w47-tree.png;panel:close;exit"
  echo "export OD2_D2S_WRITEBACK=\"$wb\" OD2_AUTOSCRIPT='$s'"
}
scenario_check() {
  grep -E "HANDBOX|AUTOSCRIPT step [0-9]+ FAIL|AUTOSCRIPT RESULT" $log.txt | cut -c1-260 | tail -8
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: script did not pass"; fail=1; }
  grep -qE 'HANDBOX hand=0 skill="Fire Bolt" descdam=true damage=[0-9]+-[0-9]+' $log.txt || { echo "FAIL: Fire Bolt did not use its descdam function"; fail=1; }
  grep -qE 'HANDBOX hand=1 skill="Meteor" descdam=true damage=[0-9]+-[0-9]+' $log.txt || { echo "FAIL: Meteor did not use its descdam function"; fail=1; }
  [ -n "${OD2_KEEP_FRAMES:-}" ] && cp $tmp/w47-*.png "$OD2_KEEP_FRAMES"/ 2>/dev/null   # optional: keep the frames for a look
  for f in char popup tree; do
    [ -s $tmp/w47-$f.png ] || { echo "FAIL: no frame $f"; fail=1; }
  done
}
