scenario_name="display 3440x1440 (screenshot size, bottom bar centred, picking at the far left / right of the world, hero walks there)"
scenario_realtime=1
scenario_timeout=240
source scripts/verify.d/lib/display.sh
scenario_env() { display_env 3440 1440; }
scenario_check() { display_check 3440 1440; }
