scenario_name="Chaos Sanctuary (level 94 sample hero: walk to each of the five seals, the seal bosses, Diablo arrives and is killed, back to the River of Flame)"
cs=$tmp/chaos
# Played by OD2_AUTOSCRIPT (docs/PLAYTEST.md, "Act 4 and 5"). The hero travels to the Pandemonium Fortress (quest flags
# forced), takes a town portal to level 108 and walks to each seal object (objects.txt 392..396), operating it. The boss
# seals (392 Infector of Souls, 394 Lord De Seis, 396 Grand Vizier of Chaos) spawn their bosses with packs at the dummy
# next to the seal; the hero kills them; the last kill with all five seals open summons Diablo at the start dummy (255),
# the hero kills him, and walks out to the River of Flame. d2boss.Seals decides, d2game/d2gamescreen/chaos.go carries it out.
# The sample character is a dead hardcore Sorceress; scripts/d2s-revive.go makes a living copy (needs D2_TABLES).
scenario_warnings_ok=1
scenario_env() {
  mkdir -p $cs/s94 $cs/wb94; rm -f $cs/s94/Hero.d2s $cs/wb94/NokkaSorc.d2s
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $cs/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    local s="wait:1;say:resetquests;say:completequest 1 6;travel:2;expect:level=40;say:completequest 2 6;travel:3;expect:level=75;say:completequest 3 6;travel:4;expect:level=103;wait:11"
    s+=";say:spawnportal 108;use:Portal;expect:level=108;wait:3;say:capframe $tmp/chaos-star.png"
    # east arm: the plain seal 393, then the boss seal 392 (Infector of Souls)
    s+=";walkto:object=393;until:CHAOS seal operated id=393,60;say:capframe $tmp/chaos-seal393.png"
    s+=";walkto:object=392;until:CHAOS spawn seal boss,60;say:capframe $tmp/chaos-infector.png;say:restorevitals;kill:near=30,150"
    # north arm: the boss seal 394 (Lord De Seis); the hero comes back to the star first
    s+=";walkto:object=394;until:CHAOS spawn seal boss,60;say:capframe $tmp/chaos-deseis.png;say:restorevitals;kill:near=30,150"
    # west arm: the plain seal 395, then the boss seal 396 (Grand Vizier of Chaos)
    s+=";walkto:object=395;until:CHAOS seal operated id=395,60"
    s+=";walkto:object=396;until:CHAOS spawn seal boss,60;say:capframe $tmp/chaos-vizier.png;say:restorevitals;kill:near=30,150"
    # all five seals open and the three bosses dead: Diablo arrives and is killed
    s+=";until:class=243 at subtile,30;say:capframe $tmp/chaos-diablo.png;say:restorevitals;kill:near=40,40;say:restorevitals;kill:near=40,40;say:restorevitals;kill:near=40,40;say:restorevitals;kill:near=40,40;say:restorevitals;kill:near=40,40"
    s+=";say:capframe $tmp/chaos-dead.png"
    s+=";walkto:exit=107;expect:level=107;wait:3;exit"
    echo "export OD2_AUTOGAME=\"$cs/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$cs/wb94\" OD2_REALMAPS=1 OD2_AUTOSPEED=8 OD2_AUTOMONSTER_DIFF=0 OD2_POPULATE=1 OD2_AUTOMAP_ASCII=1"
    echo "export OD2_AUTOSCRIPT='$s'"
  else
    echo "# no revived sample (D2_TABLES unset or the tool failed)" >&2
    echo "export OD2_AUTOSCRIPT='wait:1;exit'"
  fi
}
scenario_check() {
  grep -E "LEVEL CHANGE.*to=10[78]|POPULATE level 108|CHAOS|diablo trigger|diablo killed|AUTOSCRIPT (step [0-9]+ FAIL|RESULT)" $log.txt | cut -c1-230 | tail -50
  [ -s $cs/s94/Hero.d2s ] || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Chaos Sanctuary playthrough did not pass"; fail=1; }
  # the level is walkable all over: the monsters of its groups stand on ground the hero can reach (none removed in bulk)
  removed=$(grep -E "POPULATE level 108 " $log.txt | sed -E 's/.*\(([0-9]+) unreachable.*/\1/' | head -1)
  [ -n "$removed" ] && [ "$removed" -le 5 ] || { echo "FAIL: the Chaos Sanctuary removed ${removed:-?} unreachable monsters (want at most 5)"; fail=1; }
  grep -E "POPULATE level 108 " $log.txt | grep -qE "[1-9][0-9]* monsters" || { echo "FAIL: the Chaos Sanctuary got no monsters"; fail=1; }
  # the five seals are objects of the level
  for id in 392 393 394 395 396; do
    grep -q "real outdoor: level 108 object id=$id " $log.txt || { echo "FAIL: seal object $id is not in the Chaos Sanctuary"; fail=1; }
    grep -q "CHAOS seal operated id=$id " $log.txt || { echo "FAIL: seal $id was not operated"; fail=1; }
  done
  grep -q "real outdoor: level 108 object id=255 " $log.txt || { echo "FAIL: the Diablo start dummy (255) is not in the Chaos Sanctuary"; fail=1; }
  # the three seal bosses, then Diablo, then his death and the quest bit
  for boss in "Infector" "De Seis" "Vizier"; do
    grep -E "CHAOS spawn seal boss" $log.txt | grep -q "$boss" || { echo "FAIL: seal boss $boss did not appear"; fail=1; }
  done
  grep -q "diablo trigger: all 5 seals open and 3 seal bosses dead" $log.txt || { echo "FAIL: Diablo was not summoned"; fail=1; }
  grep -q "class=243 at subtile" $log.txt || { echo "FAIL: Diablo did not spawn"; fail=1; }
  grep -q "diablo killed" $log.txt || { echo "FAIL: Diablo was not killed"; fail=1; }
  grep -qE "MONSTER death name=Diablo " $log.txt || { echo "FAIL: no death of Diablo logged"; fail=1; }
  grep -q "DEATH hero=" $log.txt && { echo "FAIL: the level 94 hero died in the Chaos Sanctuary"; fail=1; }
  grep -qE "LEVEL CHANGE from=108 to=107 .*via=warp" $log.txt || { echo "FAIL: the hero did not leave the Chaos Sanctuary for the River of Flame"; fail=1; }
  for s in chaos-star chaos-seal393 chaos-infector chaos-deseis chaos-vizier chaos-diablo chaos-dead; do
    [ -s $tmp/$s.png ] || { echo "FAIL: no screenshot $s.png"; fail=1; }
  done
  if grep -E "Unknown tile|panic" $log.txt; then echo "FAIL: unknown tiles or panic in the Chaos Sanctuary log"; fail=1; fi
}
