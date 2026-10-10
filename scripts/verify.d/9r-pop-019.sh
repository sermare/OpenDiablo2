scenario_name="Act 1 population: Mausoleum (level 19) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 19; }
scenario_check() { poplevel_check 19 auto; }
