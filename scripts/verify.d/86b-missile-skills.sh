scenario_name="missile-function skills (Meteor, Volcano, Frozen Orb, Blizzard, Shout, Battle Cry: cast at zombies, kills, no crash)"
# A fresh Necromancer is granted and casts the skills whose behaviour is a missile function read from
# the exe (feat/missile-funcs, feat/skills-final-gaps) at zombies that respawn when all are dead.
scenario_env() {
  local list="Meteor,10;Volcano,10;Frozen Orb,10;Blizzard,10;Shout,5;Battle Cry,5"
  echo "export OD2_AUTOCAST_CLASS=necromancer OD2_AUTOCAST_MANA=900 OD2_AUTOMONSTER=\"zombie1,5\" OD2_AUTOMONSTER_SECONDS=300"
  echo "export OD2_AUTOCAST=\"$list\""
}
scenario_check() {
  grep -E "AUTOCAST (skill_done|summary)" $log.txt | cut -c1-210 | head -20
  for s in "Meteor" "Volcano" "Frozen Orb" "Blizzard" "Shout" "Battle Cry"; do
    grep -qE "CAST do skill=\"$s\" .*ok=true" $log.txt || { echo "FAIL: $s was not cast"; fail=1; }
    grep -qE "AUTOCAST skill_done skill=\"$s\" " $log.txt || { echo "FAIL: $s cast did not finish"; fail=1; }
  done
  grep -qE "MISSILE create name=meteor" $log.txt || { echo "FAIL: no meteor missile"; fail=1; }
  grep -qE "MISSILE create name=volcano" $log.txt || { echo "FAIL: no volcano missile"; fail=1; }
  grep -qE "MISSILE create name=frozenorb" $log.txt || { echo "FAIL: no frozen orb missile"; fail=1; }
  grep -qE "AUTOCAST summary .* kills=[1-9]" $log.txt || { echo "FAIL: no kills"; fail=1; }
  ! grep -qE "panic|fatal error|SIGSEGV" $log.txt || { echo "FAIL: crash in the log"; fail=1; }
}
