scenario_name="Act 2 population: Dry Hills (level 42) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 42; }
scenario_check() { poplevel_check 42 55; }
