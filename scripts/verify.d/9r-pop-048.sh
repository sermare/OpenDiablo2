scenario_name="Act 2 population: Sewers Level 2 (level 48) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 48; }
scenario_check() { poplevel_check 48 auto; }
