scenario_name="Levels walked, Act 3 temples (Disused Fane, Forgotten Temple, Ruined Fane, Disused Reliquary)"
source scripts/verify.d/lib/levelwalk.sh
scenario_warnings_ok=1
scenario_env() {
  lw_s=""; lw_town=75
  lw_into 95; lw_home
  lw_into 97; lw_home
  lw_into 98; lw_home
  lw_into 99; lw_home
  lw_env act3fanes 3 "$lw_s"
}
scenario_check() { lw_check act3fanes 95 97 98 99; }
