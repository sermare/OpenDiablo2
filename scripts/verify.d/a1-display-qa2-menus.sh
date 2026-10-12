scenario_name="display QA 2, menus (main menu, credits, cinematics, character select, hero creation, escape menu, options pages; shots in OD2_QA_DIR at OD2_QA2_SIZE)"
scenario_realtime=1
scenario_timeout=300
# Display QA wave 45 (docs/DISPLAY.md): OD2_QA2_SIZE=WxH (default 1920x1080) pins the logical size; OD2_QA_DIR=<folder outside the repo>
# receives the shots, which are looked at by eye. The scenario passes when the whole flow ran and every shot has the window size;
# menu shots must also show the centred 4:3 picture (scripts/shotcheck.go).
scenario_env() {
  local size=${OD2_QA2_SIZE:-1920x1080} W H
  W=${size%x*} H=${size#*x}
  local q=${OD2_QA_DIR:-$tmp/qa-shots}/menus-$size d=$tmp/qa2menus
  rm -rf $d; mkdir -p $d/saves $d/cfg $q
  cp "$D2S_SAMPLE_BODY" $d/saves/Sample.d2s
  cp "${OD2_VERIFY_CONFIG_DIR:-$HOME/Library/Application Support/OpenDiablo2}/config.json" $d/cfg/ 2>/dev/null
  echo "unset OD2_AUTOGAME OD2_D2S_WRITEBACK"
  echo "export OD2_DISPLAY=$size OD2_CONFIG_DIR=\"$d/cfg\" OD2_D2S_DIR=\"$d/saves\""
  local f="mainmenu,shot:$q/1-main.png,goto:credits,pause:3,shot:$q/2-credits.png,goto:cinematics,pause:3,shot:$q/3-cinematics.png"
  f="$f,goto:mainmenu,mainmenu,single,shot:$q/4-charselect.png,new,shot:$q/5-create.png,create:Sorceress:Shotty,difficulty:0"
  f="$f,wait:3,escmenu:main,shot:$q/6-escape.png,escmenu:options,shot:$q/7-options.png,escmenu:video,shot:$q/8-video.png"
  f="$f,escmenu:sound,shot:$q/9-sound.png,escmenu:automap,shot:$q/10-automap.png,escmenu:accessibility,shot:$q/11-access.png,wait:1,exit"
  echo "export OD2_AUTOFLOW=\"$f\""
  echo "$q" > $d/qdir
}
scenario_check() {
  local size=${OD2_QA2_SIZE:-1920x1080} W H
  W=${size%x*} H=${size#*x}
  local q=${OD2_QA_DIR:-$tmp/qa-shots}/menus-$size
  grep -E "AUTOFLOW|DISPLAY size" $log.txt | cut -c1-160 | head -50
  grep -qE "AUTOFLOW step [0-9]+ exit" $log.txt || { echo "FAIL: the flow did not reach its last step"; fail=1; }
  grep -q "AUTOFLOW goto: unknown" $log.txt && { echo "FAIL: unknown goto screen"; fail=1; }
  local f
  for f in 1-main 4-charselect; do
    go run scripts/shotcheck.go menu $q/$f.png $W $H || fail=1
  done
  for f in 2-credits 3-cinematics 6-escape 7-options 8-video 9-sound 10-automap 11-access; do
    go run scripts/shotcheck.go size $q/$f.png $W $H || fail=1
  done
  echo "QA shots in $q:"; ls $q
}
