scenario_name="Levels walked, Act 1 caves (Underground Passage 1-2, Hole 1-2, Pit 1-2, Burial Grounds, Crypt, Mausoleum)"
source scripts/verify.d/lib/levelwalk.sh
scenario_warnings_ok=1
scenario_env() {
  lw_s=""; lw_town=1
  lw_into 10; lw_hop 14; lw_home
  lw_into 11; lw_hop 15; lw_home
  lw_into 12; lw_hop 16; lw_home
  lw_into 17; lw_hop 18; lw_hop 17; lw_hop 19; lw_home
  lw_env act1caves 1 "$lw_s"
}
scenario_check() { lw_check act1caves 10 14 11 15 12 16 17 18 19; }
