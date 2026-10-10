scenario_name="Act 5 population: Arreat Plateau (level 112) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 112; }
scenario_check() { poplevel_check 112 auto; }
