scenario_name="Act 5 population: Pandemonium Run 3 (level 135) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 135; }
scenario_check() { poplevel_check 135 24; }
