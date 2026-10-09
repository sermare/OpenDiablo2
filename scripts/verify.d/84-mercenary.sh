scenario_name="mercenary (merc spawns, fights, dies, is revived, follows; header survives the export)"
wb=$tmp/writeback-merc
scenario_env() {
  mkdir -p $wb   # keeps the exported file out of the Saves folder
  echo "export OD2_AUTOMERC=skeleton1,4 OD2_AUTOMERC_KILL=1 OD2_D2S_WRITEBACK=\"$wb\""
}
scenario_check() {
  grep -E "AUTOMERC summary|MERC (spawn|hire|death|revive)|D2S EXPORT reparse" $log.txt | cut -c1-300
  for pat in "MERC spawn " "MERC attack .*hit=true" "MERC death " "MERC revive " "AUTOMERC follow dist=" "AUTOMERC summary" \
    "D2S EXPORT reparse: .*merc=type"; do
    grep -qE "$pat" $log.txt || { echo "FAIL: no '$pat' in the merc log"; fail=1; }
  done
}
