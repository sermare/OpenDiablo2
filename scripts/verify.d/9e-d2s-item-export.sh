scenario_name=".d2s item export (buy at Akara and pick up items in town, exit: the exported .d2s holds them and re-parses)"
# Items made in the game live in the .od2 save; the exporter encodes them (d2hero.MergeContainerItems) from the
# item generator's rolls. The game logs one "D2S EXPORT new item" line per such item, found in the re-parsed file.
# Warnings are allowed: an item the format cannot hold faithfully (rare/crafted, unknown affix) is reported and left out.
scenario_warnings_ok=1
wb9e=$tmp/9e-writeback
scenario_env() {
  mkdir -p $wb9e; rm -f $wb9e/*.d2s(N)
  echo "export OD2_D2S_WRITEBACK=\"$wb9e\" OD2_AUTOTRADE_KEEP=1 OD2_AUTOTRADE_LEVEL=8 OD2_AUTOTRADE_SEED=1"
  echo "export OD2_AUTOSCRIPT='wait:2;say:dropinv cm1;say:dropinv cm1;say:autobuy Akara;wait:1;panel:close;say:spawnitem cm1 key hp3 mp3 cm1;wait:1;loot:4,10;wait:1;exit'"
}
scenario_check() {
  grep -E "AUTOTRADE (buy|keep)|GIVEITEM|LOOT|D2S EXPORT|D2S export|AUTOSCRIPT RESULT" $log.txt | cut -c1-260
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the item export script did not pass"; fail=1; }
  grep -qE "AUTOTRADE buy vendor=Akara .*err=<nil>" $log.txt || { echo "FAIL: no scripted buy at Akara"; fail=1; }
  ls $wb9e/*.d2s >/dev/null 2>&1 || { echo "FAIL: no exported .d2s in $wb9e"; fail=1; return; }

  # the bought item: its code is in the keep line, and the exported file must hold it
  local code n
  code=$(sed -nE 's/.*AUTOTRADE keep vendor=Akara code=([a-z0-9]+) .*/\1/p' $log.txt | head -1)
  [ -n "$code" ] || { echo "FAIL: the bought item was not kept"; fail=1; return; }
  grep -qE "D2S EXPORT new item code=$code .* found=true" $log.txt || { echo "FAIL: the bought $code is not in the exported .d2s"; fail=1; }

  # the last export (the one at exit) holds everything: the bought item and the picked up ones. Items the format
  # cannot hold faithfully are reported ("not written") and then are not found, never silently lost.
  local last nt nf nw
  last=$(awk '/D2S EXPORT path=/ {blk=""} {blk = blk "\n" $0} END {print blk}' $log.txt)
  nt=$(echo "$last" | grep -cE "D2S EXPORT new item .* found=true")
  nf=$(echo "$last" | grep -cE "D2S EXPORT new item .* found=false")
  nw=$(echo "$last" | grep -c "D2S export: container item .* not written")
  echo "last export: new items found=$nt missing=$nf reported_not_written=$nw"
  [ "$nt" -ge 2 ] || { echo "FAIL: expected the bought and a picked up item in the .d2s, found $nt"; fail=1; }
  [ "$nf" -eq "$nw" ] || { echo "FAIL: $nf items missing from the re-parsed .d2s but $nw reported"; fail=1; }
  echo "$last" | grep -E "D2S EXPORT new item .* simple=false" | head -3
  grep -q "exported file does not parse" $log.txt && { echo "FAIL: the exported file does not parse"; fail=1; }
  grep -q "D2S EXPORT reparse: .*checksum=ok" $log.txt || { echo "FAIL: exported .d2s did not re-parse"; fail=1; }
  grep -E "\[ERROR\]|panic" $log.txt | grep -v "skipping missing" && { echo "FAIL: errors in the log"; fail=1; }
}
