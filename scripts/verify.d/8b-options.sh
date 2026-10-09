scenario_name="options menu (OD2_AUTOOPTIONS: sound, video and automap pages set through the menu rows, saved to config.json, automap size applied)"
cfg=$tmp/options-config
scenario_env() {
  mkdir -p $cfg
  # an isolated config dir (a copy of the real config: it holds the MPQ path), so the run never edits the real one
  cp "$HOME/Library/Application Support/OpenDiablo2/config.json" $cfg/config.json 2>/dev/null
  echo "export OD2_CONFIG_DIR=\"$cfg\" OD2_AUTOOPTIONS=1"
}
scenario_check() {
  grep -E "OPTIONS" $log.txt | cut -c1-300 | head -40
  grep -q "OPTIONS autotest open=true backend=true" $log.txt || { echo "FAIL: options menu did not open with a backend"; fail=1; }
  grep -q "OPTIONS row sound=60%" $log.txt || { echo "FAIL: sound row"; fail=1; }
  grep -q "OPTIONS row music=80%" $log.txt || { echo "FAIL: music row"; fail=1; }
  grep -q "OPTIONS row gamma=70%" $log.txt || { echo "FAIL: gamma row"; fail=1; }
  grep -q "OPTIONS row npcspeech=TEXT ONLY" $log.txt || { echo "FAIL: npc speech row"; fail=1; }
  grep -q "OPTIONS row automapsize=MINI" $log.txt || { echo "FAIL: automap size row"; fail=1; }
  grep -qE "AUTOMAP state on=true size=mini" $log.txt || { echo "FAIL: the automap ignored the chosen size"; fail=1; }
  grep -qE "OPTIONS persisted ok=true .*sound=60% music=80% .*gamma=70% .*automapsize=MINI .*automapnames=NO" $log.txt \
    || { echo "FAIL: options not saved in config.json"; fail=1; }
  grep -q '"SfxVolume": 0.6' $cfg/config.json || { echo "FAIL: SfxVolume not in config.json"; fail=1; }
  grep -q '"gamma": 7' $cfg/config.json || { echo "FAIL: gamma not in config.json"; fail=1; }
  go test ./d2core/d2config/ 2>&1 | grep -v "ld: warning\|^# " | grep -v "^ok" && fail=1
}
