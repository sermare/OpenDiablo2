scenario_name="Act 1 population: Cathedral (level 33): monsters are legal for its Levels.txt row, the density rolls are the real game's logic regions"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 33; }
scenario_check() { poplevel_check 33 auto 2427 20; }
