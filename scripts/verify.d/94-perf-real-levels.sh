scenario_realtime=1
scenario_name="performance on real generated levels (OD2_REALMAPS: start-up and level-build budget, 120 monsters under full-screen lighting; generous limits so a loaded machine does not flake)"
# Frigid Highlands (Act 5 outdoor with exact tiles) is the slowest level to build. The marks come from d2util.PerfMark /
# PerfTime ("PERF mark <name> t_ms=<ms since process start>", "PERF span generate level=N ms=<ms>"). The limits are ten to
# twenty times the measured times (assets ready ~0.12 s, level build ~0.5 s cold and ~0.1 s warm, playable ~2 s, passes
# under 2 ms); they only catch accidental O(n^2) work, per-level file re-reading or a lost cache.
scenario_env() {
  echo 'export OD2_REALMAPS=1 OD2_AUTOLEVEL=111 OD2_AUTOPERF=1 OD2_AUTOPERF_SECONDS=5 OD2_AUTOPERF_WARMUP=6 OD2_AUTOMONSTER="fallen1,120" OD2_AUTOMONSTER_SECONDS=120'
}
perf_limit() { # name value limit
  [ -z "$2" ] && { echo "FAIL: no $1"; fail=1; return; }
  awk -v a="$2" -v l="$3" 'BEGIN { exit !(a > l) }' && { echo "FAIL: $1 is $2 (limit $3)"; fail=1; }
}
scenario_check() {
  grep -E "PERF (mark|span|update|render|frame|runtime)" $log.txt | cut -c1-200
  local assets playable cold warm
  assets=$(grep -E "PERF mark assets-ready" $log.txt | sed -E 's/.*t_ms=([0-9.]+).*/\1/' | head -1)
  playable=$(grep -E "PERF mark game-playable" $log.txt | sed -E 's/.*t_ms=([0-9.]+).*/\1/' | head -1)
  cold=$(grep -E "PERF span generate level=111" $log.txt | sed -E 's/.*ms=([0-9.]+).*/\1/' | sed -n 1p)
  warm=$(grep -E "PERF span generate level=111" $log.txt | sed -E 's/.*ms=([0-9.]+).*/\1/' | sed -n 2p)
  perf_limit "time to assets ready (ms)" "$assets" 8000
  perf_limit "time to a playable game (ms)" "$playable" 40000
  perf_limit "first build of the level (ms)" "$cold" 20000
  perf_limit "second build of the level, caches warm (ms)" "$warm" 10000
  local kind avg
  for kind in update render; do
    avg=$(grep -E "PERF $kind frames=" $log.txt | sed -E 's/.* avg_ms=([0-9.]+).*/\1/' | head -1)
    perf_limit "$kind pass average with 120 monsters (ms)" "$avg" 8
  done
  grep -q "monsters_alive=1[0-9][0-9]" $log.txt || echo "note: fewer than 100 monsters were alive during the measurement"
}
