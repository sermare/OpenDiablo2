scenario_name="Act 1 population: Tamoe Highland (level 7) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 7; }
scenario_check() { poplevel_check 7 48; }
