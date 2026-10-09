scenario_name="world objects (all 22 shrine types, well, weapon rack, armor stand, barrel, chest; effects, expiry, re-arm)"
scenario_env() {
  echo 'export OD2_AUTOOBJECT="2,22;111,1;106,1;104,1;7,1;139,1;4,1" OD2_AUTOOBJECT_SHRINE=all OD2_AUTOOBJECT_LIFE=300 OD2_AUTOOBJECT_MANA=10'
}
scenario_check() {
  grep -E "OBJECT (operate|effect expired|rearmed|autotest (start|done|fast))" $log.txt | cut -c1-260
  grep -qE "OBJECT autotest done operated=28 " $log.txt || { echo "FAIL: not all 28 objects were operated"; fail=1; }
  n=$(grep -cE "OBJECT operate .*class=shrine shrine=[0-9]+" $log.txt)
  [ "$n" -eq 22 ] || { echo "FAIL: expected 22 shrine operations, saw $n"; fail=1; }
  grep -qE "OBJECT operate .*shrine=6 \"Armor Boost\".*buff state=128 stats=\[.*171.*\] duration=96s" $log.txt || { echo "FAIL: armor shrine buff"; fail=1; }
  grep -qE "OBJECT operate .*shrine=1 \"Refill\".*life=[0-9]+/[0-9]+ mana=[0-9]+/[0-9]+" $log.txt || { echo "FAIL: refill shrine"; fail=1; }
  grep -qE "OBJECT operate .*class=well .*after\[life=([0-9]+)/\1 " $log.txt || { echo "FAIL: well did not restore life"; fail=1; }
  grep -qE "OBJECT effect expired shrine:Armor Boost state=128" $log.txt || { echo "FAIL: armor shrine did not expire"; fail=1; }
  grep -qE "OBJECT rearmed id=" $log.txt || { echo "FAIL: no shrine or well re-armed"; fail=1; }
  grep -qE "OBJECT operate .*class=rack .*drops=" $log.txt || { echo "FAIL: weapon rack did not drop"; fail=1; }
}
