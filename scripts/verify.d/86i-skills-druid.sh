scenario_name="Druid skills (every Druid skill without an earlier scenario: cast by a fresh Druid, each with an effect)"
scenario_turbo=1
# w27 skill coverage (docs/skills-coverage.md, scripts/verify.d/lib/skillcov.sh): a fresh Druid (OD2_AUTOCAST_CLASS)
# casts the Druid skills that no other scenario casts, at zombies that are spawned again when all are dead, and every
# skill must show its outcome in the log (damage, state, summon, missile spawned or hit) besides the finished cast.
source scripts/verify.d/lib/skillcov.sh
scenario_env() {
  local list="Eruption,2!!;Molten Boulder,1!!@8~moltenboulder;Twister,1!!@8;Tornado,1!!@8;Shock Wave,1!!@7;Arctic Blast,1;Cyclone Armor,1;Vines,1;Cycle of Life,1;Heart of Wolverine,1;Spirit of Barbs,1;Summon Spirit Wolf,1;Summon Fenris,1;Summon Grizzly,1;Wearwolf,1;Feral Rage,1!!;Fire Claws,1!!;Fury,2!!;Wearbear,1;Maul,2!!;Volcano,1;Firestorm,1;Armageddon,1"
  echo "export OD2_AUTOCAST_CLASS=druid OD2_AUTOCAST_MANA=900 OD2_AUTOCAST_LEVEL=10 OD2_AUTOMONSTER=\"zombie1,5\" OD2_AUTOMONSTER_SECONDS=300"
  echo "export OD2_AUTOCAST=\"$list\""
}
scenario_check() {
  grep -E "AUTOCAST summary" $log.txt | cut -c1-210 | head -3
  local skills=("Eruption" "Molten Boulder" "Twister" "Tornado" "Shock Wave" "Arctic Blast" "Cyclone Armor" "Vines" "Cycle of Life" "Heart of Wolverine" "Spirit of Barbs" "Summon Spirit Wolf" "Summon Fenris" "Summon Grizzly" "Wearwolf" "Feral Rage" "Fire Claws" "Fury" "Wearbear" "Maul" "Volcano" "Firestorm" "Armageddon")
  skillcov_report "${skills[@]}"
  skillcov_cast "${skills[@]}"
  skillcov_effect missile "Eruption" 'erruption.center'
  skillcov_effect dmg "Eruption"
  # w29: real hit/damage/state effects (the harness recasts "!!" until the skill's OWN missile/area landed)
  skillcov_effect missile "Molten Boulder" 'moltenboulder'
  skillcov_effect hit "Molten Boulder" 'moltenboulder'
  skillcov_effect dmg "Molten Boulder"
  skillcov_effect missile "Twister" 'twister'
  skillcov_effect hit "Twister" 'twister'
  skillcov_effect dmg "Twister"
  skillcov_effect missile "Tornado" 'tornado'
  skillcov_effect raw "Tornado" 'DAMAGE skill="Tornado( area)?" .*dmg=[1-9]'
  skillcov_effect missile "Shock Wave" 'shockwave'
  skillcov_effect hit "Shock Wave" 'shockwave'
  skillcov_effect dmg "Shock Wave"
  skillcov_effect raw "Shock Wave" 'MISSILE hit name=shockwave .*stun:[1-9]'
  skillcov_effect chan "Arctic Blast"
  skillcov_effect state "Cyclone Armor"
  skillcov_effect summon "Vines"
  skillcov_effect raw "Vines" 'TRAP armed skill="Vines"'
  skillcov_effect summon "Cycle of Life"
  skillcov_effect raw "Cycle of Life" 'TRAP armed skill="Cycle of Life"'
  skillcov_effect summon "Heart of Wolverine"
  skillcov_effect state "Heart of Wolverine"
  skillcov_effect summon "Spirit of Barbs"
  skillcov_effect state "Spirit of Barbs"
  skillcov_effect summon "Summon Spirit Wolf"
  skillcov_effect summon "Summon Fenris"
  skillcov_effect summon "Summon Grizzly"
  skillcov_effect state "Wearwolf"
  skillcov_effect state "Feral Rage"
  skillcov_effect melee "Feral Rage"
  skillcov_effect dmg "Fire Claws"
  skillcov_effect melee "Fire Claws"
  skillcov_effect melee "Fury"
  skillcov_effect dmg "Fury"
  # the bear form was entered (the cast used to revert it: Feral Rage / Maul of earlier skills share its state group)
  skillcov_effect raw "Wearbear" 'STATE apply skill="Wearbear" unit=[^ ]+ state=bear '
  ! grep -qE 'STATE revert skill="Wearbear"' $log.txt || { echo "FAIL: Wearbear reverted instead of entering the bear form"; fail=1; }
  skillcov_effect melee "Maul"
  skillcov_effect dmg "Maul"
  skillcov_effect missile "Volcano" 'volcano.*'
  skillcov_effect missile "Firestorm" 'firestorm.*'
  skillcov_effect state "Armageddon"
  skillcov_effect missile "Armageddon" 'armageddonfire'
  ! grep -qE "panic|fatal error|SIGSEGV" $log.txt || { echo "FAIL: crash in the log"; fail=1; }
}
