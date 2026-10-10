scenario_name="Levels walked, Monastery Gate to Catacombs 2 on foot (the Warp -1 links 26-27-28 and 32-33 are crossed on the level borders; Outer Cloister, Barracks, Jail 2-3, Inner Cloister, Cathedral, Catacombs 1-2)"
source scripts/verify.d/lib/levelwalk.sh
scenario_warnings_ok=1
scenario_env() {
  lw_s=""; lw_town=1
  lw_into 26; lw_hop 27; lw_hop 28; lw_hop 29; lw_hop 30; lw_hop 31; lw_hop 32; lw_hop 33; lw_hop 34; lw_hop 35; lw_home
  lw_env act1mon 1 "$lw_s"
}
scenario_check() { lw_check act1mon 26 27 28 30 31 32 33 34 35; }
