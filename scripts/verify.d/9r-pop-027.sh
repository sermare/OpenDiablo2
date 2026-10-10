scenario_name="Act 1 population: Outer Cloister (level 27): monsters are legal for its Levels.txt row, the density rolls are the real game's logic regions"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 27; }
scenario_check() { poplevel_check 27 auto 5842 35; }
