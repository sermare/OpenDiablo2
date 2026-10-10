scenario_name="Act 1 population: Monastery Gate (level 26): monsters are legal for its Levels.txt row, the density rolls are the real game's logic regions"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 26; }
scenario_check() { poplevel_check 26 auto; }
