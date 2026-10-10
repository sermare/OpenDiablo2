scenario_name="Act 2 population: Maggot Lair Level 2 (level 63) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 63; }
scenario_check() { poplevel_check 63 5; }
