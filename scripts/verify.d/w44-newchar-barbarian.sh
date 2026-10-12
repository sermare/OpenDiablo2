scenario_name="new Barbarian made in the menu (30/20/25/10, life 55 mana 10 stamina 92, hand axe and buckler, 4 potions, 2 scrolls; same after save and reload)"
source scripts/verify.d/lib/newchar.sh
scenario_env() { newchar_env Barbarian Barby; }
scenario_check() { newchar_check 55 10 92 30 20 25 10 " weapon=Hand Axe shield=Buckler\]" 2 4 "Attack"; }
