scenario_name="death gold penalty with stash gold (OD2_AUTODEATH_STASH: level 10 loses 10% of carried + stash, the rest of the carried gold drops, the stash shrinks only for a shortfall)"
wb=$tmp/writeback-deathgold
scenario_env() {
  mkdir -p $wb
  # the sample hero carries 458536 gold; stash 3000 at level 10: loss 10% of 461536 = 46153 < carried, so 412383 drop
  # and the stash stays 3000; the sample is hardcore, so the overlay is the hardcore line; the stash-shortfall
  # and player-kill variants are covered by the d2hero unit tests
  echo "export OD2_D2S_WRITEBACK=\"$wb\" OD2_AUTODEATH=1 OD2_AUTODEATH_LEVEL=10 OD2_AUTODEATH_HP=20 OD2_AUTOMONSTER_DIFF=1 OD2_AUTODEATH_STASH=3000"
}
scenario_check() {
  grep -E "DEATH hero|DEATH overlay|D2S EXPORT reparse" $log.txt | cut -c1-330 | head -8
  grep -qE "DEATH hero=.* gold_dropped=412383 gold_kept=0 stash=3000 pk=false" $log.txt || { echo "FAIL: stash gold not counted in the loss (expect 412383 dropped)"; fail=1; }
  grep -qE "DEATH overlay \"Your deeds of valor will be remembered" $log.txt || { echo "FAIL: no hardcore death overlay"; fail=1; }
  grep -qE "D2S EXPORT reparse: .* gold=0 .* stashgold=3000 " $log.txt || { echo "FAIL: exported stash gold wrong"; fail=1; }
}
