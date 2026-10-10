scenario_name="Act 1 population: Dark Wood (level 5) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 5; }
scenario_check() { poplevel_check 5 auto; }
