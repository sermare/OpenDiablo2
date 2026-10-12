scenario_name="new Assassin made in the menu (20/20/20/25, life 50 mana 25 stamina 95, katar and buckler, 4 potions, 2 scrolls; same after save and reload)"
source scripts/verify.d/lib/newchar.sh
scenario_env() { newchar_env Assassin Sinny; }
scenario_check() { newchar_check 50 25 95 20 20 20 25 " weapon=Katar shield=Buckler\]" 2 4 "Attack"; }
