scenario_name="display bar fill (OD2_BAR_FILL=tile at 1920x1080: the bottom bar art continues beside the 800 wide bar; screenshot size and log line; look at the shot)"
scenario_realtime=1
scenario_timeout=240
# The fill is checked by eye in the screenshot (copied to /tmp/od2-barfill-shot.png, never committed); the pure layout is in
# d2common/d2display/barfill_test.go. OD2_BAR_FILL_TEST=black|tile picks the mode (default tile).
scenario_env() {
  local d=$tmp/barfill; mkdir -p $d/s94 $d/wb94; rm -f $d/s94/*.d2s(N) $d/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $d/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$d/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$d/wb94\" OD2_AUTOSPEED=3"
  fi
  echo "export OD2_DISPLAY=1920x1080 OD2_BAR_FILL=${OD2_BAR_FILL_TEST:-tile}"
  echo "export OD2_AUTOSCRIPT='wait:3;say:capframe $d/shot.png;wait:1;exit'"
}
scenario_check() {
  grep -E "BAR_FILL|AUTOSCRIPT RESULT" $log.txt | cut -c1-200 | head -4
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: barfill script did not pass"; fail=1; }
  grep -q "BAR_FILL mode=${OD2_BAR_FILL_TEST:-tile}" $log.txt || { echo "FAIL: no BAR_FILL log line"; fail=1; }
  [ -s $tmp/barfill/shot.png ] || { echo "FAIL: no screenshot"; fail=1; return; }
  dims=$(python3 -I -c 'import struct,sys; b=open(sys.argv[1],"rb").read(24); print(*struct.unpack(">II", b[16:24]))' $tmp/barfill/shot.png)
  [ "$dims" = "1920 1080" ] || { echo "FAIL: screenshot is $dims, want 1920 1080"; fail=1; }
  cp $tmp/barfill/shot.png /tmp/od2-barfill-${OD2_BAR_FILL_TEST:-tile}.png
}
