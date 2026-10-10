scenario_name="hero stats (equipment bonuses: life/mana/stamina totals, defense, attack rating, damage, resists on the character panel; mana regenerates on its own)"
scenario_env() {
  echo "export OD2_AUTOSCRIPT='wait:2;panel:character;wait:1;panel:close;wait:5;panel:character;wait:1;panel:close;exit'"
}
scenario_check() {
  grep -E "stats: .* level=|PANEL character:|AUTOSCRIPT RESULT" $log.txt | cut -c1-300
  # the real level 94 save stores life 869 / mana 221 / stamina 525 WITHOUT items and 1241 / 464 / 801 as current values:
  # the totals with equipment must be 1241 life (exact), 477 mana (464 is below it) and 801 stamina (exact)
  grep -qE "PANEL character: level=94 .*life=1241/1241 mana=[0-9]+/477 stamina=801/801" $log.txt || { echo "FAIL: character panel totals are not 1241/1241, x/477, 801/801"; fail=1; }
  grep -qE "PANEL character: .* def=[1-9][0-9]* ar=[1-9][0-9]* dmg=[0-9]+-[0-9]+ block=[0-9]+ res=fire:-?[0-9]+,cold:-?[0-9]+,light:-?[0-9]+,poison:-?[0-9]+" $log.txt || { echo "FAIL: no def/ar/dmg/res on the character panel line"; fail=1; }
  # natural mana regeneration (exe 0x57e6e0: max mana / (ManaRegen 120 * 25 frames) per 25 Hz frame, 477 mana = 3.9 per second):
  # the save's 464 has risen at the first panel (about 2 s of play) and is full (477) at the second (about 8 s of play)
  local m1 m2
  m1=$(grep "PANEL character:" $log.txt | sed -n 1p | sed -E 's/.* mana=([0-9]+)\/477.*/\1/')
  m2=$(grep "PANEL character:" $log.txt | sed -n 2p | sed -E 's/.* mana=([0-9]+)\/477.*/\1/')
  echo "mana regeneration: first panel $m1, second panel $m2 (save 464, maximum 477)"
  case "$m1$m2" in ''|*[!0-9]*) echo "FAIL: could not read the mana of two character panels"; fail=1 ;; *)
    [ "$m1" -gt 464 ] && [ "$m1" -le 477 ] || { echo "FAIL: mana did not rise from the saved 464 by the first panel ($m1)"; fail=1; }
    [ "$m2" -eq 477 ] || { echo "FAIL: mana is not full (477) at the second panel ($m2)"; fail=1; } ;;
  esac
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: scripted scenario did not pass"; fail=1; }
}
