scenario_name="Act 5 population: Halls of Anguish (level 122) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 122; }
scenario_check() { poplevel_check 122 108; }
