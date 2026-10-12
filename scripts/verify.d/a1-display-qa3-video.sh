scenario_name="display QA 3, video options (VIDEO RESOLUTION row applied live and saved, FULL SCREEN row, UI scale 2 in a small window; shots in OD2_QA_DIR)"
scenario_realtime=1
scenario_timeout=300
# Wave 49 (docs/DISPLAY.md): nothing pinned, windowed start. videores:WxH is the flow step that does what choosing a value of the
# VIDEO RESOLUTION row does (resize the window live, leave full screen, save Display of config.json). Shots go to OD2_QA_DIR.
scenario_env() {
  local d=$tmp/qa3video q=${OD2_QA_DIR:-$tmp/qa-shots}/video; mkdir -p $d/s94 $d/wb94 $q $d/cfg; rm -f $d/s94/*.d2s(N) $d/wb94/*.d2s(N)
  cp "${OD2_VERIFY_CONFIG_DIR:-$HOME/Library/Application Support/OpenDiablo2}/config.json" $d/cfg/ 2>/dev/null
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $d/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$d/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$d/wb94\""
  fi
  echo "unset OD2_DISPLAY"
  echo "export OD2_CONFIG_DIR=\"$d/cfg\" OD2_WINDOWED=1"
  local f="wait:3,escmenu:video,shot:$q/1-start.png,videores:1280X720,pause:2,shot:$q/2-1280x720.png"
  f="$f,videores:1024X768,pause:2,shot:$q/3-1024x768.png,uiscale:2,pause:2,shot:$q/4-1024x768-uiscale2.png,uiscale:1"
  f="$f,fullscreen:on,pause:3,shot:$q/5-fullscreen.png,uiscale:2,pause:3,shot:$q/6-fullscreen-uiscale2.png,uiscale:1"
  f="$f,videores:1280X720,pause:3,shot:$q/7-back-to-window.png,wait:1,exit"
  echo "export OD2_AUTOFLOW=\"$f\""
}
scenario_check() {
  local q=${OD2_QA_DIR:-$tmp/qa-shots}/video
  grep -E "AUTOFLOW cmd|AUTOFLOW .*cannot|AUTOFLOW videores" $log.txt | cut -c1-160 | head -30
  grep -qE "AUTOFLOW step [0-9]+ exit" $log.txt || { echo "FAIL: the flow did not reach its last step"; fail=1; }
  grep -q "AUTOFLOW videores" $log.txt && { echo "FAIL: a videores step was refused"; fail=1; }
  grep -E '"(Width|Height)"' $tmp/qa3video/cfg/config.json | tr -d '\n'; echo
  echo "QA shots in $q:"; ls $q
}
