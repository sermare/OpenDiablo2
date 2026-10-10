scenario_name="missile-function skills (Meteor, Volcano, Frozen Orb, Blizzard, Shout, Battle Cry, Bone Wall, Rabies: cast at zombies, kills, no crash)"
# A fresh Necromancer is granted and casts the skills whose behaviour is a missile function read from
# the exe (feat/missile-funcs, feat/skills-final-gaps) at zombies that respawn when all are dead.
scenario_env() {
  local list="Meteor,10;Volcano,10;Frozen Orb,10;Blizzard,10;Shout,5;Battle Cry,5;Bone Wall,10;Rabies,10"
  echo "export OD2_AUTOCAST_CLASS=necromancer OD2_AUTOCAST_MANA=900 OD2_AUTOMONSTER=\"zombie1,5\" OD2_AUTOMONSTER_SECONDS=300"
  echo "export OD2_AUTOCAST=\"$list\""
}
scenario_check() {
  grep -E "AUTOCAST (skill_done|summary)" $log.txt | cut -c1-210 | head -20
  for s in "Meteor" "Volcano" "Frozen Orb" "Blizzard" "Shout" "Battle Cry" "Bone Wall" "Rabies"; do
    grep -qE "CAST do skill=\"$s\" .*ok=true" $log.txt || { echo "FAIL: $s was not cast"; fail=1; }
    grep -qE "AUTOCAST skill_done skill=\"$s\" " $log.txt || { echo "FAIL: $s cast did not finish"; fail=1; }
  done
  grep -qE "MISSILE create name=meteor" $log.txt || { echo "FAIL: no meteor missile"; fail=1; }
  grep -qE "MISSILE create name=volcano" $log.txt || { echo "FAIL: no volcano missile"; fail=1; }
  grep -qE "MISSILE create name=frozenorb" $log.txt || { echo "FAIL: no frozen orb missile"; fail=1; }
  # Bone Wall: the first wall, then the two makers consumed into more pieces (feat/skills-engine-hooks)
  grep -qE "SUMMON bone wall makers .*missiles=2" $log.txt || { echo "FAIL: no bone wall makers launched"; fail=1; }
  grep -qE "MISSILE create name=bonewallmaker" $log.txt || { echo "FAIL: no bonewallmaker missile"; fail=1; }
  [ "$(grep -cE 'SUMMON skill="Bone Wall" .*created=1' $log.txt)" -ge 2 ] || { echo "FAIL: bone wall pieces beyond the first were not summoned"; fail=1; }
  # Rabies: infection, plague missile on the carrier, and (informational) spread to a second monster
  grep -qE "STATE apply by=.* state=rabies" $log.txt || { echo "FAIL: rabies infected nobody"; fail=1; }
  grep -qE "MISSILE create name=rabiesplague" $log.txt || { echo "FAIL: no rabies plague on the carrier"; fail=1; }
  echo "rabies infections: $(grep -cE 'STATE apply by=.* state=rabies' $log.txt), contagion missiles: $(grep -cE 'MISSILE create name=rabiescontagion' $log.txt)"
  grep -qE "AUTOCAST summary .* kills=[1-9]" $log.txt || { echo "FAIL: no kills"; fail=1; }
  ! grep -qE "panic|fatal error|SIGSEGV" $log.txt || { echo "FAIL: crash in the log"; fail=1; }
}
