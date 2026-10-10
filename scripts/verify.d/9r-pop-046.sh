scenario_name="Act 2 population: Canyon of the Magi (level 46) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 46; }
scenario_check() { poplevel_check 46 auto; }
