scenario_name="Act 5 population: Halls of Vaught (level 124) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 124; }
scenario_check() { poplevel_check 124 auto; }
