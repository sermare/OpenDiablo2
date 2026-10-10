#!/bin/zsh
# Full verification in parallel: the scenarios of scripts/verify.d are split round-robin (slow ones first, so they land
# in different jobs) over N independent verify.sh runs (own port, scratch dir and fresh sample save each).
#   scripts/verify_parallel.sh [jobs]      default 6; env as for verify.sh; logs in /tmp/verify-par-<pid>-<n>.log
# Exit 0 only if every job printed ALL CHECKS PASSED.
jobs=${1:-6}
cd "${0:A:h}/.." || exit 1
slow=(9b-act1-playthrough 9e-act2-playthrough 9g-act3-durance 9h-act45-playthrough 9i-act5-caves-playthrough 99-act-travel 96-multiplayer 9d-party-trade 9f-pvp-skills-ear 9g-town-portals 83-cave-chain-persist 9f-act2-lutn 86-class-skills 94-perf-real-levels 9e-cube 9g-quest-rewards-uber 9c-act1-reload 98-skillbar)
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
    OD2_VERIFY_ONLY="(${grp[$n]})*" ./scripts/verify.sh > $log 2>&1 &
  else
    SKIP_UNIT=1 OD2_VERIFY_ONLY="(${grp[$n]})*" ./scripts/verify.sh > $log 2>&1 &
  fi
  pids+=($!)
  sleep 2
done
rc=0
for k in {1..${#pids}}; do wait ${pids[$k]} || rc=1; done
for l in $logs; do printf '%s: ' $l; grep -E "ALL CHECKS PASSED|SOME CHECKS FAILED" $l | tail -1; grep -E "^FAIL" $l | sort -u | head -5; done
[ $rc -eq 0 ] && echo "PARALLEL VERIFY: ALL JOBS PASSED" || echo "PARALLEL VERIFY: FAILED"
exit $rc
