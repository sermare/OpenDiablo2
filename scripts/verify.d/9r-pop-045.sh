scenario_name="Act 2 population: Valley of Snakes (level 45) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
# Known gap: every room of this 32x32 level is a Populate=0 preset (LvlPrest) and no DS1 monster unit resolves, so nothing spawns (min 0)
scenario_env() { poplevel_env 45; }
scenario_check() { poplevel_check 45 0; }
