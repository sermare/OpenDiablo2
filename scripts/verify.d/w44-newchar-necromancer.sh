scenario_name="new Necromancer made in the menu (15/25/15/25, life 45 mana 25 stamina 79, wand, 4 potions, 2 scrolls; same after save and reload)"
source scripts/verify.d/lib/newchar.sh
scenario_env() { newchar_env Necromancer Necro; }
scenario_check() { newchar_check 45 25 79 15 25 15 25 " weapon=Wand\]" 2 4 "Attack" 0 "Raise Skeleton\(id=70,lvl=1\)"; }
