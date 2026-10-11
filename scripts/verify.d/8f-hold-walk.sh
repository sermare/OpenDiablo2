scenario_name="hold to walk (left button held in Rogue Encampment through the repeat path: the hero keeps walking, no stall > 1 s, also past NPCs; releasing stops it)"
scenario_realtime=1
scenario_timeout=240
hw=$tmp/holdwalk
# Regression for "holding the left button walks a few steps and stops" (the repeat path is GameControls.OnMouseButtonRepeat,
# driven here frame by frame by the hold: step). The cursor stays at a fixed screen spot far from the hero, who stays at the
# screen centre, so a working hold walks until the hero meets a wall. Two holds in different directions cross the camp.
scenario_env() {
  mkdir -p $hw/s94 $hw/wb94; rm -f $hw/s94/*.d2s(N) $hw/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $hw/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$hw/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$hw/wb94\""
  else
    echo "# no revived sample hero, using the default save" >&2
  fi
  # warm-up: the first walk loads the walk animations lazily (seconds of stall on a busy machine, which would eat the hold);
  # one click walk, then wait until the hero has moved and stopped (heromoved/herostill, hero_watch.go)
  local s='wait:2;skill:left=Attack;say:heromoved'
  s="$s;click:left@${HOLD_A:-hero:140,30};until:HERO moved,120;say:herostill;until:HERO still,120;say:heropos"
  s="$s;hold:4,left@${HOLD_A:-hero:140,30};wait:0.5;say:heropos"
  s="$s;hold:6,left@${HOLD_B:-hero:-200,-100};wait:0.5;say:heropos;exit"
  echo "export OD2_AUTOSCRIPT='$s'"
}
scenario_check() {
  grep -E "HOLD |HERO pos|AUTOSCRIPT RESULT" $log.txt | cut -c1-200 | head -60
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: hold-walk script did not pass"; fail=1; }
  grep -q "HERO pos=.*town=true" $log.txt || { echo "FAIL: hero not in town"; fail=1; }
  grep -q "HOLD end seconds=" $log.txt || { echo "FAIL: no hold finished"; fail=1; }
  # samples (every ~0.5 s) of each hold: total distance walked, and the longest time without moving
  grep -E "HOLD pos|HOLD start" $log.txt | tr -d '\r' | sed -E 's/.*HOLD pos t=([0-9.]+) \(([-0-9.]+),([-0-9.]+)\).*/P \1 \2 \3/; s/.*HOLD start.*/S 1/' | awk '
    /^S/ { if (seg) report(); seg++; n=0; total=0; still=0; worst=0; next }
    /^P/ { if (n>0) { d=sqrt(($3-px)^2+($4-py)^2); total+=d; if (d<0.05) { still+=$2-pt; if (still>worst) worst=still } else still=0 }
           px=$3; py=$4; pt=$2; n++ }
    function report() { printf "HOLD segment %s: samples=%d walked=%.1f longest_still=%.1fs\n", seg, n, total, worst; if (n<6 || worst>1.0) bad=1; else if (total<4) bad=1 }
    END { if (seg) report(); exit bad }' > $hw.out 2>&1
  rc=$?
  cat $hw.out
  [ $rc -eq 0 ] || { echo "FAIL: the held left button did not keep the hero walking (see HOLD segment lines)"; fail=1; }
  # releasing the button stops the hero: the position after the hold equals the one 0.5 s later is checked by heropos lines
  pos=("${(@f)$(grep -o 'HERO pos=([-0-9.]*,[-0-9.]*)' $log.txt | sed 's/HERO pos=//')}")
  [ ${#pos} -eq 3 ] || { echo "FAIL: want 3 HERO pos lines, got ${#pos}"; fail=1; }
}
