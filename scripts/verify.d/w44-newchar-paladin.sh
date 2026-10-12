scenario_name="new Paladin made in the menu (25/20/25/15, life 55 mana 15 stamina 89, short sword and buckler, 4 potions, 2 scrolls; same after save and reload)"
source scripts/verify.d/lib/newchar.sh
scenario_env() { newchar_env Paladin Pally; }
scenario_check() { newchar_check 55 15 89 25 20 25 15 " weapon=Short Sword shield=Buckler\]" 2 4 "Attack"; }
