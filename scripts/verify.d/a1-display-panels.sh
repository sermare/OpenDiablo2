scenario_name="display 1920x1080 panels (character left + inventory right + skill tree anchored in the 800x600 column, escape menu, screenshots)"
scenario_realtime=1
scenario_timeout=240
# Opens the panels at 1920x1080 and takes screenshots (the panels must stand in the left / right half of the centred 800 wide
# column, the world visible around them). The check is that the panels opened and the shots have the display size; the
# placement is looked at in the screenshots (and by the rectangles of 9f-ui-layout, which are column coordinates).
scenario_env() {
  local d=$tmp/dispp; mkdir -p $d/s94 $d/wb94; rm -f $d/s94/*.d2s(N) $d/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $d/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$d/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$d/wb94\" OD2_AUTOSPEED=3"
  fi
  echo "export OD2_DISPLAY=1920x1080"
  local s="wait:3;press:C;press:I;wait:1;say:capframe $d/both.png;wait:1;press:C;press:I;press:T;wait:1;press:Q;wait:1;say:capframe $d/skill-quest.png;wait:1"
  s="$s;press:Q;press:T;press:Escape;wait:1;say:capframe $d/esc.png;wait:1;press:Escape;exit"
  echo "export OD2_AUTOSCRIPT='$s'"
}
scenario_check() {
  grep -E "INPUT panels|AUTOSCRIPT RESULT" $log.txt | cut -c1-200 | head
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: panels script did not pass"; fail=1; }
  for f in both skill-quest esc; do
    [ -s $tmp/dispp/$f.png ] || { echo "FAIL: no screenshot $f"; fail=1; continue; }
    dims=$(python3 -I -c 'import struct,sys; b=open(sys.argv[1],"rb").read(24); print(*struct.unpack(">II", b[16:24]))' $tmp/dispp/$f.png)
    [ "$dims" = "1920 1080" ] || { echo "FAIL: screenshot $f is $dims, want 1920 1080"; fail=1; }
  done
}
