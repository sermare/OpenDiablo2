scenario_name="act towns and act travel (towns of acts 1-5: NPC menus, travel 1->2->3->4->5 with the quest rules, and back)"
# The travel rules (d2level/acttravel.go) are checked with the real quest record of the hero: trips are refused
# until the quest flags are forced with `say:completequest <act> <quest>`; the trips back from act 4 and 5 have no NPC in
# the original and need `say:travelfree 1`. Menu rows are logged as "NPC menu opened: ... class=<id> ... rows=[...]".
scenario_env() {
  # the runner shares one save between scenarios and the act playthroughs leave the hero in act 2+ (the act is
  # persisted in the .d2s), so start from a pristine copy: this scenario must begin in the Rogue Encampment
  [ -n "${D2S_SAMPLE_BODY:-}" ] && cp "$D2S_SAMPLE_BODY" "$save"
  local s="wait:1;say:resetquests"
  # act 1: Warriv offers Go East only after Andariel (A1Q6)
  s+=";refuse:2;move:npc=Warriv;wait:10;expect:log=class=155 known=true rows=[talk cancel]"
  s+=";say:completequest 1 6;move:npc=Akara;wait:10;move:npc=Warriv;wait:10;expect:log=class=155 known=true rows=[talk go east cancel]"
  s+=";travel:2;expect:level=40;wait:2"
  # act 2: Lut Gholein
  s+=";refuse:3;refuse:5;move:npc=Warriv;wait:6;expect:log=class=175 known=true rows=[talk go west cancel]"
  s+=";move:npc=Fara;wait:12;expect:log=class=178 known=true rows=[talk trade/repair cancel]"
  s+=";say:completequest 2 6;move:npc=Meshif;wait:14;expect:log=class=210 known=true rows=[talk sail east cancel]"
  s+=";travel:3;expect:level=75;wait:2"
  # act 3: Kurast Docks
  s+=";refuse:4;move:npc=Meshif;wait:6;expect:log=class=264 known=true rows=[talk sail west cancel]"
  s+=";move:npc=Hratli;wait:12;expect:log=class=253 known=true rows=[talk trade/repair cancel]"
  s+=";say:completequest 3 6;travel:4;expect:level=103;wait:2"
  # act 4: Pandemonium Fortress
  s+=";refuse:5;move:npc=Tyrael;wait:10;expect:log=class=367 known=true rows=[talk hire cancel]"
  s+=";move:npc=Halbu;wait:12;expect:log=class=257 known=true rows=[trade/repair cancel]"
  s+=";say:completequest 4 2;travel:5;expect:level=109;wait:2"
  # act 5: Harrogath
  s+=";move:npc=Larzuk;wait:12;expect:log=class=511 known=true rows=[talk trade/repair cancel]"
  s+=";move:npc=Malah;wait:8;expect:log=class=513 known=true rows=[talk trade cancel]"
  # the way back: no NPC exists for 5->4 and 4->3 (debug mode), 3->2 and 2->1 are Meshif and Warriv
  s+=";refuse:4;say:travelfree 1;travel:4;expect:level=103;wait:1;travel:3;expect:level=75;wait:1;say:travelfree 0"
  s+=";travel:2;expect:level=40;wait:1;travel:1;expect:level=1;wait:1;exit"
  echo "export OD2_AUTOSCRIPT=\"$s\""
}
scenario_check() {
  grep -E "AUTOSCRIPT RESULT|AUTOSCRIPT .*FAIL|TRAVEL |ACT (CHANGE|arrival|saved|finished)|TOWN level=|NPC menu opened" $log.txt | cut -c1-230
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: act travel scenario did not pass"; fail=1; }
  for l in 40 75 103 109; do
    grep -q "TOWN level=$l npcs=" $log.txt || { echo "FAIL: town $l was not built"; fail=1; }
    grep -q "ACT arrival level=$l .*waypoints=[1-9].* stash=[1-9]" $log.txt || { echo "FAIL: town $l has no waypoint/stash object"; fail=1; }
  done
  for a in 2 3 4 5 1; do
    grep -q "ACT saved player=.* act=$a " $log.txt || { echo "FAIL: act $a was not saved for the hero"; fail=1; }
  done
  grep -q "TRAVEL refused act 1 -> 2" $log.txt || { echo "FAIL: act 1 -> 2 was not refused before Andariel"; fail=1; }
  grep -q "ACT finished player=.* act=1 quest slot=7" $log.txt || { echo "FAIL: act 1 was not marked finished"; fail=1; }
}
