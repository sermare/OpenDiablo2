scenario_name="Act 1 population: Tristram (level 38): monsters are legal for its Levels.txt row, the density rolls are the real game's logic regions"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 38; }
scenario_check() { poplevel_check 38 auto 5460 36; }
