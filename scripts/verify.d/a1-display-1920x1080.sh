scenario_name="display 1920x1080 (screenshot size, bottom bar centred, picking at the far left / right of the world, hero walks there)"
scenario_realtime=1
scenario_timeout=240
source scripts/verify.d/lib/display.sh
scenario_env() { display_env 1920 1080; }
scenario_check() { display_check 1920 1080; }
