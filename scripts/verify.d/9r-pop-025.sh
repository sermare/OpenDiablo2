scenario_name="Act 1 population: Tower Cellar Level 5 (level 25): monsters are legal for its Levels.txt row, the density rolls are the real game's logic regions"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 25; }
scenario_check() { poplevel_check 25 10; }
