scenario_name="gamble window (Gheed) and Cain identify service (autotest, no clicking)"
scenario_env() {
  echo 'export OD2_AUTOGAMBLE=Gheed OD2_AUTOIDENTIFY=1 OD2_AUTOTRADE_SEED=3 OD2_AUTOTRADE_LEVEL=30'
}
scenario_check() {
  grep -E "AUTOGAMBLE|AUTOIDENTIFY" $log.txt | cut -c1-220
  grep -qE "AUTOGAMBLE stock vendor=Gheed #0 code=rin " $log.txt || { echo "FAIL: gamble slot 0 is not a ring"; fail=1; }
  grep -qE "AUTOGAMBLE stock vendor=Gheed #1 code=amu " $log.txt || { echo "FAIL: gamble slot 1 is not an amulet"; fail=1; }
  grep -qE "AUTOGAMBLE stock vendor=Gheed #13 " $log.txt || { echo "FAIL: gamble stock has fewer than 14 items"; fail=1; }
  grep -qE "AUTOGAMBLE buy vendor=Gheed .*identified_before=false .*err=<nil>" $log.txt || { echo "FAIL: no scripted gamble buy"; fail=1; }
  grep -qE "AUTOGAMBLE result vendor=Gheed .*identified=true" $log.txt || { echo "FAIL: gamble item not revealed"; fail=1; }
  grep -qE "AUTOGAMBLE restock vendor=Gheed .*changed=true" $log.txt || { echo "FAIL: no restock after four minutes"; fail=1; }
  grep -qE "AUTOIDENTIFY one .*cost=100 .*identified=true .*err=<nil>" $log.txt || { echo "FAIL: single identify"; fail=1; }
  grep -qE "AUTOIDENTIFY use ibk .*identified=true .*err=<nil>" $log.txt || { echo "FAIL: tome of identify"; fail=1; }
  grep -qE "AUTOIDENTIFY use isc .*identified=true .*err=<nil>" $log.txt || { echo "FAIL: scroll of identify"; fail=1; }
  grep -qE "AUTOIDENTIFY all .*unidentified_after=0 " $log.txt || { echo "FAIL: identify all"; fail=1; }
}
