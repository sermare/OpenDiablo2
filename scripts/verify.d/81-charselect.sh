scenario_name="character select lists the imported .d2s (and takes a screenshot)"
scenario_env() {
  mkdir -p $tmp/d2s && cp "$D2S_SAMPLE_BODY" $tmp/d2s/Sample.d2s
  echo "unset OD2_AUTOGAME"
  echo "export OD2_D2S_DIR=\"$tmp/d2s\" OD2_AUTOSCREEN=charselect OD2_AUTOSHOT=\"$tmp/charselect.png\" OD2_AUTOSHOT_SECONDS=6"
}
scenario_check() {
  grep -E "CHARSELECT|AUTOSHOT" $log.txt | tail -5 | cut -c1-200
  grep -qE "CHARSELECT slot=[0-9]+ name=.* class=[A-Za-z]+ level=[0-9]+ hardcore=(true|false) expansion=(true|false) ladder=(true|false) dead=(true|false) imported=true" $log.txt || { echo "FAIL: imported hero not listed"; fail=1; }
  [ -s $tmp/charselect.png ] || { echo "FAIL: no character select screenshot"; fail=1; }
}
