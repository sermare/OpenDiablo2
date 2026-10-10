scenario_name="Act 5 population: Hell1 (level 125) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 125; }
scenario_check() { poplevel_check 125 31; }
