scenario_name="scripted scenario (walk to Akara, menu opens, inventory panel, exit)"
scenario_env() {
  echo "export OD2_AUTOSCRIPT='wait:1;move:npc=Akara;wait:25;expect:log=NPC menu opened;panel:inventory;wait:1;panel:character;wait:1;panel:close;exit'"
}
scenario_check() {
  grep -E "AUTOSCRIPT" $log.txt | cut -c1-200
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: scripted scenario did not pass"; fail=1; }
}
