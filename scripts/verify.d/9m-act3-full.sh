scenario_name="Act 3 complete (Kurast Docks -> Travincal -> the Council -> the Orb -> Durance 1..3 -> Mephisto -> the red portal -> Pandemonium Fortress)"
a3=$tmp/act3m
# The main line of Act 3 played by OD2_AUTOSCRIPT (docs/PLAYTEST.md, "Act 3 in depth"): the revived level 94 hero
# sails to Kurast Docks, walks the seamless borders to Travincal, kills the Council (super uniques with packs;
# Khalim's Flail drops), cubes Khalim's Will (the three organs come from the chests that 9n plays; here
# they are given), smashes the Compelling Orb, takes the stairs through Durance 1 and 2, kills Mephisto in his
# lair and leaves through the red portal for the Pandemonium Fortress.
# One game window, OD2_AUTOSPEED=8. "KILL giving up" warnings are part of fights in the undergrowth.
scenario_warnings_ok=1
scenario_timeout=1200 # a long route: the borders of the whole act, the council, three Durance levels and Mephisto
scenario_env() {
  mkdir -p $a3/s94 $a3/wb94; rm -f $a3/s94/*.d2s(N) $a3/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $a3/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    local s="wait:1;say:resetquests;say:completequest 1 6;travel:2;expect:level=40;wait:3;say:completequest 2 6;travel:3;expect:level=75;wait:3"
    # room in the inventory for the Flail and the Will
    local i
    for i in 1 2 3 4 5 6 7 8 9; do s+=";say:dropinv cm3;say:dropinv cm1"; done
    s+=";move:npc=Ormus;until:NPC menu opened: npc=\"Ormus\",60;menu:Talk;wait:3"
    hop() { s+=";walkto:exit=$1;walkto:exit=$1;expect:level=$1;wait:1;say:capframe $tmp/act3m-$1.png"; }
    hop 76; hop 77; hop 78; hop 79; hop 80; hop 81; hop 82; hop 83
    # the Council (three super uniques with their packs) stands around the stairs of the Durance
    s+=";walkto:object=404;kill:near=18,200;kill:near=1,1;say:pickground qf1;wait:2;loot:1,5;say:capframe $tmp/act3m-council.png"
    s+=";say:giveitem qey;say:giveitem qhr;say:giveitem qbr"
    s+=";say:cubeput qf1 qey qhr qbr;say:transmute"
    s+=";walkto:object=404;wait:2;say:capframe $tmp/act3m-orb.png"
    s+=";walkto:exit=100;expect:level=100;wait:2;walkto:exit=101;expect:level=101;wait:2;walkto:exit=102;expect:level=102;wait:2"
    s+=";kill:all,300;kill:all,300;kill:all,300;kill:near=1,1;wait:14;say:capframe $tmp/act3m-lair.png"
    s+=";walkto:object=342;expect:level=103;wait:3;say:capframe $tmp/act3m-fortress.png"
    echo "export OD2_AUTOGAME=\"$a3/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$a3/wb94\" OD2_REALMAPS=1 OD2_AUTOSPEED=8 OD2_AUTOMONSTER_DIFF=0 OD2_POPULATE=0"
    echo "export OD2_AUTOSCRIPT='$s;exit'"
  else
    echo "# no revived sample (D2_TABLES unset or the tool failed)" >&2
    echo "export OD2_AUTOSCRIPT='wait:1;exit'"
  fi
}
scenario_check() {
  grep -E "ACT3 |LEVEL CHANGE|POPULATE level|QUEST A3Q|AUTOSCRIPT RESULT" $log.txt | cut -c1-230 | tail -50
  [ -s $a3/s94/Hero.d2s ] || { echo "SKIP: no revived sample hero in this run"; return; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Act 3 full playthrough did not pass"; fail=1; }
  grep -E "AUTOSCRIPT step [0-9]+ FAIL" $log.txt | cut -c1-200
  for route in "from=75 to=76 .*via=edge" "from=82 to=83 .*via=edge" "from=83 to=100 .*via=warp" "from=100 to=101 .*via=warp" \
               "from=101 to=102 .*via=warp" "from=102 to=103 .*via=act:portal"; do
    grep -qE "LEVEL CHANGE $route" $log.txt || { echo "FAIL: missing level change $route"; fail=1; }
  done
  # the Council: three named super uniques, followers, Flail, the quest counts the members
  for n in "Ismail Vilehand" "Geleb Flamefinger" "Toorc Icefist"; do
    grep -q "adopted DS1 super unique \"$n\"" $log.txt || { echo "FAIL: $n was not built as a super unique"; fail=1; }
    grep -q "MONSTER death name=$n " $log.txt || { echo "FAIL: $n was not killed"; fail=1; }
  done
  grep -q "ACT3 .* drops Khalim's Flail" $log.txt || { echo "FAIL: no Khalim's Flail"; fail=1; }
  grep -q "ACT3 Compelling Orb placed" $log.txt || { echo "FAIL: no Compelling Orb"; fail=1; }
  grep -q "CUBE .*qf2\|ITEMGEN.*qf2\|Khalim's Will" $log.txt || { echo "FAIL: the cube made no Khalim's Will"; fail=1; }
  grep -q "ACT3 the Compelling Orb is smashed" $log.txt || { echo "FAIL: the orb was not smashed"; fail=1; }
  grep -qE "QUEST A3Q2 slot=18 bits .*Compelling Orb" $log.txt || { echo "FAIL: Khalim's Will quest did not see the orb"; fail=1; }
  grep -q "warp refused level 83 -> 100" $log.txt && echo "(the Durance stairs were refused at least once before the orb was smashed)"
  # Mephisto and the red portal
  grep -q "MONSTER death name=Mephisto " $log.txt || { echo "FAIL: Mephisto was not killed"; fail=1; }
  grep -qE "QUEST A3Q6 slot=22 bits .*exe boss kill bit" $log.txt || { echo "FAIL: The Guardian quest did not see Mephisto's death"; fail=1; }
  grep -q "ACT3 the red portal to the Pandemonium Fortress opens" $log.txt || { echo "FAIL: no red portal"; fail=1; }
  grep -q "QUEST DROP mss from \"Mephisto\"" $log.txt || { echo "FAIL: no Mephisto's Soulstone"; fail=1; }
  grep -q "DEATH hero=" $log.txt && { echo "FAIL: the level 94 hero died"; fail=1; }
  last=$(grep "D2S EXPORT reparse" $log.txt | tail -1)
  echo "$last" | grep -q " act=4 " || { echo "FAIL: the exported .d2s of a hero in the Fortress is not an Act 4 save: $last" | cut -c1-200; fail=1; }
  for s in 76 83 council orb lair fortress; do [ -s $tmp/act3m-$s.png ] || { echo "FAIL: no screenshot act3m-$s.png"; fail=1; }; done
  if grep -E "\[ERROR\]|EXIT no way found" $log.txt | grep -v "skipping missing"; then echo "FAIL: errors or a walk that gave up"; fail=1; fi
  if grep -E "Unknown tile|panic|FATAL" $log.txt; then echo "FAIL: unknown tiles, panic or fatal error"; fail=1; fi
}
