scenario_name="w47 ui3: the merc screen is held open for a frame (value rows right aligned, captions on their baselines)"
scenario_warnings_ok=1
scenario_timeout=120
# Frame (read by eye, not committed): $tmp/w47-merc.png, taken while the merc screen of the save's mercenary is open.
scenario_env() {
  echo "export OD2_DISPLAY=800x600 OD2_AUTOMERC=skeleton1,4 OD2_AUTOMERC_UI=1 OD2_AUTOMERC_UI_HOLD=1 OD2_AUTOSHOT=$tmp/w47-merc.png OD2_AUTOSHOT_SECONDS=${W47_SHOT_SECONDS:-20}"
}
scenario_check() {
  grep -E "MERC ui (portrait|right click)|AUTOSHOT" $log.txt | cut -c1-200 | tail -4
  [ -s $tmp/w47-merc.png ] || { echo "FAIL: no frame captured"; fail=1; }
}
