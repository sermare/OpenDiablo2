scenario_name="object gaps (Den of Evil: locked + trapped chests with and without a key, exploding barrel damage, well pulses, rack base item, storm / exploding / poison / warping / gem shrines)"
scenario_env() {
  # chests are forced locked with spawn handler 2 (trap monster 326); the key arrives after the first object; a gem
  # is in the inventory for the Gem Upgrade shrine; fallen stand next to the hero for the explosion, the storm and the warping shrine
  echo 'export OD2_AUTOLEVEL=9 OD2_AUTOCHEST_SEED=7 OD2_AUTOOBJECT_NEAR=1 OD2_AUTOOBJECT_MONSTER=fallen1,3 OD2_AUTOOBJECT_FORCE=lock,trap=2 OD2_AUTOOBJECT_GIVE=gcv OD2_AUTOOBJECT_GIVE_AFTER=key'
  echo 'export OD2_AUTOOBJECT="5,2;11,1;111,1;106,1;104,1;2,22" OD2_AUTOOBJECT_SHRINE=all OD2_AUTOOBJECT_LIFE=300 OD2_AUTOOBJECT_MANA=10'
}
scenario_check() {
  grep -E "OBJECT (container|locked|unlocked|spawn handler|trap|explosion|hurt|operate .*(class=well|class=rack)|autotest (monster|gave|forced|done))|AUTOGROUND chest|world=" $log.txt | cut -c1-300
  grep -qE "OBJECT autotest done operated=28 " $log.txt || { echo "FAIL: not all 28 objects were operated"; fail=1; }
  # locked chest: the first has no key and stays shut, the second is opened with the key and rolls twice
  grep -qE "OBJECT locked id=5 .*a key is needed" $log.txt || { echo "FAIL: the locked chest did not ask for a key"; fail=1; }
  grep -qE "OBJECT unlocked id=5 " $log.txt || { echo "FAIL: the chest was not unlocked with the key"; fail=1; }
  grep -qE "AUTOGROUND chest id=5 .* rolls=2 locked=true" $log.txt || { echo "FAIL: the locked chest did not roll its treasure twice"; fail=1; }
  [ "$(grep -cE 'AUTOGROUND chest id=5 ' $log.txt)" -eq 1 ] || { echo "FAIL: exactly one chest (the unlocked one) should have dropped"; fail=1; }
  # trapped chest: handler 2 is the firebolt trap monster (monstats 326)
  grep -qE "OBJECT spawn handler 2 \(trap monster 326\) of id=5 scheduled" $log.txt || { echo "FAIL: the trap was not armed"; fail=1; }
  grep -qE "OBJECT trap id=5 handler=2 monster=326 " $log.txt || { echo "FAIL: the trap monster was not created"; fail=1; }
  # exploding barrel: radius 3, every unit in it is tested, at least one is hurt
  grep -qE "OBJECT explosion id=11 name=.* radius=3 targets=[1-9]" $log.txt || { echo "FAIL: the barrel explosion reached nobody"; fail=1; }
  grep -qE "OBJECT hurt (hero|monster) source=barrel .*dmg=[1-9]" $log.txt || { echo "FAIL: the barrel explosion hurt nobody"; fail=1; }
  # well: two pulses, the first one is spent
  grep -qE "OBJECT operate .*class=well .*pulses_left=1 " $log.txt || { echo "FAIL: the well did not keep a second pulse"; fail=1; }
  grep -qE "OBJECT rearmed id=.* pulses=" $log.txt || { echo "FAIL: the well did not return a pulse"; fail=1; }
  # rack and stand: one base item, no treasure class
  n=$(grep -cE "OBJECT operate .*class=rack .*base=\"[a-z0-9]+\" ilvl=[0-9]+ .*drops=1" $log.txt)
  [ "$n" -eq 2 ] || { echo "FAIL: expected 2 rack base items, saw $n"; fail=1; }
  # world shrines
  grep -qE "world=storm life_lost=50% units=[1-9][0-9]* total_life_lost=[1-9]" $log.txt || { echo "FAIL: storm shrine did not take life"; fail=1; }
  grep -qE "world=Exploding Shrine potions=[5-9] dropped=[1-9]" $log.txt || { echo "FAIL: exploding shrine dropped no potions"; fail=1; }
  grep -qE "world=Poison Shrine potions=[5-9] dropped=[1-9]" $log.txt || { echo "FAIL: poison shrine dropped no potions"; fail=1; }
  grep -qE "world=warping monster=" $log.txt || { echo "FAIL: warping shrine made nobody unique"; fail=1; }
  grep -qE "world=gem-upgrade gems_in_inventory=1 code=gfv upgraded=true" $log.txt || { echo "FAIL: gem shrine did not upgrade the chipped amethyst"; fail=1; }
  grep -qE "panic:|SIGSEGV|fatal error" $log.txt && { echo "FAIL: crash during object operation"; fail=1; }
}
