scenario_name="Act 2 population: Sewers Level 3 (level 49) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 49; }
scenario_check() { poplevel_check 49 auto; }
