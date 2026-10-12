scenario_name="display QA 2, game at OD2_QA2_SIZE with OD2_BAR_FILL (town, panels, automap, waypoint; shots in OD2_QA_DIR; the wave looked at 3440x1440 and 1440x1080 (4:3))"
scenario_realtime=1
scenario_timeout=300
# Display QA wave 45 (docs/DISPLAY.md). OD2_QA2_SIZE=WxH (default 3440x1440) pins the logical size; OD2_BAR_FILL=none|tile|black is
# passed to the game; OD2_QA_DIR=<folder outside the repo> receives the shots, which are looked at by eye.
scenario_env() {
  local size=${OD2_QA2_SIZE:-3440x1440} fill=${OD2_BAR_FILL:-none}
  local d=$tmp/qa2game q=${OD2_QA_DIR:-$tmp/qa-shots}/game-$size-$fill; mkdir -p $d/s94 $d/wb94 $q; rm -f $d/s94/*.d2s(N) $d/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $d/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$d/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$d/wb94\" OD2_AUTOSPEED=3"
  fi
  echo "export OD2_DISPLAY=$size OD2_BAR_FILL=$fill"
  local s="wait:3;say:capframe $q/town.png;panel:character;panel:inventory;wait:1;say:capframe $q/char-inv.png;panel:close"
  s="$s;panel:skills;panel:quest;wait:1;say:capframe $q/quest-skills.png;panel:close"
  s="$s;automap:full;wait:1;say:capframe $q/automap.png;automap:off"
  s="$s;use:Waypoint;wait:2;say:capframe $q/waypoint.png;panel:close;exit"
  echo "export OD2_AUTOSCRIPT='$s'"
}
scenario_check() {
  local size=${OD2_QA2_SIZE:-3440x1440} fill=${OD2_BAR_FILL:-none} f
  local q=${OD2_QA_DIR:-$tmp/qa-shots}/game-$size-$fill
  grep -E "AUTOSCRIPT RESULT|DISPLAY size|BAR_FILL" $log.txt | cut -c1-200 | head
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: display QA script did not pass"; fail=1; }
  for f in town char-inv quest-skills automap waypoint; do
    go run scripts/shotcheck.go size $q/$f.png ${size%x*} ${size#*x} || fail=1
  done
  echo "QA shots in $q:"; ls $q
}
