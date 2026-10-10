scenario_name="Act 4 population: Outer Steppes (level 104) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 104; }
scenario_check() { poplevel_check 104 auto; }
