#!/bin/zsh
# Full verification in parallel: the scenarios of scripts/verify.d are split round-robin (slow ones first, so they land
# in different jobs) over N independent verify.sh runs (own port, scratch dir and fresh sample save each).
#   scripts/verify_parallel.sh [jobs]      default 6; env as for verify.sh; logs in /tmp/verify-par-<pid>-<n>.log
# Exit 0 only if every job printed ALL CHECKS PASSED.
jobs=${1:-6}
cd "${0:A:h}/.." || exit 1
# slow-first order by wall seconds measured alone (turbo scenarios with their turbo time): the turbo ones (9e-act2, 9g-act3,
# 9h-act45, 9i-*, 9j-quest-staff) dropped to 10-45 s and moved down; numbers in ~/git/d2-re-notes/turbo-mode.md
slow=(9b-act1-playthrough 9f-pvp-skills-ear 9d-act1-sample-hero 9g-town-portals 9j-quest-radament 9d-party-trade 9k-quest-izual 9s-walk-act1-monastery 9n-act3-dungeons 9j-quest-taintedsun 86-class-skills 9s-walk-act1-caves 9k-quest-siege 9s-walk-act5-ice 9s-walk-act2-rest 9g-act3-durance 9j-chaos-sanctuary 86b-missile-skills 9k-quest-blade 9m-act3-full 9j-realm-multiplayer 83-cave-chain-persist 9l-walkability 95b-perf-camp 9k-reward-npcs 9k-quest-lamesen 83-menu-flow 9i-act5-caves-playthrough 9l-ingame-audio 90-ambient-audio 98-skillbar 99-act-travel 96-multiplayer 9f-act2-lutn 94-perf-real-levels 9e-cube 9g-quest-rewards-uber 9c-act1-reload 9k-quest-rescue)
all=(${${(f)"$(ls scripts/verify.d/*.sh | sed 's#.*/##; s#\.sh$##')"}})
ordered=($slow ${all:|slow})
typeset -A grp
i=0
for s in $ordered; do [ -f scripts/verify.d/$s.sh ] || continue; n=$(( i % jobs )); grp[$n]="${grp[$n]:+${grp[$n]}|}$s"; i=$((i+1)); done
pids=(); logs=()
for n in {0..$((jobs-1))}; do
  [ -z "${grp[$n]}" ] && continue
  log=/tmp/verify-par-$$-$n.log; logs+=($log)
  if [ $n -eq 0 ]; then
    OD2_VERIFY_ONLY="(${grp[$n]}).sh" ./scripts/verify.sh > $log 2>&1 &
  else
    SKIP_UNIT=1 OD2_VERIFY_ONLY="(${grp[$n]}).sh" ./scripts/verify.sh > $log 2>&1 &
  fi
  pids+=($!)
  sleep 2
done
rc=0
for k in {1..${#pids}}; do wait ${pids[$k]} || rc=1; done
for l in $logs; do
  cnt=$(grep -c '^SCENARIOS RUN:' $l); nsec=$(grep -c '^== ' $l); run=$(grep '^SCENARIOS RUN:' $l | tail -1 | sed 's/.*: //')
  echo "$l: scenarios run=${run:-0} sections('== ')=$nsec $(grep -E '^SCENARIOS run=' $l | tail -1)"
  if [ "${run:-0}" -eq 0 ]; then echo "FAIL: job ran 0 scenarios ($l)"; rc=1; fi
  printf '%s: ' $l; grep -E "ALL CHECKS PASSED|SOME CHECKS FAILED" $l | tail -1; grep -E "^FAIL" $l | sort -u | head -5; done
[ $rc -eq 0 ] && echo "PARALLEL VERIFY: ALL JOBS PASSED" || echo "PARALLEL VERIFY: FAILED"
exit $rc
