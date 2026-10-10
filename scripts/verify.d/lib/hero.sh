# Shared by the playthrough scenarios (9b, 9d, 9e, 9f, 9g, 9h, 9i, 9j, 9l): make the level 94 hero they start from.
#
#   make_hero <out.d2s>      needs D2_TABLES and D2S_SAMPLE_BODY; returns non-zero when the file could not be made
#   hero_class_name          the class name for OD2_AUTONEWCHAR (9b starts a fresh level 1 hero of the class)
#
# OD2_HERO selects the hero (default: the sample Sorceress):
#   (unset) | sample   the sample Sorceress, revived (scripts/d2s-revive.go: she is a dead hardcore character).
#                      She melees with a broken flail and casts Blizzard and Fire Ball from the right button.
#   amazon | sorc | necro | paladin | barb | druid | assassin
#                      a level 94 hero of the class generated at run time from the game tables
#                      (scripts/d2s-make-hero.go, d2core/d2hero/herogen): class-legal unique gear, a skill build around
#                      one main attack on the left button, 16 belt potions. It shares the sample's quests, waypoints, map
#                      seed, difficulty and mercenary, so the scenario plays the same game. (sorc is the generated Fire
#                      Sorceress; the sample Sorceress is "sample".)
# The generated file is written next to the scenario's other scratch files and never committed.
# scripts/verify_classes.sh runs the Act 1 to 5 chain once per class with this variable (nightly, slow).

# hero_preset prints the herogen preset name of OD2_HERO, or nothing for the sample Sorceress.
hero_preset() {
  case "${OD2_HERO:-}" in
    ""|sample|revived) ;;
    amazon|ama) echo amazon ;;
    sorc|sorceress) echo sorc ;;
    necro|necromancer) echo necro ;;
    paladin|pal) echo paladin ;;
    barb|barbarian) echo barb ;;
    druid|dru) echo druid ;;
    assassin|sin|ass) echo assassin ;;
    *) echo "make_hero: unknown OD2_HERO=${OD2_HERO} (use sample, amazon, sorc, necro, paladin, barb, druid or assassin)" >&2; echo "?" ;;
  esac
}

# hero_class_name prints the class name of OD2_HERO as OD2_AUTONEWCHAR spells it (Sorceress when unset).
hero_class_name() {
  case "$(hero_preset)" in
    amazon) echo Amazon ;;
    necro) echo Necromancer ;;
    paladin) echo Paladin ;;
    barb) echo Barbarian ;;
    druid) echo Druid ;;
    assassin) echo Assassin ;;
    *) echo Sorceress ;;
  esac
}

make_hero() {
  local out=$1 msg preset
  preset=$(hero_preset)
  case "$preset" in
    "")
      msg=$(go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" "$out" "$D2_TABLES" 2>&1) || { echo "make_hero: revive failed: $msg" >&2; return 1; } ;;
    "?") return 1 ;;
    *)
      msg=$(go run scripts/d2s-make-hero.go -class "$preset" -template "$D2S_SAMPLE_BODY" "$out" "$D2_TABLES" 2>&1) || { echo "make_hero: $preset failed: $msg" >&2; return 1; }
      echo "# OD2_HERO=$preset: $msg" >&2 ;;
  esac
}
