# Shared by the Nightmare / Hell scenarios (9u-nm-*, 9u-hell-*): start a strong hero at a difficulty, check the monster
# scaling against hand computations of the real tables, the resist penalty, the automap difficulty line, the Andariel
# kill, the drop levels and the act travel of that difficulty. Wave 36 (notes: ~/git/d2-re-notes/w36-hell.md).
#
#   w36_hero_name            the hero preset the scenarios use (OD2_HERO, default the generated Barbarian "barb": the
#                            sample Sorceress dies to Hell quill rats, so the default hero of these scenarios is the
#                            generated level 94 Barbarian; any other OD2_HERO skips the exact resist numbers)
#   w36_play_env <1|2>       scenario_env of the play scenario (difficulty 1 Nightmare, 2 Hell)
#   w36_play_check <1|2>     scenario_check of the play scenario
#   w36_quest_env <1|2> <act1|act5>, w36_quest_check <1|2> <act1|act5>   the quest reward scenarios (OD2_AUTOQUEST)
# Variables of this file start with w36_ (the runner shares one shell with the scenarios: n, f, s, log, tmp are taken).

w36_hero_name() { echo "${OD2_HERO:-barb}"; }

w36_diffname() { case $1 in 1) echo Nightmare ;; 2) echo Hell ;; *) echo Normal ;; esac; }

# the hand computations of the real tables (patch_d2 monlvl.txt L-* columns x monstats percents, integer division like the
# exe's MulDiv; every number was read off the tables when this was written, see the notes):
#   Blood Moor monsters: levels.txt MonLvl2Ex 36 (Nightmare) / MonLvl3Ex 67 (Hell)
#   quillrat1 N: defense 528*80/100, to-hit 738*65/100, damage 31*30/100..31*75/100, xp 2568*70/100
#   quillrat1 H: defense 1334*80/100, to-hit 2724*65/100, damage 76*30/100..76*75/100, xp 26733*70/100
#   Andariel (a boss: monstats Level 49 / 75, not the area level): life 968*2562/100 (N), 5032*1193/100 (H);
#   defense 684*110/100 (N), 1475*110/100 (H)
w36_rat_line() {
  case $1 in
    1) echo "level=36 hp=[0-9]+ defense=$((528*80/100)) xp=$((2568*70/100)) tc=\"[^\"]*\(N\)\" a1=$((738*65/100))/$((31*30/100))-$((31*75/100))" ;;
    2) echo "level=67 hp=[0-9]+ defense=$((1334*80/100)) xp=$((26733*70/100)) tc=\"[^\"]*\(H\)\" a1=$((2724*65/100))/$((76*30/100))-$((76*75/100))" ;;
  esac
}
w36_andariel_line() {
  case $1 in
    1) echo "level=49 hp=$((968*2562/100)) defense=$((684*110/100))" ;;
    2) echo "level=75 hp=$((5032*1193/100)) defense=$((1475*110/100))" ;;
  esac
}
# the shown resistances of the generated Barbarian (raw fire/cold/lightning 97, poison 149, shown capped at 75 and with the
# difficulty penalty -40 Nightmare / -100 Hell taken off first): Normal 75/75/75/75
w36_res_line() {
  case $1 in
    1) echo "fire:57,cold:57,light:57,poison:75" ;;
    2) echo "fire:-3,cold:-3,light:-3,poison:49" ;;
  esac
}
w36_min_ilvl() { case $1 in 1) echo 36 ;; 2) echo 67 ;; esac; }

w36_play_env() {
  local OD2_HERO=$(w36_hero_name) w36_d=$1 w36_s
  local w36_dir=$tmp/w36play
  mkdir -p $w36_dir/s94 $w36_dir/wb94; rm -f $w36_dir/s94/Hero.d2s $w36_dir/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && make_hero $w36_dir/s94/Hero.d2s; then
    # the quest record of this difficulty is empty: act travel is refused until the quests are done
    w36_s="wait:1;say:resetquests;refuse:2;panel:character;wait:1;panel:close;automap:on;automap:stats;wait:1;automap:off"
    # Blood Moor: a fight at the difficulty's monster level
    w36_s+=";walkto:exit=2;expect:level=2;wait:2;kill:near=30,45;loot:30,40;walkto:exit=1;expect:level=1"
    # Andariel: the portal to the Catacombs, the stairs down, the boss fight (OD2_AUTOSCRIPT_HEAL=1 tops the hero's life
    # up on every update: the scripted strong hero); the corpse portal leads back to the camp
    w36_s+=";wait:8;say:spawnportal 36;use:Portal;expect:level=36;walkto:exit=37;expect:level=37;wait:3"
    w36_s+=";kill:name=Andariel,120"
    w36_s+=";wait:2;use:Portal;expect:level=1;wait:2"
    # a player talks to Warriv first: the Andariel kill leaves slot 6 at reward pending, bit 0 (done) comes from his talk and
    # is the only thing the server tests for the trip (0x577b70)
    w36_s+=";move:npc=Warriv;until:NPC menu opened: npc=\"Warriv\",60;menu:Talk;wait:6;panel:close"
    # act travel of this difficulty: Warriv (Andariel is dead), then the quest flags of the later acts are forced
    w36_s+=";travel:2;expect:level=40;wait:2;refuse:4;say:completequest 2 6;travel:3;expect:level=75;wait:2"
    w36_s+=";say:completequest 3 6;travel:4;expect:level=103;wait:2;refuse:3;say:completequest 4 2;travel:5;expect:level=109;wait:2;exit"
    echo "export OD2_AUTOGAME=\"$w36_dir/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$w36_dir/wb94\" OD2_AUTOSPEED=4 OD2_AUTODIFFICULTY=$w36_d OD2_AUTOSCRIPT_HEAL=1"
    echo "export OD2_AUTOSCRIPT='$w36_s'"
  else
    echo "# no hero (D2_TABLES unset or the tool failed)" >&2
    echo "export OD2_AUTOSCRIPT='wait:1;exit'"
  fi
}

