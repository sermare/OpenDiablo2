scenario_realtime=1
scenario_timeout=240
scenario_warnings_ok=1
scenario_name="frame-time meter while the hero walks through the Rogue Encampment or Blood Moor (OD2_PERF_CAMP_ZONE=town|moor, default town; CPU profile to OD2_PPROF_CPU_OUT)"
# Measures update/render/frame percentiles while the hero is MOVING (the idle meter of 95-perf.sh misses per-step costs).
scenario_env() {
  local zone="${OD2_PERF_CAMP_ZONE:-town}"
  echo "export OD2_AUTOPERF=1 OD2_AUTOPERF_SECONDS=${OD2_PERF_CAMP_SECONDS:-30} OD2_AUTOPERF_WARMUP=${OD2_PERF_CAMP_WARMUP:-4} ${OD2_PERF_CAMP_ENV:-}"
  [ -n "${OD2_PPROF_CPU_OUT:-}" ] && echo "export OD2_PPROF_CPU=$OD2_PPROF_CPU_OUT"
  if [ "$zone" = moor ]; then
    echo "export OD2_AUTOSCRIPT='wait:1;walkto:exit=2;expect:level=2;wait:1;move:20,20;wait:6;move:60,60;wait:6;move:20,60;wait:6;move:60,20;wait:6;move:20,20;wait:6;move:60,60;wait:6;move:20,60;wait:6;move:60,20;wait:6;exit'"
  else
    echo "export OD2_AUTOSCRIPT='wait:1;move:npc=Akara;wait:7;panel:close;move:npc=Kashya;wait:7;panel:close;move:npc=Charsi;wait:7;panel:close;move:npc=Gheed;wait:7;panel:close;move:npc=Warriv;wait:7;panel:close;move:npc=Akara;wait:7;panel:close;move:npc=Kashya;wait:7;panel:close;move:npc=Charsi;wait:7;exit'"
  fi
}
scenario_check() {
  grep -E "PERF (update|render|frame|runtime|cpu |slow)|AUTOSCRIPT" $log.txt | cut -c1-200
}
