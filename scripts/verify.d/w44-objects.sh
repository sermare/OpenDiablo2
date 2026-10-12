scenario_name="w44 objects audit: Jerhyn marker (Lut Gholein), shrine classes by Parm0 (health, mana), level-id shrine roll"
scenario_turbo=1
scenario_warnings_ok=1
# Wave 44. A fresh hero travels to Lut Gholein (the Seven Tombs is not offered yet, so Jerhyn stands at marker 122,
# not at marker 121 next to the arrival) and uses three spawned shrines: objects 85 (Parm0 1) must roll the health
# boost (code 2), 166 (Parm0 2) the mana boost (code 3) and 116 (Parm0 3) one of refill, the boosters or a magic shrine.
h5=$tmp/w44obj
scenario_env() {
  mkdir -p $h5/new $h5/wb; rm -f $h5/new/*.d2s(N) $h5/wb/*.d2s(N)
  {
    echo '#!/bin/zsh'
    echo "export OD2_PORT=$OD2_PORT OD2_AUTONEWCHAR=Necromancer OD2_AUTONEWCHAR_NAME=Hunter OD2_D2S_WRITEBACK=$h5/new OD2_AUTOEXIT=1"
    echo "$tmp/od2 > $h5/newchar.log 2>&1"
  } > $h5/newchar.command
  chmod +x $h5/newchar.command; launch_game $h5/newchar.command; wait_run
  local s="wait:1;say:levelup 12;say:completequest 1 6;travel:2;expect:level=40;wait:4"
  s+=";say:spawnchest 85 166 116;wait:1;click:left@object:85;wait:4;click:left@object:166;wait:4;click:left@object:116;wait:4"
  echo "export OD2_AUTOGAME=\"$h5/new/Hunter.d2s\" OD2_D2S_WRITEBACK=\"$h5/wb\" OD2_AUTOSPEED=3 OD2_NOPOPULATE=1"
  echo "export OD2_AUTOSCRIPT='$s;exit'"
}
scenario_check() {
  grep -E "AUTOSCRIPT step [0-9]+ FAIL|AUTOSCRIPT RESULT|Jerhyn placeholder|added jerhyn|class=shrine" $log.txt | cut -c1-220 | tail -20
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: script did not pass"; fail=1; }
  grep -qE "act town: added jerhyn at his placeholder object \(14\." $log.txt || { echo "FAIL: Jerhyn is not at marker 122 (14.x,29.x)"; fail=1; }
  grep -qE "OBJECT operate id=85 .*class=shrine shrine=2 " $log.txt || { echo "FAIL: object 85 (Parm0 1) did not give the health boost"; fail=1; }
  grep -qE "OBJECT operate id=166 .*class=shrine shrine=3 " $log.txt || { echo "FAIL: object 166 (Parm0 2) did not give the mana boost"; fail=1; }
  grep -qE "OBJECT operate id=116 .*class=shrine shrine=(1|6|7|8|9|10|11|12|13|14|15|17|18|19|20|21|22) " $log.txt || { echo "FAIL: object 116 (Parm0 3) rolled an unexpected shrine"; fail=1; }
}
scenario_timeout=600
