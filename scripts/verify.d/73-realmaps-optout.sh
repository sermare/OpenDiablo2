scenario_name="real maps opt-out (OD2_REALMAPS=0 keeps the old placeholder map: no DRLG level is built even with OD2_AUTOLEVEL)"
scenario_env() {
  echo 'export OD2_REALMAPS=0 OD2_AUTOLEVEL=9'
  echo "export OD2_AUTOSCRIPT='wait:3;exit'"
}
scenario_check() {
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: opt-out scenario did not pass"; fail=1; }
  if grep -q "real maze:\|real town:" $log.txt; then echo "FAIL: DRLG ran although OD2_REALMAPS=0"; fail=1; fi
}
