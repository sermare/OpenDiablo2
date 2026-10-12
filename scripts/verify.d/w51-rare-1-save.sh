scenario_name="w51 rare save 1/2 (give rare, magic, crafted, unique and set items, log their tooltips, exit: the exported .d2s holds all of them)"
# Items made in the game used to lose every rare/crafted item in the .d2s export. Phase 1 gives items of every quality,
# logs their tooltips (ITEMTEXT tag=before) and exits so that the .d2s is written to $tmp/w51wb. Phase 2
# (w51-rare-2-reload.sh, run right after this one: OD2_VERIFY_ONLY='w51-rare-*') imports that file into a fresh config
# folder and compares the tooltips.
scenario_warnings_ok=1
w51=$tmp/w51wb
scenario_env() {
  mkdir -p $w51; rm -f $w51/*.d2s(N)
  echo "export OD2_D2S_WRITEBACK=\"$w51\" OD2_AUTOSAVE=1"
  echo "export OD2_AUTOSCRIPT='wait:2;say:clearinv;say:giveitemq cap 6;say:giveitemq lsd 6;say:giveitemq uap 6;say:giveitemq cap 4;say:giveitemq lsd 8;say:giveitemq uap 7;say:giveitemq bhm 5;say:giveitemq cap 3;say:itemtext before;wait:1;exit'"
}
scenario_check() {
  grep -E "GIVEITEM|ITEMTEXT|D2S EXPORT|D2S export|AUTOSCRIPT RESULT" $log.txt | cut -c1-300
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the script did not pass"; fail=1; }
  ls $w51/*.d2s(N) >/dev/null 2>&1 || { echo "FAIL: no exported .d2s"; fail=1; return; }
  local n; n=$(grep -c "ITEMTEXT tag=before" $log.txt)
  [ "$n" -ge 8 ] || { echo "FAIL: expected 8 items in the inventory, found $n"; fail=1; }
  grep -q "not written" $log.txt && { echo "FAIL: an item was reported as not written"; fail=1; }
  grep -q "D2S EXPORT reparse: .*checksum=ok" $log.txt || { echo "FAIL: exported .d2s did not re-parse"; fail=1; }
  # every new item must be in the re-parsed file
  local last nf
  last=$(awk '/D2S EXPORT path=/ {blk=""} {blk = blk "\n" $0} END {print blk}' $log.txt)
  nf=$(echo "$last" | grep -cE "D2S EXPORT new item .* found=false")
  [ "$nf" -eq 0 ] || { echo "FAIL: $nf new items missing from the exported file"; fail=1; }
}
