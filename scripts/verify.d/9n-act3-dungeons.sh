scenario_name="Act 3 dungeons and quest objects (Swampy Pit 2 + 3, Flayer Dungeon 2 + 3, Sewers lever, Khalim's Eye / Brain / Heart chests, Lam Esen's Tome)"
a3=$tmp/act3n
# The levels 9g and 9m do not walk, entered by town portals (one at a time): the second and third levels of the
# Swampy Pit and the Flayer Dungeon by their stairs, and the quest objects of Act 3 operated by walking to them: the Eye in
# Spider Cavern, the Brain in Flayer Dungeon 3, the lever of Sewers 1 and the Heart in
# Sewers 2, Lam Esen's Tome in the Ruined Temple.
scenario_warnings_ok=1
scenario_env() {
  mkdir -p $a3/s94 $a3/wb94; rm -f $a3/s94/*.d2s(N) $a3/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $a3/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    local s="wait:1;say:resetquests;say:completequest 1 6;travel:2;expect:level=40;wait:3;say:completequest 2 6;travel:3;expect:level=75;wait:7"
    local i
    for i in 1 2 3 4 5 6; do s+=";say:dropinv cm3"; done
    for i in 1 2 3 4 5 6; do s+=";say:dropinv cm1"; done
    # into <level>: a town portal there; home: a town portal back
    into() { s+=";say:spawnportal $1;use:Portal;expect:level=$1;wait:3;say:capframe $tmp/act3n-$1.png"; }
    home() { s+=";wait:5;say:spawnportal 75;use:Portal;expect:level=75;wait:7"; }
    hop() { s+=";walkto:exit=$1;expect:level=$1;wait:2;say:capframe $tmp/act3n-$1.png"; }
    into 85; s+=";walkto:object=407;wait:3;loot:10,15"; home
    into 86; hop 87; hop 90; home
    into 88; hop 89; hop 91; s+=";kill:near=30,80;kill:near=1,1;walkto:object=406;wait:3;loot:10,15"; home
    into 92; s+=";walkto:object=367;wait:3;walkto:exit=93;expect:level=93;wait:3;walkto:object=405;wait:3;loot:10,15"; home
    into 94; s+=";walkto:object=193;wait:3;loot:10,15"; home
    echo "export OD2_AUTOGAME=\"$a3/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$a3/wb94\" OD2_REALMAPS=1 OD2_AUTOSPEED=8 OD2_AUTOMONSTER_DIFF=0"
    echo "export OD2_AUTOSCRIPT='$s;exit'"
  else
    echo "# no revived sample (D2_TABLES unset or the tool failed)" >&2
    echo "export OD2_AUTOSCRIPT='wait:1;exit'"
  fi
}
scenario_check() {
  grep -E "ACT3 |LEVEL CHANGE|warp refused|AUTOSCRIPT RESULT" $log.txt | grep -v "ACT3 level 75" | cut -c1-200 | tail -40
  [ -s $a3/s94/Hero.d2s ] || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Act 3 dungeon walk did not pass"; fail=1; }
  grep -E "AUTOSCRIPT step [0-9]+ FAIL" $log.txt | cut -c1-200
  for route in "from=86 to=87 .*via=warp" "from=87 to=90 .*via=warp" "from=88 to=89 .*via=warp" "from=89 to=91 .*via=warp"; do
    grep -qE "LEVEL CHANGE $route" $log.txt || { echo "FAIL: missing level change $route"; fail=1; }
  done
  for lvl in 87 89; do
    grep -E "POPULATE level $lvl " $log.txt | grep -qE "[1-9][0-9]* monsters" || { echo "FAIL: level $lvl got no monsters"; fail=1; }
  done
  # the quest objects of Act 3 gave their items, and the quest system saw them picked up
  for obj in "Khalim's Eye chest" "Khalim's Brain chest" "Khalim's Heart chest" "Lam Esen's Tome" "Sewer lever"; do
    grep -q "ACT3 $obj operated" $log.txt || { echo "FAIL: $obj was not operated"; fail=1; }
  done
  grep -q "LEVEL CHANGE from=92 to=93 .*via=warp" $log.txt || { echo "FAIL: the stairs to Sewers 2 do not work after the lever"; fail=1; }
  if grep -E "\[ERROR\]|EXIT gave up|EXIT no way found" $log.txt | grep -v "skipping missing"; then echo "FAIL: errors or a walk that gave up"; fail=1; fi
  if grep -E "Unknown tile|panic|FATAL" $log.txt; then echo "FAIL: unknown tiles, panic or fatal error"; fail=1; fi
}
