scenario_name="Act 1 population: Burial Grounds (level 17) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 17; }
scenario_check() { poplevel_check 17 auto; }
