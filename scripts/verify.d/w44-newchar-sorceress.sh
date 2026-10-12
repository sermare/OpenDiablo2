scenario_name="new Sorceress made in the menu (10/25/10/35, life 40 mana 35 stamina 74, short staff, 4 potions, 2 scrolls; same after save and reload)"
source scripts/verify.d/lib/newchar.sh
scenario_env() { newchar_env Sorceress Sorcy; }
scenario_check() { newchar_check 40 35 74 10 25 10 35 " weapon=Short Staff\]" 2 4 "Attack" 0 "Fire Bolt\(id=36,lvl=1\)"; }
