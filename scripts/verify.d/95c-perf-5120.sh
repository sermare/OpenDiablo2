scenario_realtime=1
scenario_name="frame-time meter at 5120x1440 (ultrawide; per-row tile cull; 120 monsters on a real level; generous limits)"
scenario_env() {
  echo 'export OD2_DISPLAY=5120x1440 OD2_AUTOLEVEL=111 OD2_AUTOPERF=1 OD2_AUTOPERF_SECONDS=6 OD2_AUTOPERF_WARMUP=6 OD2_AUTOMONSTER="fallen1,120" OD2_AUTOMONSTER_SECONDS=120'
}
scenario_check() {
  grep -E "PERF (update|render|frame|runtime)|DRAW|DISPLAY size" $log.txt | cut -c1-220 | head -12
  local kind avg
  # CPU time per pass is limited; the frame line (wall clock, includes the GPU present and vsync of the huge target) is only reported
  for kind in update render frame; do
    avg=$(grep -E "PERF $kind frames=" $log.txt | sed -E 's/.* avg_ms=([0-9.]+).*/\1/' | head -1)
    [ -z "$avg" ] && { echo "FAIL: no PERF $kind line"; fail=1; continue; }
    [ $kind = frame ] && continue
    awk -v a="$avg" 'BEGIN { exit !(a > 8.0) }' && { echo "FAIL: $kind pass averages ${avg} ms at 5120x1440 (limit 8)"; fail=1; }
  done
}
