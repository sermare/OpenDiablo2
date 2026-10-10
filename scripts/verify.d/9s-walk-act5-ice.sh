scenario_name="Levels walked, Act 5 ice caves and Nihlathak's temple (Frozen River, Drifter Cavern, Icy Cellar, Temple, Halls of Anguish, Pain, Vaught)"
source scripts/verify.d/lib/levelwalk.sh
scenario_warnings_ok=1
scenario_env() {
  lw_s=""; lw_town=109
  lw_into 114; lw_home
  lw_into 116; lw_home
  lw_into 119; lw_home
  lw_into 121; lw_hop 122; lw_hop 123; lw_hop 124; lw_home
  lw_env act5ice 5 "$lw_s"
}
scenario_check() { lw_check act5ice 114 116 119 121 122 123 124; }
