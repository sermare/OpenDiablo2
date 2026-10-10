scenario_name="Act 3 population: Upper Kurast (level 81) monsters are legal for its Levels.txt row"
source scripts/verify.d/lib/act3pop.sh
scenario_env() { act3pop_env 81; }
scenario_check() { act3pop_check 81 4; }
