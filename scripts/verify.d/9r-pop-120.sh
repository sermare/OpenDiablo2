scenario_name="Act 5 population: Rocky Summit (level 120) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
# MonDen 0: the level has no natural monsters (its monsters come from DS1 presets); the check expects none
scenario_env() { poplevel_env 120; }
scenario_check() { poplevel_check 120 0; }
