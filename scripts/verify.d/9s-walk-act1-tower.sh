scenario_name="Levels walked, Forgotten Tower and Tower Cellar 1-5"
source scripts/verify.d/lib/levelwalk.sh
scenario_warnings_ok=1
scenario_env() {
  lw_s=""; lw_town=1
  lw_into 20; lw_hop 21; lw_hop 22; lw_hop 23; lw_hop 24; lw_hop 25; lw_home
  lw_env act1tower 1 "$lw_s"
}
scenario_check() { lw_check act1tower 20 21 22 23 24 25; }
