scenario_name="Act 5 population: Hell2 (level 126) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 126; }
scenario_check() { poplevel_check 126 auto; }
