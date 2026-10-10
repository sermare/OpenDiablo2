scenario_name="Act 2 population: Claw Viper Temple Level 1 (level 58) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 58; }
scenario_check() { poplevel_check 58 176; }
