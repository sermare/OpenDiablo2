scenario_name="Act 2 population: Sewers Level 1 (level 47) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 47; }
scenario_check() { poplevel_check 47 56; }
