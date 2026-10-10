scenario_name="Act 1 population: Den of Evil (level 8) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 8; }
scenario_check() { poplevel_check 8 14; }
