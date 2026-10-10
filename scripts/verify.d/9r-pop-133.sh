scenario_name="Act 5 population: Pandemonium Run 1 (level 133) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 133; }
scenario_check() { poplevel_check 133 auto; }
