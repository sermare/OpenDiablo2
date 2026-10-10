scenario_name="monster summons (Nest, MinionSpawner and Hydra casts create SUMMON units; per-caster limit holds, no runaway, hydras expire, summons are hostile)"
# Main game: a Foul Crow Nest (skill 91, class from monstats spawn=foulcrow1, AI limit aip3=6). Two sub games below run a
# MinionSpawner (skill 135, spawn=minion1) and a Zakarum High Priest (council member, skill 144 Hydra, lifetime 250 frames).
# The hero is passive for the nest and the spawner (he must not kill the caster) and fights the priest.
# GAP (documented, not asserted): summons do NOT die with their caster. Neither the notes nor the table say they do (a nest laying
# crows or a spawner calling minions leaves them as ordinary monsters; only Hydra has a lifetime), and the Director keeps no
# such rule. Hydra expiry (Dismiss after 250 frames) is asserted instead.
scenario_env() { echo 'export OD2_AUTOMONSTER="crownest1" OD2_AUTOMONSTER_SECONDS=20 OD2_AUTOMONSTER_PASSIVE=1 OD2_AUTOMONSTER_SUMMONS=1'; }

# summon_asserts <label> <log.txt> <class> <max_per_caster_bound>
summon_asserts() {
  local label=$1 lg=$2 class=$3 bound=$4 n mx last ids id hostile=0
  grep -E "SUMMON (caster|refused)" $lg | head -4 | cut -c1-200
  n=$(grep -c "SUMMON caster=.* class=$class " $lg)
  [ "$n" -ge 1 ] || { echo "FAIL: $label: no SUMMON line for class $class"; fail=1; }
  # the live count per caster respects the limit
  mx=$(grep -o "SUMMONCHECK live=[0-9]* max_per_caster=[0-9]*" $lg | sed 's/.*max_per_caster=//' | sort -n | tail -1)
  echo "$label: created=$n max_per_caster=${mx:-none} (bound $bound)"
  [ -n "$mx" ] && [ "$mx" -le "$bound" ] || { echo "FAIL: $label: per-caster live summons ${mx:-none} exceed $bound"; fail=1; }
  # no runaway: the live count of the last samples stays under the bound
  last=$(grep -o "SUMMONCHECK live=[0-9]* max_per_caster=[0-9]*" $lg | tail -3 | sed 's/.*max_per_caster=//' | sort -n | tail -1)
  [ -z "$last" ] || [ "$last" -le "$bound" ] || { echo "FAIL: $label: summons keep growing"; fail=1; }
  grep -q "SUMMONCHECK" $lg || { echo "FAIL: $label: no SUMMONCHECK samples"; fail=1; }
  # hostile: a summoned unit aggroes on, or attacks, the hero
  ids=$(grep "SUMMON caster=.* class=$class " $lg | sed 's/.* id=\([0-9]*\) .*/\1/')
  for id in ${=ids}; do
    grep -qE "MONSTER (aggro|attack) name=.* id=${id}([^0-9]|\$)" $lg && hostile=$((hostile+1))
  done
  echo "$label: hostile summons (aggro or attack lines) $hostile of $n"
  [ "$hostile" -ge 1 ] || { echo "FAIL: $label: no summoned unit became hostile"; fail=1; }
}

scenario_check() {
  local main_log=$log fam ref class bound fight slot id
  grep -E "AUTOMONSTER (start|summary)" $log.txt | cut -c1-200
  summon_asserts nest $log.txt foulcrow1 6   # TUNE: aip3 limit of crownest1 on Normal
  grep -qE "AUTOMONSTER summary spawned=[0-9]+" $log.txt || { echo "FAIL: nest: no summary"; fail=1; }

  for fam in minionspawner1:minion1:25:0 councilmember1:hydra1:30:1; do
    ref=${${(s.:.)fam}[1]}; class=${${(s.:.)fam}[2]}; bound=${${(s.:.)fam}[3]}; fight=${${(s.:.)fam}[4]}
    cp -f "$save" "$tmp/97s-$ref.d2s"; rm -f "$tmp/97s-$ref.d2s.bak"
    sub_cmd=$tmp/97s-$ref.command log=$tmp/97s-$ref.log; rm -f $log
    {
      echo '#!/bin/zsh'
      echo "export OD2_PORT=$OD2_PORT"
      echo "export OD2_AUTOGAME=\"$tmp/97s-$ref.d2s\" OD2_AUTOEXIT=1 OD2_AUTOTEST_MUTE=1 OD2_AUTOSPEED=8"
      echo "export OD2_AUTOMONSTER=$ref OD2_AUTOMONSTER_SECONDS=20 OD2_AUTOMONSTER_SUMMONS=1"
      [ "$fight" = 0 ] && echo "export OD2_AUTOMONSTER_PASSIVE=1"
      echo "$tmp/od2 2>&1 | tee $log"
    } > $sub_cmd
    chmod +x $sub_cmd
    slot=$(./scripts/gameslot.sh acquire $$)
    launch_game $sub_cmd
    wait_run
    ./scripts/gameslot.sh release $slot
    sed 's/\x1b\[[0-9;]*m//g' $log > $log.txt
    summon_asserts $ref $log.txt $class $bound
    grep -E "\[(ERROR|WARNING)\]|panic" $log.txt | grep -v "skipping missing" | head -3 | grep -q . && { echo "FAIL: errors in the $ref log"; fail=1; }
    if [ $class = hydra1 ]; then
      # lifetime: frames=250 in the SUMMON line, and every hydra dies (expiry or kill) before the end
      grep -q "SUMMON caster=.* class=hydra1 .* frames=250" $log.txt || { echo "FAIL: hydra lifetime is not 250 frames"; fail=1; }
      for id in ${=$(grep "SUMMON caster=.* class=hydra1 " $log.txt | head -3 | sed 's/.* id=\([0-9]*\) .*/\1/')}; do
        grep -qE "MONSTER (death|dismiss) name=.* id=${id}([^0-9]|\$)" $log.txt || { echo "FAIL: hydra $id never died"; fail=1; }
      done
    fi
  done
  log=$main_log
}
