scenario_name="drops (hero kills monsters on a real DRLG map; every dropped item is made by the item creator and logged with its rolled stats)"
scenario_env() {
  local s='wait:2' m
  # ordinary monsters, then bosses (their treasure classes drop the rich items)
  for m in fallen1 fallen1 fallen1 fallen1 fallen1 fallen1 zombie1 skeleton1 andariel duriel mephisto diablo andariel duriel mephisto diablo; do
    s+=";say:spawnmon $m;wait:1;say:killnear;wait:1"
  done
  # a champion pack, a unique pack with a minion, a super unique with a minion: the hero kills the leader
  # and the champion / unique / super unique treasure class must drop
  s+=";say:spawnrank champion fallen1;wait:1;say:killleader;wait:1"
  s+=";say:spawnrank unique fallen1;wait:1;say:killleader;wait:1;say:killnear;wait:1"
  s+=";say:spawnrank super Bishibosh;wait:1;say:killleader;wait:1;say:killnear;wait:1"
  echo 'export OD2_AUTOLEVEL=9'
  echo "export OD2_AUTOSCRIPT='$s;exit'"
}
scenario_check() {
  grep -E "MONSTER drop|ITEMGEN created|MONSTER rank" $log.txt | cut -c1-330 | head -60
  # the exe's choice by monster rank (d2drop.MonsterTreasureClass): monstats TreasureClass2 for champions, TreasureClass3
  # for uniques, the SuperUniques.txt TC for super uniques, the normal class for minions
  grep "MONSTER rank kind=champion" $log.txt | grep -q "type_flags=0x5 " || { echo "FAIL: no champion recorded with type flags 0x5"; fail=1; }
  grep "MONSTER rank kind=unique" $log.txt | grep -q "type_flags=0x9 .*mods=\[[0-9]" || { echo "FAIL: no unique leader with flags 0x9 and rolled modifiers"; fail=1; }
  grep "MONSTER rank kind=super" $log.txt | grep -q 'super_unique="Bishibosh"' || { echo "FAIL: the super unique key was not recorded"; fail=1; }
  grep "MONSTER drop" $log.txt | grep -qE 'tc="Act 1 (\([NH]\) )?Champ A".*type_flags=0x5 ' || { echo "FAIL: no drop from the champion class"; fail=1; }
  grep "MONSTER drop" $log.txt | grep -qE 'tc="Act 1 (\([NH]\) )?Unique A".*type_flags=0x9 ' || { echo "FAIL: no drop from the unique class"; fail=1; }
  grep "MONSTER drop" $log.txt | grep -qE 'tc="Act 1 (\([NH]\) )?Super A".*super_unique="Bishibosh"' || { echo "FAIL: no drop from the super unique class"; fail=1; }
  grep -E 'MONSTER drop .*tc="Act 1 (\([NH]\) )?(Champ|Unique|Super) A"' $log.txt | cut -c1-300 | head -6
  kills=$(grep -c "MONSTER death" $log.txt)
  items=$(grep -c "ITEMGEN created source=monster" $log.txt)
  echo "kills=$kills created_items=$items"
  [ "$kills" -ge 8 ] || { echo "FAIL: the hero killed only $kills monsters"; fail=1; }
  [ "$items" -ge 1 ] || { echo "FAIL: no item was created by the monsters' drops"; fail=1; }
  # each created item logs the creator's quality, item level and the rolled stat writes
  if grep "ITEMGEN created" $log.txt | grep -qv "quality=.* ilvl=[0-9]* .* defense=.* durability=.* props=\["; then
    echo "FAIL: an ITEMGEN line lacks the rolled values"; fail=1
  fi
  grep -q "real maze: level 9 .* stamps placed" $log.txt || { echo "FAIL: the DRLG level was not generated"; fail=1; }
}
