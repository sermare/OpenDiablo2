scenario_name="Act 4 population: City of the Damned (level 106) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 106; }
scenario_check() { poplevel_check 106 auto; }
