scenario_name="Act 5 population: Rigid Highlands (level 111) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 111; }
scenario_check() { poplevel_check 111 auto; }
