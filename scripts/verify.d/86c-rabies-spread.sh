scenario_name="Rabies spreads between monsters (a dense zombie pack, three casts: the plague hands the infection on)"
# A fresh Druid-skill caster (necromancer save, Rabies granted) bites into a pack of zombies. Each bite infects
# one monster and puts the plague missile on it; the plague's contagion (hit function 53) must infect OTHER
# monsters, so there are more rabies infections than casts (feat/skills-engine-hooks).
scenario_env() {
  echo "export OD2_AUTOCAST_CLASS=necromancer OD2_AUTOCAST_MANA=900 OD2_AUTOMONSTER=\"zombie1,16\" OD2_AUTOMONSTER_SECONDS=300"
  echo "export OD2_AUTOCAST=\"Rabies,3\""
}
scenario_check() {
  grep -E "AUTOCAST (skill_done|summary)" $log.txt | cut -c1-210 | head -5
  casts=$(grep -cE 'CAST do skill="Rabies" .*ok=true' $log.txt)
  inf=$(grep -cE 'STATE apply by=.* state=rabies' $log.txt)
  echo "rabies casts=$casts infections=$inf contagion missiles=$(grep -cE 'MISSILE create name=rabiescontagion' $log.txt)"
  [ "$casts" -ge 1 ] || { echo "FAIL: Rabies was not cast"; fail=1; }
  [ "$inf" -gt "$casts" ] || { echo "FAIL: the plague infected no other monster ($inf infections for $casts casts)"; fail=1; }
  ! grep -qE "panic|fatal error|SIGSEGV" $log.txt || { echo "FAIL: crash in the log"; fail=1; }
}
