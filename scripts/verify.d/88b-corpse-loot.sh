scenario_name="corpse looting (OD2_AUTODEATH with the real level-94 sample hero: the worn items move onto the corpse and the .d2s corpse section, the walk-on loot gives them back)"
scenario_realtime=1
scenario_timeout=300
# the sample hero wore a shield it lacks the strength for: it comes back into the inventory, and the exporter warns about every body-to-container move (known, w50 notes)
scenario_warnings_ok=1
cl=$tmp/corpseloot
scenario_env() {
  mkdir -p $cl/s94 $cl/wb; rm -f $cl/s94/*.d2s(N) $cl/wb/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $cl/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$cl/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$cl/wb\""
  else
    echo "# no revived sample hero, using the default save" >&2
  fi
  echo 'export OD2_AUTODEATH=1 OD2_AUTODEATH_CLEARINV=1 OD2_AUTODEATH_HP=20 OD2_AUTOMONSTER_DIFF=1'
}
scenario_check() {
  grep -E "DEATHTEST|DEATH |D2S EXPORT reparse" $log.txt | cut -c1-300
  grep -qE "DEATH hero=" $log.txt || { echo "FAIL: no death"; fail=1; }
  # the file written while the hero is dead holds a corpse
  grep -qE "D2S EXPORT reparse: .*died=true corpse=[1-9]" $log.txt || { echo "FAIL: the export at the time of death has no corpse items"; fail=1; }
  grep -qE "DEATH respawn .* equipment=\[\] " $log.txt || { echo "FAIL: the respawned hero still has its equipment"; fail=1; }
  grep -qE "DEATH corpse looted items=[1-9][0-9]* all taken" $log.txt || { echo "FAIL: the corpse items were not looted back"; fail=1; }
}
