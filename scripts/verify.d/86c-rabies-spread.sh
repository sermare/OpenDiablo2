scenario_name="Rabies spreads between monsters (a dense zombie pack, four casts: the plague hands the infection on)"
# A fresh Druid-skill caster (necromancer save, Rabies granted) bites into a pack of zombies. Each bite infects
# one monster and puts the plague missile on it; the plague's contagion (hit function 53) must infect OTHER
# monsters, so there are more rabies infections than casts (feat/skills-engine-hooks).
# Root cause of the full-verify failures (3 infections for 3 casts): a contagion missile flies ~5 subtiles in a
# RANDOM direction and only infects a neighbour that happens to stand on that line; a carrier dies within a second,
# so a cast gets 2-3 missiles, and on a loaded machine the pack can be arranged so that a whole run misses. That is
# luck, not an engine bug (a missile that hits an infected unit keeps flying, as the exe's hit function 53 returns
# "damage" for it). So the scenario stacks the odds without changing what it proves (monsters other than the cast
# targets get infected through the missile chain): 24 zombies, every cast waits (OD2_AUTOCAST_CROWD) until its target,
# a monster not infected yet, stands among at least three others, and the bite is repeated (OD2_AUTOCAST_SPREAD,
# eight extra casts at most) until two monsters beyond the cast targets have carried the infection.
scenario_env() {
  echo "export OD2_AUTOCAST_CLASS=necromancer OD2_AUTOCAST_MANA=900 OD2_AUTOMONSTER=\"zombie1,24\" OD2_AUTOCAST_CROWD=\"3,rabies\" OD2_AUTOCAST_SPREAD=\"rabies,2\" OD2_AUTOMONSTER_SECONDS=300"
  echo "export OD2_AUTOCAST=\"Rabies,4\""
}
scenario_check() {
  grep -E "AUTOCAST (skill_done|summary)" $log.txt | cut -c1-210 | head -5
  casts=$(grep -cE 'CAST do skill="Rabies" .*ok=true' $log.txt)
  inf=$(grep -cE 'STATE apply by=.* state=rabies' $log.txt)
  echo "rabies casts=$casts infections=$inf contagion missiles=$(grep -cE 'MISSILE create name=rabiescontagion' $log.txt)"
  [ "$casts" -ge 1 ] || { echo "FAIL: Rabies was not cast"; fail=1; }
  [ "$inf" -gt "$casts" ] || { echo "FAIL: the plague infected no other monster ($inf infections for $casts casts)"; fail=1; }
  [ "$inf" -ge $((casts + 2)) ] || { echo "FAIL: the plague infected fewer than two other monsters ($inf infections for $casts casts)"; fail=1; }
  ! grep -qE "panic|fatal error|SIGSEGV" $log.txt || { echo "FAIL: crash in the log"; fail=1; }
}
