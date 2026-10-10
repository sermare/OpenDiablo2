scenario_name="Act 5 population: Hell3 (level 127) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 127; }
scenario_check() { poplevel_check 127 auto; }
