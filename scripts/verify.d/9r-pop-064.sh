scenario_name="Act 2 population: Maggot Lair Level 3 (level 64) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 64; }
scenario_check() { poplevel_check 64 17; }
