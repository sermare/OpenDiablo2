scenario_name="display 1920x1080 character select (centred 800x600 content on black, full screen dim)"
scenario_timeout=120
scenario_env() {
  mkdir -p $tmp/d2s && cp "$D2S_SAMPLE_BODY" $tmp/d2s/Sample.d2s
  echo "unset OD2_AUTOGAME"
  echo "export OD2_DISPLAY=1920x1080 OD2_D2S_DIR=\"$tmp/d2s\" OD2_AUTOSCREEN=charselect OD2_AUTOSHOT=\"$tmp/cs1920.png\" OD2_AUTOSHOT_SECONDS=6"
}
scenario_check() {
  grep -E "CHARSELECT|AUTOSHOT" $log.txt | tail -3 | cut -c1-200
  [ -s $tmp/cs1920.png ] || { echo "FAIL: no screenshot"; fail=1; return; }
  dims=$(python3 -I -c 'import struct,sys; b=open(sys.argv[1],"rb").read(24); print(*struct.unpack(">II", b[16:24]))' $tmp/cs1920.png)
  [ "$dims" = "1920 1080" ] || { echo "FAIL: screenshot is $dims, want 1920 1080"; fail=1; }
}
