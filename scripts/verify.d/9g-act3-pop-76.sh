scenario_name="Act 3 population: Spider Forest (level 76) monsters are legal for its Levels.txt row"
source scripts/verify.d/lib/act3pop.sh
scenario_env() { act3pop_env 76; }
scenario_check() { act3pop_check 76 8; }
