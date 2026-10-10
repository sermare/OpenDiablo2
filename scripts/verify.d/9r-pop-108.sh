scenario_name="Act 4 population: Chaos Sanctum (level 108) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 108; }
scenario_check() { poplevel_check 108 40; }
