scenario_name="hero stats (equipment bonuses: life/mana/stamina totals, defense, attack rating, damage, resists on the character panel)"
scenario_env() {
  echo "export OD2_AUTOSCRIPT='wait:2;panel:character;wait:1;panel:close;exit'"
}
scenario_check() {
  grep -E "stats: .* level=|PANEL character:|AUTOSCRIPT RESULT" $log.txt | cut -c1-300
  # the real level 94 save stores life 869 / mana 221 / stamina 525 WITHOUT items and 1241 / 464 / 801 as current values:
  # the totals with equipment must be 1241 life (exact), 477 mana (464 is below it) and 801 stamina (exact)
  grep -qE "PANEL character: level=94 .*life=1241/1241 mana=464/477 stamina=801/801" $log.txt || { echo "FAIL: character panel totals are not 1241/1241, 464/477, 801/801"; fail=1; }
  grep -qE "PANEL character: .* def=[1-9][0-9]* ar=[1-9][0-9]* dmg=[0-9]+-[0-9]+ block=[0-9]+ res=fire:-?[0-9]+,cold:-?[0-9]+,light:-?[0-9]+,poison:-?[0-9]+" $log.txt || { echo "FAIL: no def/ar/dmg/res on the character panel line"; fail=1; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: scripted scenario did not pass"; fail=1; }
}
