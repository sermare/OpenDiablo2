scenario_name="tooltips2 (set item name block green, piece list red/green, ethereal+socketed on one line, exact saved-item tooltips; screenshots)"
scenario_realtime=1
scenario_timeout=300
# The revived sample hero (a level 94 sorceress; her Battle Boots are the set item Aldur's Advance, worn alone) hovers
# worn items through the real input path (hoverworn) and a frame is saved after each hover, so the tooltip can be
# looked at; the files stay in $tmp, never in the repo. TOOLTIP lines log the text with the colour tokens.
tt=$tmp/tooltips2
scenario_env() {
  mkdir -p $tt/s94 $tt/wb94; rm -f $tt/s94/*.d2s(N) $tt/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $tt/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$tt/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$tt/wb94\" OD2_AUTOSPEED=3"
  else
    echo "# no revived sample hero, using the default save" >&2
  fi
  local s='wait:2'
  # the worn set item, a worn unique, the socketed helm (a jewel in a unique)
  s="$s;say:hoverworn xtb;wait:1;shot:$tmp/tt-set.png;say:tooltiplog worn xtb"
  s="$s;say:hoverworn ulc;wait:1;shot:$tmp/tt-unique.png;say:tooltiplog worn ulc"
  s="$s;say:hoverworn uhm;wait:1;shot:$tmp/tt-socketed.png;say:tooltiplog worn uhm;exit"
  echo "export OD2_AUTOSCRIPT='$s'"
}
scenario_check() {
  grep -E "TOOLTIP|HOVERWORN|AUTOSCRIPT RESULT" $log.txt | cut -c1-700 | head -20
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the tooltip script did not pass"; fail=1; }
  # the whole name block of a set item is green (INV_BuildSetItemTooltip 0x489440): name and base item
  grep -qE "TOOLTIP where=worn code=xtb .*\[green\]Aldur's Advance \| \[green\]Battle Boots" $log.txt || { echo "FAIL: set item name block is not green"; fail=1; }
  # the set name is gold, the pieces follow: the worn piece green, the others red
  grep -qE "TOOLTIP where=worn code=xtb .*\[gold\]Aldur's Watchtower .*\[red\]" $log.txt || { echo "FAIL: no gold set name followed by missing (red) pieces"; fail=1; }
  grep -qE "TOOLTIP where=worn code=xtb .*\[green\]Aldur's Advance \| .*\[gold\]Aldur's Watchtower .*\[green\]Aldur's Advance" $log.txt || { echo "FAIL: the worn piece is not green in the piece list"; fail=1; }
  grep -qE "TOOLTIP where=worn code=ulc .*\[gold\]Arachnid Mesh \| \[gold\]Spiderweb Sash" $log.txt || { echo "FAIL: unique name block is not gold"; fail=1; }
  grep -qE "TOOLTIP where=worn code=uhm .*Socketed \(1\)" $log.txt || { echo "FAIL: no socket line"; fail=1; }
  for n in set unique socketed; do
    [ -s $tmp/tt-$n.png ] || { echo "FAIL: no screenshot tt-$n.png"; fail=1; }
  done
  go test ./d2core/d2item/d2itemdesc/ -run "TestDescribeSetBonuses|TestSetBonusAddFunc|TestTemperedItem|TestEtherealSocketedIsOneLine|TestUnidentifiedSetItemHasNoSetText" 2>&1 | grep -v "ld: warning\|^# " | grep -v "^ok" && fail=1
  go test ./d2core/d2hero/ -run TestStatItemWeaponAtZeroDurabilityStaysOn 2>&1 | grep -v "ld: warning\|^# " | grep -v "^ok" && fail=1
}
