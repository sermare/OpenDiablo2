scenario_name="Act 1 population: Moo Moo Farm (level 39) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 39; }
scenario_check() { poplevel_check 39 auto; }
