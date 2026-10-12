scenario_name="new Amazon made in the menu (20/25/20/15, life 50 mana 15 stamina 84, javelin and buckler, 4 potions, 2 scrolls; same after save and reload)"
source scripts/verify.d/lib/newchar.sh
scenario_env() { newchar_env Amazon Amaz; }
scenario_check() { newchar_check 50 15 84 20 25 20 15 " weapon=Javelin shield=Buckler" 2 4 "Attack" 60; }
