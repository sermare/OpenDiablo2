scenario_name="Act 1 population: Inner Cloister (level 32): monsters are legal for its Levels.txt row, the density rolls are the real game's logic regions"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 32; }
scenario_check() { poplevel_check 32 8; }
