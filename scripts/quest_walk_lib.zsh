# Helpers of the quest walkthrough scenarios (scripts/verify.d/9j-*.sh .. 9k-*.sh). Sourced by them; builds
# OD2_AUTOSCRIPT text and the environment of a run with the revived level 94 sample hero whose quests were reset.
#
#   qw_env <name> <script>     echoes the exports of a scenario (needs D2_TABLES and D2S_SAMPLE_BODY); the revived
#                              hero and the write-back folder are in $tmp/<name>/
#   qw_begin [act]             script start: reset quests, empty the inventory grid, finish act 1 (for the trip east)
#                              and travel to the town of the act
#   qw_talk <npc>              walk to the NPC, wait for the menu, choose Talk, wait for the speech
#   qw_go <level>...           walk through the exits (borders, stairs, entrances) to each level in turn
#   qw_wp <level>              open the waypoint of the level the hero stands in and travel to <level>
#   qw_chest <object> [secs]   walk to a (quest) object (name or objects.txt row), then pick up the quest items near it
#   qw_panel <act> <quest>     log the title and page text the quest log panel shows

qw_talk() { printf 'move:npc=%s;until:NPC menu opened: npc="%s",60;menu:Talk;wait:%s;' "$1" "$1" "${2:-8}"; }
qw_go() { local l; for l in "$@"; do printf 'walkto:exit=%s;expect:level=%s;wait:2;' "$l" "$l"; done; }
qw_wp() { printf 'use:Waypoint;waypoint:%s;expect:level=%s;wait:3;' "$1" "$1"; }
qw_chest() { printf 'walkto:object=%s;wait:3;say:lootquest 15 %s;' "$1" "${2:-30}"; }
qw_panel() { printf 'say:questpanel %s %s;' "$1" "$2"; }
qw_begin() {
  printf 'wait:1;say:resetquests;say:clearinv;'
  case "${1:-2}" in
    2) printf 'say:completequest 1 6;travel:2;expect:level=40;wait:3;' ;;
    3) printf 'say:completequest 1 6;travel:2;expect:level=40;say:completequest 2 6;travel:3;expect:level=75;wait:3;' ;;
    4) printf 'say:completequest 1 6;travel:2;expect:level=40;say:completequest 2 6;travel:3;expect:level=75;say:completequest 3 6;travel:4;expect:level=103;wait:3;' ;;
    5) printf 'say:completequest 1 6;travel:2;expect:level=40;say:completequest 2 6;travel:3;expect:level=75;say:completequest 3 6;travel:4;expect:level=103;say:completequest 4 2;travel:5;expect:level=109;wait:3;' ;;
  esac
}

qw_env() {
  local name=$1 script=$2 d=$tmp/$1
  mkdir -p $d/s $d/wb; rm -f $d/s/*.d2s(N) $d/wb/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $d/s/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$d/s/Hero.d2s\" OD2_D2S_WRITEBACK=\"$d/wb\" OD2_REALMAPS=1 OD2_AUTOSPEED=8 OD2_AUTOMONSTER_DIFF=0"
    echo "export OD2_AUTOSCRIPT='$script'"
  else
    echo "# no revived sample (D2_TABLES unset or the tool failed)" >&2
    echo "export OD2_AUTOSCRIPT='wait:1;exit'"
  fi
}

# qw_sample <name>: succeeds when the scenario had a revived hero (else the check skips)
qw_sample() { [ -s $tmp/$1/s/Hero.d2s ]; }

# qw_laterquests: the laterquests=[...] list of the last .d2s the game exported
qw_laterquests() { grep -E "D2S EXPORT reparse" $log.txt | tail -1 | grep -oE "laterquests=\[[^]]*\]"; }
qw_act1quests() { grep -E "D2S EXPORT reparse" $log.txt | tail -1 | grep -oE "act1quests=\[[^]]*\]"; }
# qw_slot <slot> <hexmask>: the slot of the exported .d2s has all bits of the mask (4 hex digits, e.g. 0x0001)
qw_slot_has() {
  local line v; line=$(qw_laterquests); [ -n "$line" ] || line=$(qw_act1quests)
  v=$(echo "$line" | grep -oE "(^|[ \[])$1:0x[0-9a-f]{4}" | tail -1 | sed 's/.*://')
  [ -n "$v" ] && (( (v & $2) == $2 ))
}
