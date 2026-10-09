scenario_name="death and new character (OD2_AUTODEATH: die, respawn, corpse, exported .d2s; OD2_AUTONEWCHAR: create a hardcore Necromancer .d2s and diff it with a real new Druid)"
wb=$tmp/writeback-death
# a real new character (header-only .d2s) is the weak hero and the oracle; without
# it the run uses the default save with its life set low
newref="${D2S_SAMPLE_NEW:-$HOME/git/d2s-test/Maricon.d2s}"
scenario_env() {
  mkdir -p $wb   # keeps every exported .d2s out of the Saves folder
  echo "export OD2_D2S_WRITEBACK=\"$wb\" OD2_AUTODEATH=1 OD2_AUTODEATH_LEVEL=10 OD2_AUTODEATH_HP=20 OD2_AUTOMONSTER_DIFF=1"
  echo "export OD2_AUTONEWCHAR=Necromancer,hardcore"
  if [ -f "$newref" ]; then
    cp "$newref" $tmp/DeathHero.d2s
    echo "export OD2_AUTOGAME=\"$tmp/DeathHero.d2s\" OD2_AUTONEWCHAR_REF=\"$newref\""
  fi
}
scenario_check() {
  grep -E "DEATHTEST|DEATH |NEWCHAR|D2S EXPORT reparse" $log.txt | cut -c1-330
  grep -qE "DEATH hero=.* exp_lost=[1-9][0-9]* .* gold_dropped=[1-9]" $log.txt || { echo "FAIL: no death with an experience loss and dropped gold"; fail=1; }
  grep -qE "DEATH respawn pos=\(([0-9]+),([0-9]+)\) town_start=\(\1,\2\) .* gold=0 " $log.txt || { echo "FAIL: no respawn at the town start with no gold"; fail=1; }
  grep -qE "DEATH respawn .* equipment=\[\] " $log.txt || { echo "FAIL: the respawned hero still has its equipment"; fail=1; }
  grep -q "DEATH corpse recovered" $log.txt || { echo "FAIL: corpse not recovered"; fail=1; }
  grep -qE "DEATHTEST summary deaths=1 .* died_flag=false .* corpse_pending=false" $log.txt || { echo "FAIL: bad death summary"; fail=1; }
  grep -qE "D2S EXPORT reparse: .*died=true" $log.txt || { echo "FAIL: the export at the time of death lacks the died flag"; fail=1; }
  grep -qE "D2S EXPORT reparse: .*died=false" $log.txt || { echo "FAIL: the export after the respawn still has the died flag"; fail=1; }
  grep -qE "NEWCHAR parse: .*class=Necromancer .*hardcore=true .*new character" $log.txt || { echo "FAIL: new hardcore Necromancer not created"; fail=1; }
  grep -qE "NEWCHAR first-save reparse: .*class=Necromancer .*hardcore=true .*checksum=ok" $log.txt || { echo "FAIL: first save of the new character"; fail=1; }
  if [ -f "$newref" ]; then
    grep -qE "NEWCHAR oracle byte_identical=true" $log.txt || { echo "FAIL: new character is not byte-identical to the real file"; fail=1; }
    grep -qE "NEWCHAR diff vs .*: unexpected=" $log.txt || { echo "FAIL: no diff report"; fail=1; }
    D2S_SAMPLE_NEW="$newref" go test -run 'NewCharacter|Name|DiffHeaders' ./d2common/d2fileformats/d2s/ 2>&1 | grep -v "ld: warning\|^# " | grep -v "^ok" && fail=1
  fi
}
