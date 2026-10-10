scenario_name="Act 2 population: Ancient Tunnels (level 65) monsters are legal for its Levels.txt row and match its density"
source scripts/verify.d/lib/poplevel.sh
scenario_env() { poplevel_env 65; }
scenario_check() { poplevel_check 65 41; }
