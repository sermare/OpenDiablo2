scenario_name="containers (stash, cube, belt, inventory import; stash object; belt potions)"
scenario_env() { echo 'export OD2_AUTOSTASH=1 OD2_AUTOPANEL=stash,cube,belt,inventory OD2_AUTOBELT=1,2,3,4'; }
scenario_check() {
  grep -E "AUTOPANEL (stash object|panel=[a-z]+ size|spec round)|containers loaded" $log.txt | cut -c1-200
  grep -qE "AUTOPANEL stash object opened the stash" $log.txt || { echo "FAIL: the stash object did not open the stash"; fail=1; }
  for p in stash cube belt inventory; do
    grep -qE "AUTOPANEL panel=$p size=" $log.txt || { echo "FAIL: no $p panel"; fail=1; }
  done
  grep -qE "AUTOPANEL spec round trip: checked=[0-9]+ mismatched=0" $log.txt || { echo "FAIL: saved items do not rebuild identically"; fail=1; }
}
