scenario_name="Act 2 population: Far Oasis (level 43) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 43; }
scenario_check() { poplevel_check 43 auto; }
