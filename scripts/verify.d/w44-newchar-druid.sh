scenario_name="new Druid made in the menu (15/20/25/20, life 55 mana 20 stamina 84, club and buckler, 4 potions, 2 scrolls; same after save and reload)"
source scripts/verify.d/lib/newchar.sh
scenario_env() { newchar_env Druid Druidy; }
scenario_check() { newchar_check 55 20 84 15 20 25 20 " weapon=Club shield=Buckler\]" 2 4 "Attack"; }
