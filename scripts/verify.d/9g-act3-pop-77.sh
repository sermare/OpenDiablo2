scenario_name="Act 3 population: Great Marsh (level 77) monsters are legal for its Levels.txt row"
source scripts/verify.d/lib/act3pop.sh
scenario_env() { act3pop_env 77; }
scenario_check() { act3pop_check 77 8; }
