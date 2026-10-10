scenario_name="ui layout (every panel's rectangles against the original's numbers)"
scenario_env() { echo 'export OD2_AUTOUILAYOUT=1'; }
# scripts/verify.d/ui-layout.golden holds one expectation per line: "[~N ]<panel> <name> <x> <y> <w> <h>".
# Without a prefix every number must match exactly; "~N" lets each number differ by up to N pixels (used for text: its
# anchor is the centre of the drawn string, so a different font metric moves it by a pixel); "*" skips a number.
# The run (OD2_AUTOUILAYOUT=1) opens every panel on the real game screen and logs the rectangles the interface placed.
scenario_check() {
  golden=scripts/verify.d/ui-layout.golden
  grep -q "AUTOUILAYOUT done" $log.txt || { echo "FAIL: the layout run did not finish"; fail=1; return; }
  tr -d '\r' < $log.txt | sed -n 's/.*UILAYOUT \([a-z]* [a-z_0-9.]* [0-9 -]*\)$/\1/p' > $tmp/ui-layout.got
  n=0; bad=0
  while read -r line; do
    case "$line" in "#"*|"") continue;; esac
    n=$((n+1))
    tol=0
    case "$line" in "~"*) tol=${line%% *}; tol=${tol#\~}; line=${line#* };; esac
    key=${line%% [0-9*-]*}
    got=$(grep -- "^$key " $tmp/ui-layout.got | head -1)
    if [ -z "$got" ]; then echo "FAIL: ui layout: '$key' was not logged"; fail=1; bad=$((bad+1)); continue; fi
    ok=$(awk -v w="$line" -v g="$got" -v t=$tol 'BEGIN{ nw=split(w,a," "); ng=split(g,b," "); if(nw!=ng){print 0; exit}
      for(i=3;i<=nw;i++){ if(a[i]=="*") continue; d=a[i]-b[i]; if(d<0)d=-d; if(d>t){print 0; exit} } print 1 }')
    [ "$ok" = 1 ] || { echo "FAIL: ui layout: expected '$line' (+-$tol), got '$got'"; fail=1; bad=$((bad+1)); }
  done < $golden
  echo "ui layout: $n rectangles checked, $bad deviating"
  grep -c "^" $tmp/ui-layout.got | sed 's/^/ui layout: rectangles logged by the run: /'
}
