scenario_name="Act 5 population: Cellar of Pity (level 114) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 114; }
scenario_check() { poplevel_check 114 auto; }
