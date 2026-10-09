scenario_name="equip rules (OD2_AUTOEQUIP: body locations, requirements, class, two-handers, weapon switch, recalculated totals; OD2_AUTOMONSTER fight: durability loss)"
wb=$tmp/writeback-equip
scenario_env() {
  mkdir -p $wb   # keeps the exported .d2s out of the Saves folder
  echo "export OD2_D2S_WRITEBACK=\"$wb\" OD2_AUTOEQUIP=1 OD2_AUTOMONSTER=\"fallen1,pack\" OD2_AUTOMONSTER_SECONDS=20 OD2_AUTOMONSTER_DIFF=0 OD2_DURABILITY_CHANCE=60"
}
scenario_check() {
  grep -E "EQUIP |DURABILITY|SOUND hero voice|equipment loaded" $log.txt | cut -c1-330
  # every worn item of the real save is on and the totals match scenario 89's
  grep -q "equipment loaded: worn=[1-9]" $log.txt || { echo "FAIL: the worn items of the save were not loaded"; fail=1; }
  # taking a worn item off and putting it back restores the totals
  grep -qE "EQUIP auto roundtrip slot=.* put back .* ok=true restored=true" $log.txt || { echo "FAIL: no worn item went off and back with restored totals"; fail=1; }
  grep -qE "EQUIP auto roundtrip slot=.* took off .*def=" $log.txt || { echo "FAIL: taking an item off did not recalculate the totals"; fail=1; }
  # a rule violation is refused with its reason
  grep -qE "EQUIP auto wrongslot .* ok=false reason=\"item does not fit this body location\"" $log.txt || { echo "FAIL: wrong slot not refused"; fail=1; }
  grep -qE "EQUIP auto strength item=.* ok=false reason=\"not enough strength\"" $log.txt || grep -q "EQUIP auto strength skipped" $log.txt || { echo "FAIL: strength requirement not enforced"; fail=1; }
  grep -qE "EQUIP auto class item=.* ok=false reason=\"item is restricted to another class\"" $log.txt || { echo "FAIL: class restriction not enforced"; fail=1; }
  grep -qE "EQUIP auto unidentified .* ok=false" $log.txt || { echo "FAIL: unidentified item equipped"; fail=1; }
  grep -qE "EQUIP auto hands item=sst .* ok=true .*displaced=\[lhand\]" $log.txt || { echo "FAIL: two-handed weapon did not push out the shield"; fail=1; }
  grep -qE "EQUIP auto rings item=rin slot=lring ok=true" $log.txt || { echo "FAIL: second ring not accepted"; fail=1; }
  grep -qE "EQUIP auto rings item=rin slot=neck ok=false" $log.txt || { echo "FAIL: ring accepted as an amulet"; fail=1; }
  # W switches the weapon sets and the switch is persisted
  grep -qE "EQUIP auto swap to set II active_arms=1" $log.txt || { echo "FAIL: weapon switch"; fail=1; }
  grep -qE "EQUIP auto swap back active_arms=0 same_stats=true" $log.txt || { echo "FAIL: switching back"; fail=1; }
  grep -qE "EQUIP auto restored=true totals_equal=true" $log.txt || { echo "FAIL: the scenario did not leave the body as it found it"; fail=1; }
  # the fight wears things down
  grep -qE "DURABILITY strike item=.* [0-9]+->[0-9]+/[0-9]+" $log.txt || { echo "FAIL: no weapon durability loss in the fight"; fail=1; }
  grep -qE "DURABILITY hit item=.* [0-9]+->[0-9]+/[0-9]+" $log.txt || { echo "FAIL: no armor durability loss from simulated hits"; fail=1; }
  # a broken armor piece stops counting: the defense drops
  grep -qE "EQUIP auto durability break item=.* broken=true before=\[def=([0-9]+) .*after=\[def=" $log.txt || { echo "FAIL: armor did not break"; fail=1; }
  go test ./d2common/d2equip/ 2>&1 | grep -v "ld: warning\|^# " | grep -v "^ok" && fail=1
}
