scenario_name="w51 rare save 2/2 (import the .d2s of phase 1 into a fresh config folder: every tooltip equals the one before the save)"
scenario_warnings_ok=1
w51=$tmp/w51wb
scenario_env() {
  local src; src=$(ls $w51/*.d2s(N[1]))
  mkdir -p $tmp/w51in $tmp/w51cfg2; rm -f $tmp/w51in/*.d2s(N)
  [ -n "$src" ] && cp "$src" $tmp/w51in/Hero.d2s
  [ -f "${OD2_VERIFY_CONFIG_DIR:-}/config.json" ] && cp "$OD2_VERIFY_CONFIG_DIR/config.json" $tmp/w51cfg2/config.json
  # a config folder of its own, so the hero is imported from the .d2s and not found in the .od2 of phase 1
  echo "export OD2_AUTOGAME=\"$tmp/w51in/Hero.d2s\" OD2_CONFIG_DIR=\"$tmp/w51cfg2\""
  echo "export OD2_AUTOSCRIPT='wait:2;say:itemtext after;wait:1;exit'"
}
scenario_check() {
  grep -E "ITEMTEXT|AUTOSCRIPT RESULT" $log.txt | cut -c1-300
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the script did not pass"; fail=1; }
  local before=$tmp/w51-rare-1-save.log.txt
  [ -f "$before" ] || { echo "FAIL: no log of phase 1 ($before): run both scenarios together"; fail=1; return; }
  grep "ITEMTEXT tag=before" "$before" | sed -E 's/.*tag=before #[0-9]+ //' | sort > $tmp/w51-before.txt
  grep "ITEMTEXT tag=after" $log.txt | sed -E 's/.*tag=after #[0-9]+ //' | sort > $tmp/w51-after.txt
  [ -s $tmp/w51-before.txt ] || { echo "FAIL: no tooltips of phase 1"; fail=1; return; }
  if ! diff $tmp/w51-before.txt $tmp/w51-after.txt; then echo "FAIL: tooltips differ after the save and reload"; fail=1; fi
}
