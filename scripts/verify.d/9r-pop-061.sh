scenario_name="Act 2 population: Claw Viper Temple Level 2 (level 61) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 61; }
scenario_check() { poplevel_check 61 16; }
