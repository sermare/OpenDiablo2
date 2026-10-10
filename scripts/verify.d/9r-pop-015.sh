scenario_name="Act 1 population: Hole Level 2 (level 15) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 15; }
scenario_check() { poplevel_check 15 auto; }
