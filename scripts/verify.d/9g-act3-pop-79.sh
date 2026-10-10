scenario_name="Act 3 population: Lower Kurast (level 79) monsters are legal for its Levels.txt row"
source scripts/verify.d/lib/act3pop.sh
scenario_env() { act3pop_env 79; }
scenario_check() { act3pop_check 79 4; }
