scenario_name="drops (hero kills monsters on a real DRLG map; every dropped item is made by the item creator and logged with its rolled stats)"
scenario_env() {
  local s='wait:2' m
  # ordinary monsters, then bosses (their treasure classes drop the rich items)
  for m in fallen1 fallen1 fallen1 fallen1 fallen1 fallen1 zombie1 skeleton1 andariel duriel mephisto diablo andariel duriel mephisto diablo; do
    s+=";say:spawnmon $m;wait:1;say:killnear;wait:1"
  done
  echo 'export OD2_REALMAPS=1 OD2_AUTOLEVEL=9'
  echo "export OD2_AUTOSCRIPT='$s;exit'"
}
scenario_check() {
  grep -E "MONSTER drop|ITEMGEN created" $log.txt | cut -c1-330 | head -30
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
