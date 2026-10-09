scenario_realtime=1
scenario_name="frame-time meter (OD2_AUTOPERF reports update/render/frame percentiles and runtime stats; the engine stays far inside a 16.7 ms frame)"
scenario_env() { echo 'export OD2_AUTOPERF=1 OD2_AUTOPERF_SECONDS=6 OD2_AUTOPERF_WARMUP=4 OD2_AUTOMONSTER="fallen1,pack" OD2_AUTOMONSTER_SECONDS=120'; }
scenario_check() {
  grep -E "PERF (update|render|frame|runtime|cpu )" $log.txt | cut -c1-200
  local kind
  for kind in update render frame; do
    grep -qE "PERF $kind frames=[1-9][0-9]* avg_ms=[0-9.]+ p95_ms=[0-9.]+ p99_ms=[0-9.]+ max_ms=[0-9.]+" $log.txt || { echo "FAIL: no PERF $kind line"; fail=1; }
  done
  grep -qE "PERF runtime goroutines=[0-9]+ heap_alloc_mb=.* gc_cycles=[0-9]+" $log.txt || { echo "FAIL: no PERF runtime line"; fail=1; }
  # CPU time per pass, not wall-clock frame time: the machine may be busy with other programs. A regression to
  # per-frame scans (the input polling and entity lookups fixed in the perf pass cost 3 ms and more) would
  # show up here; the limit leaves ample room for a loaded machine.
  local avg
  for kind in update render; do
    avg=$(grep -E "PERF $kind frames=" $log.txt | sed -E 's/.* avg_ms=([0-9.]+).*/\1/' | head -1)
    [ -n "$avg" ] && awk -v a="$avg" 'BEGIN { exit !(a > 8.0) }' && { echo "FAIL: $kind pass averages ${avg} ms (limit 8)"; fail=1; }
  done
}
