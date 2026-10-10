scenario_name="Act 1 walk with the level 94 sample hero (revived copy of the hardcore corpse): town, Blood Moor, Den of Evil, back"
a1=$tmp/act1
# The sample character is a dead hardcore Sorceress; scripts/d2s-revive.go makes a living copy (needs D2_TABLES).
# Normal difficulty monsters (OD2_AUTOMONSTER_DIFF=0): the level 94 hero is not built for the Hell monsters of his save.
scenario_env() {
  mkdir -p $a1/s94 $a1/wb94; rm -f $a1/s94/*.d2s(N) $a1/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && make_hero $a1/s94/Hero.d2s; then
    echo "export OD2_AUTOGAME=\"$a1/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$a1/wb94\" OD2_AUTOSPEED=3 OD2_AUTOMONSTER_DIFF=0"
    echo "export OD2_AUTOSCRIPT='wait:1;say:capframe $tmp/act1-94-town.png;move:npc=Akara;until:NPC menu opened: npc=\"Akara\",60;menu:Talk;wait:3;walkto:exit=2;expect:level=2;kill:near=30,45;loot:30,40;walkto:exit=8;expect:level=8;kill:all,200;say:capframe $tmp/act1-94-den.png;walkto:exit=2;expect:level=2;walkto:exit=1;expect:level=1;exit'"
  else
    echo "# no revived sample (D2_TABLES unset or the tool failed)" >&2
    echo "export OD2_AUTOSCRIPT='wait:1;exit'"
  fi
}
scenario_check() {
  grep -E "HERO state|LEVEL CHANGE|POPULATE|KILL (start|done)|LOOT (start|done)|AUTOSCRIPT RESULT" $log.txt | cut -c1-230 | tail -24
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the level 94 walk did not pass"; fail=1; }
  [ -s $a1/s94/Hero.d2s ] || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -E "AUTOSCRIPT step [0-9]+ FAIL" $log.txt | cut -c1-200
  for route in "from=1 to=2 .*via=edge" "from=2 to=8 .*via=warp" "from=8 to=2 .*via=warp" "from=2 to=1 .*via=edge"; do
    grep -qE "LEVEL CHANGE $route" $log.txt || { echo "FAIL: missing level change $route"; fail=1; }
  done
  grep -q "MONSTER death" $log.txt || { echo "FAIL: the level 94 hero killed nothing"; fail=1; }
  grep -q "HERO state at start: level=94" $log.txt || { echo "FAIL: the sample hero was not played at level 94"; fail=1; }
  grep -q "DEATH hero=" $log.txt && { echo "FAIL: the level 94 hero died on the way"; fail=1; }
  [ -s $tmp/act1-94-den.png ] || { echo "FAIL: no screenshot of the Den of Evil"; fail=1; }
  if grep -E "Unknown tile|panic" $log.txt; then echo "FAIL: unknown tiles or panic"; fail=1; fi
}
