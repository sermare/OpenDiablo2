scenario_name="positional monster sounds (near and far ring, volume and pan per sound)"
scenario_env() { echo 'export OD2_AUTOMONSTER="fallen1,4" OD2_AUTOMONSTER_FAR=40 OD2_AUTOMONSTER_SECONDS=20'; }
scenario_check() {
  grep -E "AUTOMONSTER summary" $log.txt | cut -c1-300
  grep -E "SOUNDAT kind=(attack|weapon|hit|death|footstep)" $log.txt | head -3 | cut -c1-260
  grep -q "SOUNDAT kind=.* dist=[0-9]* radius=[0-9]* gain=[0-9.]* vol=[0-9.]* pan=[-+][0-9.]*" $log.txt || { echo "FAIL: no positional SOUNDAT lines"; fail=1; }
  grep -q "SOUNDAT kind=\(attack\|weapon\|death\|footstep\) who=\"Fallen\"" $log.txt || { echo "FAIL: no monster sounds"; fail=1; }
  grep -q "AUTOMONSTER summary .* sounds=[1-9]" $log.txt || { echo "FAIL: no sound summary"; fail=1; }
}
