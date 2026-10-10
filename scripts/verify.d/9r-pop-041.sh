scenario_name="Act 2 population: Rocky Waste (level 41) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 41; }
scenario_check() { poplevel_check 41 41; }
