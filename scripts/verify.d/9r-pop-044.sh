scenario_name="Act 2 population: Lost City (level 44) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 44; }
scenario_check() { poplevel_check 44 auto; }
