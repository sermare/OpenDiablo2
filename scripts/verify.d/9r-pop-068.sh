scenario_name="Act 2 population: Tal Rasha's Tomb (level 68) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 68; }
scenario_check() { poplevel_check 68 auto; }
