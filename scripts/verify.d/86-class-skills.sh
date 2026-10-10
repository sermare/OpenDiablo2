scenario_name="class skills (one or more skills of every class: damage, kills, summons, states, auras, traps)"
# A fresh Necromancer (OD2_AUTOCAST_CLASS) is granted and casts a list of skills of all seven
# classes at zombies that are spawned again when they are all dead. Casting order matters: the
# long lasting damage fields (Holy Fire, Hurricane, Thunder Storm) come last, because they kill
# everything the other skills are meant to hit, curses before the kills, corpse skills after kills
# (a corpse is made when none lies around).
scenario_env() {
  local list="Bash,1;War Cry,3;Battle Orders,1"
  list+=";Poison Javelin,2;Multiple Shot,1"
  list+=";Amplify Damage,1;Iron Maiden,1;Poison Dagger,1;Bone Armor,1;Corpse Explosion,1;Raise Skeleton,1"
  list+=";Sacrifice,2;Might,1"
  list+=";Fire Ball,1;Frost Nova,1;Chain Lightning,2;Teleport,1;Energy Shield,1;Blizzard,1"
  list+=";Raven,1;Firestorm,1"
  # feat/skills-npsd: AI curses, Poison Nova, Hydra, Plague Poppy, Telekinesis, Conversion, Hunger
  list+=";Dim Vision,1;Confuse,1;Attract,1;Poison Nova,1;Hydra,1;Plague Poppy,1;Telekinesis,2;Conversion,8;Hunger,2"
  list+=";Fists of Fire,2;Dragon Talon,1;Lightning Sentry,1;Shadow Warrior,1"
  list+=";Thunder Storm,1;Hurricane,1;Holy Fire,1"
  echo "export OD2_AUTOCAST_CLASS=necromancer OD2_AUTOCAST_MANA=500 OD2_AUTOMONSTER=\"zombie1,5\" OD2_AUTOMONSTER_SECONDS=900"
  echo "export OD2_AUTOCAST=\"$list\""
}
scenario_check() {
  grep -E "AUTOCAST (skill_done|summary)|MINION spawn|SUMMON|TRAP armed" $log.txt | cut -c1-210 | head -60
  # every listed skill ran its cast to the end
  for s in "Bash" "War Cry" "Battle Orders" "Poison Javelin" "Multiple Shot" "Amplify Damage" \
           "Iron Maiden" "Poison Dagger" "Bone Armor" "Corpse Explosion" "Raise Skeleton" "Sacrifice" "Might" "Holy Fire" \
           "Fire Ball" "Frost Nova" "Chain Lightning" "Teleport" "Energy Shield" "Blizzard" "Raven" "Firestorm" "Hurricane" \
           "Dim Vision" "Confuse" "Attract" "Poison Nova" "Hydra" "Plague Poppy" "Telekinesis" "Conversion" "Hunger" \
           "Fists of Fire" "Dragon Talon" "Lightning Sentry" "Shadow Warrior" "Thunder Storm"; do
    grep -qE "CAST do skill=\"$s\" .*ok=true" $log.txt || { echo "FAIL: $s was not cast"; fail=1; }
  done
  # Barbarian: melee, buffs
  grep -qE "SKILL melee skill=\"Bash\" .*hit=true" $log.txt || { echo "FAIL: Bash never hit"; fail=1; }
  grep -qE "SKILL area skill=\"War Cry\" " $log.txt || { echo "FAIL: War Cry did not land"; fail=1; }
  grep -qE "STATE apply skill=\"Battle Orders\" .*item_maxhp_percent=" $log.txt || { echo "FAIL: Battle Orders gave no stats"; fail=1; }
  # Amazon: javelin poison over time
  grep -qE "STATE hit skill=\"Poison Javelin\" .*applied=\[poison\]" $log.txt || { echo "FAIL: Poison Javelin did not poison"; fail=1; }
  grep -qE "DAMAGE dot target=.* poison=[1-9]" $log.txt || { echo "FAIL: no poison damage over time"; fail=1; }
  # Necromancer: curses, summons
  grep -qE "STATE apply skill=\"Amplify Damage\" unit=Zombie state=amplifydamage .*damageresist=-100" $log.txt || { echo "FAIL: Amplify Damage"; fail=1; }
  grep -qE "MINION spawn name=Necroskeleton .*kind=minion" $log.txt || { echo "FAIL: no skeleton was raised"; fail=1; }
  grep -qE "MINION attack name=Necroskeleton" $log.txt || { echo "FAIL: the skeleton never attacked"; fail=1; }
  grep -qE "STATE apply skill=\"Bone Armor\" .*bonearmor=" $log.txt || { echo "FAIL: Bone Armor"; fail=1; }
  grep -qE "SKILL area skill=\"Corpse Explosion\" .*targets=" $log.txt || { echo "FAIL: Corpse Explosion"; fail=1; }
  # Paladin: aura states, sacrifice
  grep -qE "STATE aura skill=\"Might\" mode=friendly .*damagepercent=" $log.txt || { echo "FAIL: Might aura"; fail=1; }
  grep -qE "DAMAGE skill=\"Holy Fire\" " $log.txt || { echo "FAIL: Holy Fire never burned anyone"; fail=1; }
  grep -qE "DAMAGE skill=\"Sacrifice\" self" $log.txt || { echo "FAIL: Sacrifice did not hurt the caster"; fail=1; }
  # Sorceress: nova/chain/teleport/shield
  grep -qE "MISSILE hit name=chainlightning" $log.txt || { echo "FAIL: Chain Lightning never hit"; fail=1; }
  grep -qE "MOVE skill=\"Teleport\"" $log.txt || { echo "FAIL: Teleport did not move the hero"; fail=1; }
  grep -qE "STATE apply skill=\"Energy Shield\" .*x_energyshield_pct=" $log.txt || { echo "FAIL: Energy Shield"; fail=1; }
  grep -qE "SKILL strikes skill=\"Blizzard\"" $log.txt || { echo "FAIL: Blizzard"; fail=1; }
  grep -qE "DAMAGE skill=\"Thunder Storm\" .*dmg=[1-9]" $log.txt || { echo "FAIL: Thunder Storm never struck"; fail=1; }
  # Druid: ravens, area spells
  grep -qE "MINION spawn name=Raven" $log.txt || { echo "FAIL: no ravens"; fail=1; }
  grep -qE "STATE storm skill=\"Hurricane\"" $log.txt || { echo "FAIL: Hurricane"; fail=1; }
  # feat/skills-npsd: curses change the monster AI, Hydra raises three shooters, Poppy a vine, Hunger leeches
  grep -qE "STATE forced unit=.* kind=blind" $log.txt || { echo "FAIL: Dim Vision did not force the blind AI"; fail=1; }
  grep -qE "STATE forced unit=.* kind=confuse" $log.txt || { echo "FAIL: Confuse did not force the confuse AI"; fail=1; }
  grep -qE "STATE forced unit=.* kind=attract" $log.txt || { echo "FAIL: Attract did not force the attract AI"; fail=1; }
  grep -qE "SUMMON skill=\"Hydra\" key=hydra1 kind=trap created=3" $log.txt || { echo "FAIL: Hydra did not raise three hydras"; fail=1; }
  grep -qE "TRAP armed skill=\"Hydra\"" $log.txt || { echo "FAIL: the hydras were not armed"; fail=1; }
  grep -qE "SUMMON skill=\"Plague Poppy\" key=plaguepoppy kind=trap created=1" $log.txt || { echo "FAIL: Plague Poppy"; fail=1; }
  grep -qE "SKILL area skill=\"Poison Nova\"|MISSILE hit name=poisonnova|CAST do skill=\"Poison Nova\" .*missiles=[1-9]" $log.txt || { echo "FAIL: Poison Nova"; fail=1; }
  grep -qE "SKILL melee skill=\"Hunger\"" $log.txt || { echo "FAIL: Hunger never struck"; fail=1; }
  grep -qE "(STATE convert skill=\"Conversion\"|SKILL melee skill=\"Conversion\")" $log.txt || { echo "FAIL: Conversion neither converted nor struck"; fail=1; }
  # Assassin: charges released, sentries, shadow
  grep -qE "STATE apply skill=\"Fists of Fire\" .*state=progressive_fire" $log.txt || { echo "FAIL: Fists of Fire charge"; fail=1; }
  grep -qE "STATE clear skill=\"Dragon Talon\"" $log.txt || { echo "FAIL: Dragon Talon released no charge"; fail=1; }
  grep -qE "TRAP fire skill=\"Lightning Sentry\"" $log.txt || { echo "FAIL: the Lightning Sentry never fired"; fail=1; }
  grep -qE "MINION spawn name=Shadow Warrior" $log.txt || { echo "FAIL: no Shadow Warrior"; fail=1; }
  # the totals
  grep -qE "AUTOCAST summary .* kills=[1-9][0-9]* .* area=[1-9]" $log.txt || { echo "FAIL: no kills or area hits in the summary"; fail=1; }
}
