scenario_name="Act 3 population: Kurast Causeway (level 82) monsters are legal for its Levels.txt row"
source scripts/verify.d/lib/act3pop.sh
scenario_env() { act3pop_env 82; }
scenario_check() { act3pop_check 82 1; }
