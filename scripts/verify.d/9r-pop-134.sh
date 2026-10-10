scenario_name="Act 5 population: Pandemonium Run 2 / Forgotten Sands (level 134): monsters are legal for its Levels.txt row, the density rolls are the real game's logic regions"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 134; }
scenario_check() { poplevel_check 134 auto 6253 89; }
