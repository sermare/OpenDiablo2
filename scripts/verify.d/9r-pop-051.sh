scenario_name="Act 2 population: Harem Level 2 (level 51) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 51; }
scenario_check() { poplevel_check 51 78; }
