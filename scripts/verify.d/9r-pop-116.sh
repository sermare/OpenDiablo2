scenario_name="Act 5 population: Echo Chamber (level 116) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 116; }
scenario_check() { poplevel_check 116 auto; }
