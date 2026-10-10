scenario_name="Levels walked, Act 2 leftovers (Palace Cellar 2, Stony Tomb 1-2, Halls of the Dead 3)"
source scripts/verify.d/lib/levelwalk.sh
scenario_warnings_ok=1
scenario_env() {
  lw_s=""; lw_town=40
  lw_into 52; lw_hop 53; lw_home
  lw_into 55; lw_hop 59; lw_home
  lw_into 57; lw_hop 60; lw_home
  lw_env act2rest 2 "$lw_s"
}
scenario_check() { lw_check act2rest 53 55 59 60; }
