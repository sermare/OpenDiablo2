scenario_name="display 1512x982 (screenshot size, bottom bar centred, picking at the far left / right of the world, hero walks there)"
scenario_realtime=1
scenario_timeout=240
source scripts/verify.d/lib/display.sh
scenario_env() { display_env 1512 982; }
scenario_check() { display_check 1512 982; }
