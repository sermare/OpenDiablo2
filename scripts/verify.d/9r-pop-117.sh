scenario_name="Act 5 population: Tundra Wastelands (level 117) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 117; }
scenario_check() { poplevel_check 117 auto; }