w36_play_check() {
  local w36_d=$1 w36_name w36_rat w36_and w36_ilvl
  w36_name=$(w36_diffname $w36_d)
  grep -E "DIFFICULTY|AUTOMAP text|PANEL character: level=[0-9]+ str|MONSTER spawn name=Andariel|MONSTER death name=Andariel|HERO died|QUEST A1Q6|LEVEL CHANGE .*act|QUEST travel|AUTOSCRIPT RESULT|D2S EXPORT reparse" $log.txt | cut -c1-240 | tail -40
  [ -s $tmp/w36play/s94/Hero.d2s ] || { echo "SKIP: no hero in this run"; return; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the $w36_name scenario did not pass"; fail=1; }
  grep -E "AUTOSCRIPT step [0-9]+ FAIL" $log.txt | cut -c1-200
  # a hero with the unlock flags starts at the difficulty through the difficulty screen
  grep -q "DIFFICULTY screen .*unlocked=\[true true true\]" $log.txt || { echo "FAIL: the hero does not have Nightmare and Hell unlocked"; fail=1; }
  grep -q "DIFFICULTY chosen .*difficulty=$w36_name" $log.txt || { echo "FAIL: $w36_name was not chosen"; fail=1; }
  grep -q "DIFFICULTY .* refused" $log.txt && { echo "FAIL: the difficulty was refused"; fail=1; }
  # the monsters of the game are all of this difficulty, and the quill rats match the hand computation of the real tables
  if grep -q "DIFFTEST monster .*difficulty=[0-9]" $log.txt; then
    grep "DIFFTEST monster" $log.txt | grep -qv "difficulty=$w36_d " && { echo "FAIL: a monster spawned at another difficulty"; fail=1; }
  else
    echo "FAIL: no DIFFTEST monster lines"; fail=1
  fi
  w36_rat=$(w36_rat_line $w36_d)
  grep -qE "DIFFTEST monster id=quillrat1 difficulty=$w36_d $w36_rat" $log.txt || { echo "FAIL: the $w36_name quill rat differs from the hand computation of the real tables ($w36_rat)"; fail=1; }
  # the resist penalty of the difficulty on the character panel (the generated Barbarian has known raw resistances)
  if [ "$(w36_hero_name)" = barb ]; then
    grep -qE "PANEL character: level=94 str=.* res=$(w36_res_line $w36_d) " $log.txt || { echo "FAIL: the shown resistances lack the $w36_name penalty (want $(w36_res_line $w36_d))"; fail=1; }
  fi
  # the automap shows the difficulty after the version line
  grep -q "AUTOMAP text lines=\[.*\"v 1.14b\" \"Difficulty: $w36_name\"\]" $log.txt || { echo "FAIL: the automap lacks its difficulty line"; fail=1; }
  # Andariel: boss level and life from the monstats row and the monlvl table, killed by the scripted hero, the quest records it
  w36_and=$(w36_andariel_line $w36_d)
  grep -qE "MONSTER spawn name=Andariel .*ai=Andariel $w36_and " $log.txt || { echo "FAIL: Andariel differs from the hand computation ($w36_and)"; fail=1; }
  grep -q "MONSTER death name=Andariel" $log.txt || { echo "FAIL: Andariel was not killed"; fail=1; }
  grep -qE "QUEST A1Q6 slot=6 bits .*Andariel killed" $log.txt || { echo "FAIL: the quest did not see Andariel die"; fail=1; }
  grep -q "HERO died" $log.txt && { echo "FAIL: the scripted strong hero died"; fail=1; }
  # drop levels: the items of monsters are at least the monster level of the difficulty, Andariel's at her own level
  w36_ilvl=$(w36_min_ilvl $w36_d)
  if ! grep -q "ITEMGEN created source=monster" $log.txt; then echo "FAIL: no monster drops"; fail=1; fi
  grep "ITEMGEN created source=monster" $log.txt | sed -E 's/.* ilvl=([0-9]+) .*/\1/' | awk -v m=$w36_ilvl '$1 < m {bad=1} END {exit bad}' \
    || { echo "FAIL: a monster drop below item level $w36_ilvl"; fail=1; }
  grep "ITEMGEN created source=monster monster=Andariel" $log.txt | grep -qvE " ilvl=$(w36_andariel_line $w36_d | sed -E 's/level=([0-9]+).*/\1/') " \
    && { echo "FAIL: an Andariel drop is not at her monster level"; fail=1; }
  # act travel works at the difficulty: refused on the empty record, then the trips of the four acts
  for w36_t in "1 to=40" "40 to=75" "75 to=103" "103 to=109"; do
    grep -qE "LEVEL CHANGE from=${w36_t% *} to=${w36_t#* to=} .*via=act" $log.txt || { echo "FAIL: no act trip $w36_t"; fail=1; }
  done
  # the saved hero keeps the difficulty and the act of the last trip
  grep "D2S EXPORT reparse" $log.txt | tail -1 | grep -qE "difficulty=$w36_d act=5 " || { echo "FAIL: the exported .d2s lacks difficulty $w36_d / act 5"; fail=1; }
}

# ---- quest rewards (OD2_AUTOQUEST with the difficulty chosen through the difficulty screen) ----

w36_quest_env() {
  echo "export OD2_AUTODIFFICULTY=$1 OD2_AUTOQUEST=$2 OD2_D2S_WRITEBACK=$tmp/wbq"
  mkdir -p $tmp/wbq
}

w36_quest_check() {
  local w36_d=$1 w36_act=$2 w36_name w36_lo w36_hi w36_tier w36_code
  w36_name=$(w36_diffname $w36_d)
  grep -E "DIFFICULTY chosen|AUTOQUEST (start|RESULT)|QUEST EFFECT (give-item|reward (gold|experience|resist))" $log.txt | cut -c1-240 | tail -24
  grep -q "DIFFICULTY chosen .*difficulty=$w36_name" $log.txt || { echo "FAIL: $w36_name was not chosen"; fail=1; }
  grep -q "AUTOQUEST RESULT PASS" $log.txt || { echo "FAIL: the $w36_act quest line did not pass at $w36_name"; fail=1; }
  grep -E "AUTOQUEST expect .*FAIL" $log.txt | cut -c1-200
  grep -q "AUTOQUEST start .* difficulty=$w36_d" $log.txt || { echo "FAIL: the quest record is not the one of $w36_name"; fail=1; }
  case $w36_act in
  act1)
    # Akara's ring for finding Cain: item level and quality by difficulty (VERIFIED 0x58ffb0): 7 magic / 30 rare / 60 rare
    case $w36_d in 1) w36_code="code=rin quality=2 ilvl/count=30" ;; 2) w36_code="code=rin quality=2 ilvl/count=60" ;; esac
    grep -qE "QUEST EFFECT give-item $w36_code " $log.txt || { echo "FAIL: Akara's ring is not item level/quality for $w36_name ($w36_code)"; fail=1; }
    ;;
  act5)
    # Baal's death: gold piles between difficulty*6000+1500 and +3000 (VERIFIED 0x58bce0)
    w36_lo=$((w36_d*6000+1500)); w36_hi=$((w36_d*6000+3000))
    grep -qE "QUEST EFFECT reward gold-pile value=$w36_lo .*range $w36_lo-$w36_hi," $log.txt || { echo "FAIL: Baal's gold is not $w36_lo-$w36_hi"; fail=1; }
    # Rite of Passage pays more experience each difficulty, the level gate is 20 * (difficulty + 1) (the hero is above it)
    grep -qE "QUEST EFFECT reward experience " $log.txt || { echo "FAIL: no Rite of Passage experience"; fail=1; }
    # Malah's potion once per game (the step after the potion was used asks her again)
    [ "$(grep -c 'QUEST EFFECT give-item code=ice ' $log.txt)" = 1 ] || { echo "FAIL: Malah's potion was given other than once"; fail=1; }
    grep -q "AUTOQUEST expect Malah gives the potion only once per game" $log.txt || { echo "FAIL: the once-per-game check did not run"; fail=1; }
    # Anya's item tier by difficulty and hero level (the sample Sorceress is level 94): exceptional (Nightmare) / elite (Hell)
    case $w36_d in 1) w36_tier="ob6|ob7|ob8|ob9|oba" ;; 2) w36_tier="obb|obc|obd|obe|obf" ;; esac
    grep -qE "QUEST EFFECT give-item code=($w36_tier) quality=2 ilvl/count=94 " $log.txt || { echo "FAIL: Anya's item is not of the $w36_name tier ($w36_tier)"; fail=1; }
    ;;
  esac
}
