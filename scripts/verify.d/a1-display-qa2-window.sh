scenario_name="display QA 2, window (resize, 4:3, UI scale 2, full screen on and off; shots in OD2_QA_DIR; nothing pinned: the logical size follows the window)"
scenario_realtime=1
scenario_timeout=300
# Display QA wave 45 (docs/DISPLAY.md). The window sizes are inside a MacBook screen: a window cannot be larger than the
# monitor, the bigger shapes (ultrawide) are covered by a1-display-qa2-game with OD2_DISPLAY. The flow steps resize:WxH,
# uiscale:N and fullscreen:on|off|toggle are in d2gamescreen/autoflow.go. Shots go to OD2_QA_DIR (outside the repo).
scenario_env() {
  local d=$tmp/qa2win q=${OD2_QA_DIR:-$tmp/qa-shots}/window; mkdir -p $d/s94 $d/wb94 $q $d/cfg; rm -f $d/s94/*.d2s(N) $d/wb94/*.d2s(N)
  cp "${OD2_VERIFY_CONFIG_DIR:-$HOME/Library/Application Support/OpenDiablo2}/config.json" $d/cfg/ 2>/dev/null
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $d/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$d/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$d/wb94\""
  fi
  echo "unset OD2_DISPLAY"
  echo "export OD2_CONFIG_DIR=\"$d/cfg\" OD2_WINDOWED=1"
  local f="wait:3,shot:$q/1-start.png,resize:1280x720,pause:2,shot:$q/2-1280x720.png,resize:1200x900,pause:2,shot:$q/3-4x3-1200x900.png"
  f="$f,resize:1000x600,pause:2,shot:$q/4-1000x600.png,resize:800x600,pause:2,shot:$q/5-800x600.png"
  # the UI scale doubles the pixels of a screen whose window is at least 1600x1200: only full screen has that on a laptop
  f="$f,fullscreen:on,pause:4,shot:$q/6-fullscreen.png,escmenu:main,shot:$q/7-fullscreen-escape.png"
  f="$f,uiscale:2,pause:3,shot:$q/8-fullscreen-uiscale2.png,escmenu:main,shot:$q/9-fullscreen-uiscale2-escape.png,uiscale:1"
  f="$f,fullscreen:off,pause:3,shot:$q/10-windowed-again.png,wait:1,exit"
  echo "export OD2_AUTOFLOW=\"$f\""
}
scenario_check() {
  local q=${OD2_QA_DIR:-$tmp/qa-shots}/window
  grep -E "AUTOFLOW cmd|DISPLAY size|AUTOFLOW .*cannot" $log.txt | cut -c1-160 | head -30
  grep -qE "AUTOFLOW step [0-9]+ exit" $log.txt || { echo "FAIL: the flow did not reach its last step"; fail=1; }
  grep -q "AUTOFLOW .*: cannot" $log.txt && { echo "FAIL: a flow command was refused"; fail=1; }
  echo "QA shots in $q:"; ls $q
}
