scenario_name="Act 2 population: Tal Rasha's Tomb (level 66) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 66; }
scenario_check() { poplevel_check 66 80; }
