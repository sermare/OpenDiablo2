#!/bin/zsh
# Tests scripts/verify.d/lib/hero.sh without a game: the OD2_HERO names, the class name 9b hands to OD2_AUTONEWCHAR,
# and (with D2_TABLES and D2S_SAMPLE_BODY set) that make_hero writes a hero file for every class.
#   D2_TABLES=... D2S_SAMPLE_BODY=... scripts/test_hero_sh.sh
set -u
cd "${0:A:h}/.." || exit 1
source scripts/verify.d/lib/hero.sh

fail=0
check() { if eval "$2"; then echo "ok   $1"; else echo "FAIL $1"; fail=1; fi; }

typeset -A preset=(
  "" "" sample "" revived "" amazon amazon ama amazon sorc sorc sorceress sorc necro necro necromancer necro
  paladin paladin pal paladin barb barb barbarian barb druid druid dru druid assassin assassin sin assassin ass assassin
)
typeset -A cname=(
  "" Sorceress sample Sorceress amazon Amazon sorc Sorceress necro Necromancer paladin Paladin barb Barbarian
  druid Druid assassin Assassin sin Assassin
)

for k in "${(@k)preset}"; do
  got=$(OD2_HERO=$k hero_preset)
  check "OD2_HERO='$k' -> preset '${preset[$k]}'" '[ "$got" = "${preset[$k]}" ]'
done

for k in "${(@k)cname}"; do
  got=$(OD2_HERO=$k hero_class_name)
  check "OD2_HERO='$k' -> OD2_AUTONEWCHAR=${cname[$k]}" '[ "$got" = "${cname[$k]}" ]'
done

# unset behaves like the sample
unset OD2_HERO
check "unset OD2_HERO -> the sample Sorceress" '[ -z "$(hero_preset)" ] && [ "$(hero_class_name)" = Sorceress ]'

# an unknown name is refused
OD2_HERO=wizard
out=$(make_hero /tmp/never.d2s 2>&1); rc=$?
check "unknown OD2_HERO is refused" '[ $rc -ne 0 ] && echo "$out" | grep -q "unknown OD2_HERO=wizard" && [ ! -e /tmp/never.d2s ]'

if [ -n "${D2_TABLES:-}" ] && [ -n "${D2S_SAMPLE_BODY:-}" ]; then
  d=$(mktemp -d /tmp/od2-herosh.XXXXXX)
  for c in amazon sorc necro paladin barb druid assassin; do
    OD2_HERO=$c make_hero $d/$c.d2s 2> $d/$c.err
    check "make_hero writes the $c hero" '[ -s $d/$c.d2s ] && grep -q "OD2_HERO=$c:" $d/$c.err'
  done
  rm -rf $d
else
  echo "skip make_hero (D2_TABLES / D2S_SAMPLE_BODY not set)"
fi

[ $fail -eq 0 ] && echo "ALL hero.sh TESTS PASSED" || { echo "hero.sh TESTS FAILED"; exit 1; }
