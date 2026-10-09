scenario_name="UI parity (gamma/contrast applied to rendering, automap options read from config, quest log tabs for all five acts; screenshots)"
cfg=$tmp/ui-parity-config
shots=$tmp/ui-parity
scenario_env() {
  mkdir -p $cfg
  # an isolated copy of the real config (it holds the MPQ path) with gamma 8, contrast 7, automap fade on,
  # centre-when-cleared off, party markers on, names off
  python3 -c '
import json, sys
c = json.load(open(sys.argv[1]))
c["Options"] = dict(c.get("Options") or {}, gamma=8, contrast=7, automapfade=0, automapcenter=1, automapparty=0, automapnames=1)
json.dump(c, open(sys.argv[2], "w"), indent=2)' "$HOME/Library/Application Support/OpenDiablo2/config.json" $cfg/config.json
  echo "export OD2_CONFIG_DIR=\"$cfg\""
  echo "export OD2_AUTOSCRIPT='wait:2;say:capframe $shots-plain.png;automap:full;wait:1;automap:stats;say:capframe $shots-automap.png;automap:off;panel:quest1;wait:1;say:capframe $shots-quest1.png;panel:quest2;wait:1;say:capframe $shots-quest2.png;panel:quest3;wait:1;panel:quest4;wait:1;panel:quest5;wait:1;say:capframe $shots-quest5.png;panel:close;exit'"
}
scenario_check() {
  grep -E "OPTIONS video|AUTOMAP options|PANEL quest" $log.txt | cut -c1-300
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: ui-parity script did not pass"; fail=1; }
  grep -q "OPTIONS video gamma step=8 contrast step=7 applied to rendering" $log.txt || { echo "FAIL: gamma/contrast not applied at start"; fail=1; }
  grep -q "AUTOMAP options fade=true center=false party=true names=false" $log.txt || { echo "FAIL: automap options not read from config"; fail=1; }
  for a in 1 2 3 4 5; do
    grep -qE "PANEL quest: act=$a tabs=5 quests=[0-9]+ " $log.txt || { echo "FAIL: no quest log page for act $a"; fail=1; }
  done
  grep -qE 'PANEL quest: act=1 .* selected=[1-6] title="[^"]+" text="[^"]+"' $log.txt || { echo "FAIL: the Act 1 page shows no quest title and text"; fail=1; }
  for s in plain automap quest1 quest2 quest5; do
    [ -s $shots-$s.png ] || { echo "FAIL: no screenshot $s"; fail=1; }
  done
  echo "screenshots: $shots-*.png"
}
