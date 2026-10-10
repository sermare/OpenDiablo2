scenario_name="ui layout (every panel's rectangles against the original's numbers)"
scenario_env() { echo 'export OD2_AUTOUILAYOUT=1'; }
scenario_check() {
  golden=scripts/verify.d/ui-layout.golden
  grep -q "AUTOUILAYOUT done" $log.txt || { echo "FAIL: the layout run did not finish"; fail=1; return; }
  sed -n 's/.*UILAYOUT \([a-z]* [a-z_0-9.]* [0-9 -]*\)$/\1/p' $log.txt > $tmp/ui-layout.got
  n=0
  while read -r line; do
    case "$line" in "#"*|"") continue;; esac
    n=$((n+1))
    grep -qx -- "$line" $tmp/ui-layout.got || { echo "FAIL: ui layout: expected '$line', got: $(grep -- "^${line%% [0-9]*} " $tmp/ui-layout.got | head -1)"; fail=1; }
  done < $golden
  echo "ui layout: $n rectangles checked"
}
