scenario_name="Act 5 population: Halls of Death's Calling (level 123) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 123; }
scenario_check() { poplevel_check 123 124; }
