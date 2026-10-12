scenario_name="character select order (the expansion list is newest save first: CHARSEL_AddCharacterEntry 0x434060)"
newref="${D2S_SAMPLE_NEW:-$HOME/git/d2s-test/Maricon.d2s}"
scenario_env() {
  mkdir -p $tmp/d2s && cp "$D2S_SAMPLE_BODY" $tmp/d2s/Sample.d2s
  [ -f "$newref" ] && cp "$newref" $tmp/d2s/Newer.d2s
  # the sample is the older save, the new character the newer one
  touch -t 202001010000 $tmp/d2s/Sample.d2s
  touch -t 202401010000 $tmp/d2s/Newer.d2s 2>/dev/null
  echo "unset OD2_AUTOGAME"
  echo "export OD2_D2S_DIR=\"$tmp/d2s\" OD2_AUTOSCREEN=charselect OD2_AUTOSHOT=\"$tmp/charselect-order.png\" OD2_AUTOSHOT_SECONDS=6"
}
scenario_check() {
  grep -E "CHARSELECT slot" $log.txt | cut -c1-200
  if [ -f "$newref" ]; then
    first=$(grep -E "CHARSELECT slot=0 " $log.txt | head -1)
    second=$(grep -E "CHARSELECT slot=1 " $log.txt | head -1)
    [[ -n "$first" && -n "$second" ]] || { echo "FAIL: two characters not listed"; fail=1; }
    # the newer save (the new character) must come first; the older sample second
    [[ "$first" == *"imported=true"* && "$first" != "$second" ]] || { echo "FAIL: slot 0 not an imported hero"; fail=1; }
    [[ "$first" == *"name=\"Maricon\""* && "$second" == *"name=\"NokkaSorc\""* ]] || { echo "FAIL: newest save is not first"; fail=1; }
  fi
  [ -s $tmp/charselect-order.png ] || { echo "FAIL: no character select screenshot"; fail=1; }
}
