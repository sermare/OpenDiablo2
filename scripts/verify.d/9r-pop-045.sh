scenario_name="Act 2 population: Valley of Snakes (level 45) spawns no natural monsters (all 16 rooms are Populate=0 presets, as in the original) and its tiles build"
source scripts/verify.d/lib/poplevel.sh
# Not a gap: the real game flags every room of this level 0x800000 (DRLG_CreatePresetRoom: LvlPrest Populate = 0, emulator: 16 of 16), so
# MONREGION_ShouldSpawnForRoom refuses them; the only units are the two Viper1.ds1 towers, each kept with chance 1/3. Rolls 0, rooms 16.
scenario_env() { poplevel_env 45; }
scenario_check() {
  poplevel_check 45 0 0 16
  grep -q "TILESTATS level=45 exact=true" $log.txt || { echo "FAIL: level 45 tiles are not the exact records (game error 0x2b1?)"; fail=1; }
}
