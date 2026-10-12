scenario_name="quick pickup (OD2_PICKUP=quick: a click on a ground item with an empty cursor puts it straight into the inventory; a potion goes to the belt)"
scenario_realtime=1
scenario_timeout=240
qp=$tmp/quickpickup
scenario_env() {
  mkdir -p $qp/s94 $qp/wb94; rm -f $qp/s94/*.d2s(N) $qp/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $qp/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$qp/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$qp/wb94\" OD2_AUTOSPEED=3"
  else
    echo "# no revived sample hero, using the default save" >&2
  fi
  echo 'export OD2_REALMAPS=1 OD2_AUTOLEVEL=2 OD2_PICKUP=quick'
  local still='say:herostill;until:HERO still,120'
  local s='expect:level=2;say:heromoved;click:left@hero:200,0;until:HERO moved,120;'"$still"
  s="$s;say:clearinv;say:spawnground tbk 8 8;until:AUTOGROUND spawn source=command name=,30;click:left@item:Tome;until:QUICKPICKUP name=,120;$still;say:spawnground hp1 9 9;until:AUTOGROUND spawn source=command name=,30;click:left@item:Healing;until:QUICKPICKUP name=\"Minor Healing,120;$still;say:fillinv tbk;say:spawnground tbk 8 8;until:AUTOGROUND spawn source=command name=,30;click:left@item:Tome;until:QUICKPICKUP no room for,120;$still;exit"
  echo "export OD2_AUTOSCRIPT='$s'"
}
scenario_check() {
  grep -E "QUICKPICKUP|AUTOGROUND pickup|AUTOSCRIPT RESULT" $log.txt | cut -c1-200 | head
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the quick pickup script did not pass"; fail=1; }
  grep -q "QUICKPICKUP name=" $log.txt || { echo "FAIL: the item did not go straight into the inventory"; fail=1; }
  # w50 (ITEMMODE_ServerPickupGroundItemToInventory 0x561370): no room -> the item stays on the ground and the hero says cantcarry
  grep -q "QUICKPICKUP no room for" $log.txt || { echo "FAIL: a full inventory did not refuse the pickup"; fail=1; }
  grep -qE "HEROVOICE id=0x17 handle=[a-z]+_cant_?carry" $log.txt || { echo "FAIL: no cantcarry voice for the full inventory"; fail=1; }
}
