# Shared by the a1-display-* scenarios (docs/DISPLAY.md): boot the game at a given logical size (OD2_DISPLAY=WxH),
# take a screenshot, click at the far right and far left of the world and check
#   - the logged display model (size, column origin: centred, standing on the bottom edge; bottom bar span),
#   - the screenshot's pixel size (the world fills the whole W x H),
#   - that world picking at the screen edges is exact (the picked world point is where the screen offset says),
#   - that the hero walks to the right, then to the left, on screen.
# display_env W H     prints the scenario_env lines;  display_check W H  is the scenario_check.
display_env() {
  local W=$1 H=$2 d=$tmp/disp$1x$2
  mkdir -p $d/s94 $d/wb94; rm -f $d/s94/*.d2s(N) $d/wb94/*.d2s(N)
  if [ -n "${D2_TABLES:-}" ] && go run scripts/d2s-revive.go "$D2S_SAMPLE_BODY" $d/s94/Hero.d2s "$D2_TABLES" >/dev/null 2>&1; then
    echo "export OD2_AUTOGAME=\"$d/s94/Hero.d2s\" OD2_D2S_WRITEBACK=\"$d/wb94\" OD2_AUTOSPEED=3"
  fi
  echo "export OD2_DISPLAY=${W}x${H}"
  local s="wait:3;say:heropos;say:capframe $d/shot.png;wait:1"
  s="$s;skill:left=Attack;click:left@$((W-30)),$((H/2-80));wait:6;say:heropos"   # far right of the world
  s="$s;click:left@30,$((H/2-80));wait:10;say:heropos;say:capframe $d/shot2.png;wait:1;exit"        # far left
  echo "export OD2_AUTOSCRIPT='$s'"
}

display_check() {
  local W=$1 H=$2 d=$tmp/disp$1x$2
  local ox=$(( (W-800)/2 )) oy=$(( H-600 ))
  grep -E "DISPLAY size|world-pick|HERO pos|AUTOSCRIPT RESULT" $log.txt | cut -c1-220 | head -12
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: display script did not pass"; fail=1; }
  grep -q "DISPLAY size=${W}x${H} scale=1 column=($ox,$oy) hud_x=$ox..$((ox+800)) hud_bottom=$H " $log.txt \
    || { echo "FAIL: display model line not as expected (want size=${W}x${H} column=($ox,$oy) hud_x=$ox..$((ox+800)) hud_bottom=$H)"; fail=1; }
  for f in $d/shot.png $d/shot2.png; do
    if [ ! -s $f ]; then echo "FAIL: no screenshot $f"; fail=1; continue; fi
    dims=$(python3 -I -c 'import struct,sys; b=open(sys.argv[1],"rb").read(24); print(*struct.unpack(">II", b[16:24]))' $f)
    [ "$dims" = "$W $H" ] || { echo "FAIL: screenshot $f is $dims, want $W $H"; fail=1; }
  done
  # picking: screen -> world agrees with the ortho offset from the hero, at both edges
  tr -d '\r' < $log.txt | grep "INPUT world-pick" | sed -E 's/.*world=\(([-0-9.]+),([-0-9.]+)\) hero=\(([-0-9.]+),([-0-9.]+)\) hero_screen=\(([-0-9]+),([-0-9]+)\).*/\1 \2 \3 \4 \5 \6/' > $d/picks.txt
  n=$(grep -c . $d/picks.txt)
  [ "$n" -ge 2 ] || { echo "FAIL: want 2 world-pick lines, got $n"; fail=1; return; }
  awk -v W=$W -v H=$H 'NR==1 { want=(W-30) } NR==2 { want=30 }
    { dx=(($1-$2)-($3-$4))*80; got=want-$5; d=dx-got; if (d<0) d=-d; printf "pick %d: screen x %d, hero screen x %d, ortho dx %.0f want %d (diff %.0f)\n", NR, want, $5, dx, got, d; if (d>40) bad=1 }
    END { exit bad }' $d/picks.txt || { echo "FAIL: picking at the screen edge does not match the camera"; fail=1; }
  # the hero walked right, then left (ortho x = (wx - wy) * 80; positions from the heropos lines)
  pos=("${(@f)$(grep -o 'HERO pos=([-0-9.]*,[-0-9.]*)' $log.txt | tr -d '\r' | sed 's/HERO pos=(//; s/)//')}")
  if [ ${#pos} -ne 3 ]; then echo "FAIL: want 3 HERO pos lines, got ${#pos}"; fail=1; else
    ox1=$(echo "$pos[1] $pos[2] $pos[3]" | awk '{for(i=1;i<=3;i++){split($i,a,","); o[i]=(a[1]-a[2])*80} printf "%.0f %.0f %.0f", o[1], o[2], o[3]}')
    set -- ${=ox1}
    [ $2 -gt $(( $1 + 40 )) ] || { echo "FAIL: the hero did not walk right on screen (ortho x $1 -> $2)"; fail=1; }
    [ $3 -lt $(( $2 - 40 )) ] || { echo "FAIL: the hero did not walk left on screen (ortho x $2 -> $3)"; fail=1; }
  fi
}
