scenario_name="Act 4 population: River of Flame (level 107) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 107; }
scenario_check() { poplevel_check 107 auto; }
