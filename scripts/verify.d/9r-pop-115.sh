scenario_name="Act 5 population: Crystalized Cavern Level 2 (level 115) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 115; }
scenario_check() { poplevel_check 115 83; }
