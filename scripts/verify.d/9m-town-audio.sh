scenario_realtime=1
scenario_unmuted=1
scenario_timeout=240
scenario_name="town audio (Rogue Encampment: town music and ambience reach the device, menu clicks play UI sounds)"
cfg=$tmp/town-audio-config
# the options autotest clicks menu rows (UI sounds) and saves its settings: use an isolated config dir (a copy of the
# real config, which holds the MPQ path) so the run never edits the real one
scenario_env() {
  mkdir -p $cfg
  cp "$HOME/Library/Application Support/OpenDiablo2/config.json" $cfg/config.json 2>/dev/null
  echo 'export OD2_REALMAPS=1 OD2_SOUNDLOG=1'
  echo "export OD2_CONFIG_DIR=\"$cfg\" OD2_AUTOOPTIONS=1"
}
scenario_check() {
  grep -a "AUDIOSTAT" $log.txt | cut -c1-600
  grep -aq "AUDIOSTAT" $log.txt || { echo "FAIL: no AUDIOSTAT line"; fail=1; return; }
  local line=$(grep -a "AUDIOSTAT" $log.txt | tail -1)
  local direct=$(echo "$line" | sed -n 's/.*direct_plays=\([0-9]*\).*/\1/p')
  echo "AUDIO ui direct_plays=${direct:-0}"
  [ "${direct:-0}" -ge 5 ] || { echo "FAIL: menu clicks played no UI sounds (direct_plays=${direct:-0})"; fail=1; }
  local c fl
  for c in music ambience; do
    fl=$(echo "$line" | sed -n "s/.* $c=[0-9]*(vol [0-9.]* flow \([0-9]*\)B amp \([0-9]*\)).*/\1 \2/p")
    echo "AUDIO $c flow/amp=$fl"
    [ "${${=fl}[1]:-0}" -gt 0 ] && [ "${${=fl}[2]:-0}" -gt 0 ] || { echo "FAIL: town $c did not reach the device"; fail=1; }
  done
}
