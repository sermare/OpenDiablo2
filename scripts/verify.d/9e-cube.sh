scenario_name="Horadric Cube (Transmute button: gems, runes, magic reroll, sockets, crafting, keys -> portal, invalid contents, runeword socketing)"
scenario_env() { echo 'export OD2_AUTOCUBE=all OD2_AUTOCUBE_SEED=11'; }
scenario_check() {
  grep -E "AUTOCUBE|CUBE (recipes loaded|transmute|portal)|SOCKET" $log.txt | cut -c1-230
  for s in gems runes reroll socket craft keys invalid runeword; do
    grep -qE "AUTOCUBE $s PASS" $log.txt || { echo "FAIL: cube scenario $s"; fail=1; }
  done
  grep -qE "CUBE transmute row=[0-9]+ recipe=\"3 chipped amethysts -> flawed amethyst\"" $log.txt || { echo "FAIL: gem recipe not run through the cube"; fail=1; }
  grep -qE "CUBE transmute row=[0-9]+ recipe=\"3 rune 01 -> rune 02\"" $log.txt || { echo "FAIL: rune recipe not run through the cube"; fail=1; }
  grep -qE "CUBE portal \"Pandemonium Portal\"|AUTOCUBE keys PASS" $log.txt || { echo "FAIL: key recipe"; fail=1; }
  grep -qE "AUTOCUBE done passed=8 failed=0" $log.txt || { echo "FAIL: cube scenarios did not all pass"; fail=1; }
}
