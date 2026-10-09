scenario_name="in-game autotest (imports the save, starts it, checks NPC menus and vendor trade)"
scenario_env() {
  echo 'export OD2_AUTOMENU="Akara,Charsi,Gheed,Warriv,Kashya" OD2_AUTOMENU_CHOOSE=Talk'
  echo 'export OD2_AUTOTRADE="Akara,Charsi" OD2_AUTOTRADE_SEED=1 OD2_AUTOTRADE_LEVEL=8'
}
scenario_check() {
  grep -E "imported|equipment:|NPC menu opened" $log.txt | cut -c1-200
  grep -E "AUTOTRADE (buy|sell|repair)" $log.txt | cut -c1-200
  grep -qE "NPC menu opened: npc=\"Akara\"" $log.txt || { echo "FAIL: no Akara menu"; fail=1; }
  for v in Akara Charsi; do
    grep -qE "AUTOTRADE buy vendor=$v .*err=<nil>" $log.txt || { echo "FAIL: no scripted buy at $v"; fail=1; }
    grep -qE "AUTOTRADE sell vendor=$v .*err=<nil>" $log.txt || { echo "FAIL: no scripted sell at $v"; fail=1; }
  done
  grep -qE "AUTOTRADE repair vendor=Charsi .*err=<nil>" $log.txt || { echo "FAIL: no Charsi repair"; fail=1; }
}
