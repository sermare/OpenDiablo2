scenario_name="Act 5 population: Throne of Destruction (level 131) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 131; }
scenario_check() { poplevel_check 131 36; }
