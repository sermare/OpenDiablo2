scenario_name="monster AI states (forced fear / confuse / charm override the AI, are injected with the forcestate command and restore it; monsters fight monsters)"
# faithful N-Z ports (feat/ai-faithful-b): one in-game subject per family, spawned beside the main subject
fb_families="Vampire SuccubusWitch ZakarumPriest ZakarumZealot OblivionKnight Overseer Regurgitator VileMother VileDog ThornHulk PinHead PutridDefiler QuillMother SiegeBeast ReanimatedHorde Spirit TrappedSoul Trap-Melee SandMaggotQueen Tentacle FrogDemon WillOWisp ShadowWarrior Sarcophagus"
scenario_env() { echo 'export OD2_AUTOAI="skeleton1,state=fear+confuse+charm" OD2_AUTOAI_SECONDS=22'; echo "export OD2_AUTOAI_ALSO=\"${fb_families// /+}\""; }
scenario_check() {
  grep -E "AUTOAI (start|inject|summary)|MONSTER (state|aistate)|forcestate:" $log.txt | cut -c1-220
  grep -q "AUTOAI start ref=skeleton1" $log.txt || { echo "FAIL: AUTOAI did not start"; fail=1; }
  for kind in fear confuse charm; do
    grep -q "AUTOAI inject: forcestate [0-9]* $kind " $log.txt || { echo "FAIL: $kind was not injected"; fail=1; }
    grep -q "MONSTER state .* kind=$kind " $log.txt || { echo "FAIL: no state line for $kind"; fail=1; }
  done
  # fear swaps the think function and gives the class AI back when it ends
  grep -q "MONSTER aistate .* from=Skeleton to=State11+fear" $log.txt || { echo "FAIL: fear did not override the AI"; fail=1; }
  grep -q "MONSTER aistate .* from=State11+fear to=Skeleton" $log.txt || { echo "FAIL: fear did not restore the AI"; fail=1; }
  grep -q "MONSTER aistate .* from=Skeleton to=Skeleton+confuse" $log.txt || { echo "FAIL: confuse not applied"; fail=1; }
  grep -q "MONSTER aistate .* from=Skeleton+confuse to=Skeleton" $log.txt || { echo "FAIL: confuse not restored"; fail=1; }
  grep -q "MONSTER aistate .* to=Skeleton+charm+allied" $log.txt || { echo "FAIL: charm not applied"; fail=1; }
  grep -q "MONSTER aistate .* from=Skeleton+charm+allied to=Skeleton" $log.txt || { echo "FAIL: charm not restored"; fail=1; }
  # monsters against monsters (confused or converted monster and the bystanders)
  grep -qE "MONSTER attack name=.* target=[^ ]+\([0-9]+\) " $log.txt || { echo "FAIL: no monster-versus-monster attack"; fail=1; }
  grep -q "AUTOAI summary" $log.txt || { echo "FAIL: no AUTOAI summary"; fail=1; }
  # archetype coverage (monster-ai-4): the subject runs an implemented AI, every monai.txt name has a Go think
  # function and the ported ground archetypes act (static tests, no game needed)
  grep -q "AUTOAI start ref=skeleton1 .*implemented=true" $log.txt || { echo "FAIL: subject AI not implemented"; fail=1; }
  go test ./d2common/d2monster/ -run 'TestMonaiTableCoverage|TestNoCommonMonsterIdles|TestMonsterAI4' -count=1 2>&1 | grep -v "ignoring duplicate libraries" | tail -5 | grep -q '^ok' || { echo "FAIL: monster AI archetype tests"; fail=1; }
  # every faithful-port family ran as an in-game subject with its implemented think function
  for fam in $fb_families; do
    grep -q "AUTOAI also ref=$fam monster=.* implemented=true" $log.txt || { echo "FAIL: in-game subject $fam missing or not implemented"; fail=1; }
  done
  grep -q "AUTOAI also-summary" $log.txt || { echo "FAIL: no also-summary lines"; fail=1; }
  go test ./d2common/d2monster/ -run 'TestFaithfulB' -count=1 2>&1 | grep -v "ignoring duplicate libraries" | tail -5 | grep -q '^ok' || { echo "FAIL: faithful N-Z AI tests"; fail=1; }
}
