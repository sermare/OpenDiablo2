# Shared by the playthrough scenarios (9d, 9e, 9f, 9g, 9h, 9i, 9j, 9l): make the level 94 hero they start from.
#
#   make_hero <out.d2s>      needs D2_TABLES and D2S_SAMPLE_BODY; returns non-zero when the file could not be made
#
# OD2_HERO selects the hero (default: the sample Sorceress):
#   (unset) | sorc   the sample Sorceress, revived (scripts/d2s-revive.go: she is a dead hardcore character).
#                    She runs out of mana (the engine has no natural mana regeneration yet) and melees with a broken flail.
#   barb | barbarian a level 94 Barbarian generated at run time from the game tables (scripts/d2s-make-hero.go,
#                    d2core/d2hero/herogen): a unique two-hand sword, unique gear, 16 belt potions. It shares the sample's
#                    quests, waypoints, map seed, difficulty and mercenary, so the scenario plays the same game.
# The generated file is written next to the scenario's other scratch files and never committed.
make_hero() {
  local out=$1 msg
  case "${OD2_HERO:-sorc}" in
    sorc|sorceress)
      msg=$(go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" "$out" "$D2_TABLES" 2>&1) || { echo "make_hero: revive failed: $msg" >&2; return 1; } ;;
    barb|barbarian)
      msg=$(go run scripts/d2s-make-hero.go -class barbarian -template "$D2S_SAMPLE_BODY" "$out" "$D2_TABLES" 2>&1) || { echo "make_hero: barbarian failed: $msg" >&2; return 1; }
      echo "# OD2_HERO=barb: $msg" >&2 ;;
    *) echo "make_hero: unknown OD2_HERO=${OD2_HERO} (use sorc or barb)" >&2; return 1 ;;
  esac
}
