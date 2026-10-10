scenario_name="Act 1 population: Crypt (level 18) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 18; }
scenario_check() { poplevel_check 18 auto; }
