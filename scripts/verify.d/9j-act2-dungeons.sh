scenario_name="Act 2 town dungeons and the Canyon of the Magi (Sewers, Palace, Arcane Sanctuary pads, the seven tomb entrances)"
a2=$tmp/act2dun
# The town side of Act 2 and the places with no seamless neighbour:
#   Lut Gholein -> Sewers 1-3 (47-49) and back, Lut Gholein -> Harem 1 (50) -> Harem 2 -> Palace Cellar 1-3 (51-54),
#   a portal into the Arcane Sanctuary (74) where a teleport pad moves the hero, the Canyon of the Magi (46, entered
#   through a portal: it has no seamless neighbour) with its seven tomb entrances 66-72, each walked into and back.
scenario_warnings_ok=1
scenario_env() {
  mkdir -p $a2/s94 $a2/wb94; rm -f $a2/s94/*.d2s $a2/wb94/*.d2s
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $a2/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    local s="wait:1;say:resetquests;say:completequest 1 6;travel:2;expect:level=40"
    s+=";walkto:exit=47;expect:level=47;wait:2;walkto:exit=48;expect:level=48;walkto:exit=49;expect:level=49"
    s+=";say:spawnportal 40;wait:6;use:Portal;expect:level=40"
    s+=";walkto:exit=50;expect:level=50;wait:2;say:capframe $tmp/act2-harem.png;walkto:exit=51;expect:level=51;walkto:exit=52;expect:level=52"
    s+=";say:spawnportal 54;wait:6;use:Portal;expect:level=54;wait:2;say:capframe $tmp/act2-cellar.png"
    s+=";say:spawnportal 74;wait:6;use:Portal;expect:level=74;wait:3;say:capframe $tmp/act2-arcane.png"
    s+=";walkto:object=192;wait:3"
    s+=";say:spawnportal 73;wait:6;use:Portal;expect:level=73;wait:3;say:capframe $tmp/act2-duriel.png;say:spawnportal 40;wait:6;use:Portal;expect:level=40;say:spawnportal 46;wait:6;use:Portal;expect:level=46;wait:3;say:capframe $tmp/act2-canyon.png"
    local t
    for t in 66 67 68 69 70 71 72; do s+=";walkto:exit=$t;expect:level=$t;wait:2;walkto:exit=46;expect:level=46"; done
    s+=";exit"
    echo "export OD2_AUTOGAME=\"$a2/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$a2/wb94\" OD2_REALMAPS=1 OD2_AUTOSPEED=3 OD2_AUTOMONSTER_DIFF=0"
    echo "export OD2_AUTOSCRIPT='$s'"
  else
    echo "# no revived sample (D2_TABLES unset or the tool failed)" >&2
    echo "export OD2_AUTOSCRIPT='wait:1;exit'"
  fi
}
scenario_check() {
  grep -E "LEVEL CHANGE|AUTOSCRIPT step [0-9]+ FAIL|AUTOSCRIPT RESULT|Tal Rasha's tomb|teleport pad" $log.txt | cut -c1-200 | tail -60
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the Act 2 dungeon walk did not pass"; fail=1; }
  [ -s $a2/s94/Hero.d2s ] || { echo "SKIP: no revived sample hero in this run"; return; }
  for route in "from=40 to=47 .*via=warp" "from=47 to=48" "from=48 to=49" "from=40 to=50 .*via=warp" "from=50 to=51" "from=51 to=52" "to=54 " "to=74 " "to=73 " "to=46 "; do
    grep -qE "LEVEL CHANGE $route" $log.txt || { echo "FAIL: missing level change $route"; fail=1; }
  done
  # all seven tomb entrances lead to a different tomb and back
  for t in 66 67 68 69 70 71 72; do
    grep -qE "LEVEL CHANGE from=46 to=$t " $log.txt && grep -qE "LEVEL CHANGE from=$t to=46 " $log.txt || { echo "FAIL: tomb $t not entered and left"; fail=1; }
  done
  # exactly one tomb is the real one and only it has the orifice
  [ "$(grep -c "Tal Rasha's tomb level [0-9]*: real=true .* orifice objects=1" $log.txt)" -ge 1 ] || { echo "FAIL: no real tomb with an orifice"; fail=1; }
  grep -E "Tal Rasha's tomb level [0-9]*: real=false .* orifice objects=[1-9]" $log.txt && { echo "FAIL: a decoy tomb has an orifice"; fail=1; }
  grep -qE "OBJECT teleport pad id=[0-9]+ .* hero lands at" $log.txt || { echo "FAIL: the teleport pad did nothing"; fail=1; }
}
