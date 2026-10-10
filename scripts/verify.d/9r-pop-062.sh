scenario_name="Act 2 population: Maggot Lair Level 1 (level 62) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 62; }
scenario_check() { poplevel_check 62 auto; }
