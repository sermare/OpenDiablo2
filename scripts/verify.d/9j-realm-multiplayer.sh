scenario_realtime=1
scenario_name="realm multiplayer (two game windows through the menus: host / join, walk, chat, cast, kill sync, equal worlds by digest)"
# The game screen plays on the authoritative realm (package d2realm, simulation d2mp): the host process runs the
# realm inside itself, the joiner dials it. Both processes are started at the main menu and driven by OD2_AUTOFLOW
# (Multiplayer -> TCP/IP -> Host Game / Join Game -> pick the hero -> play), so the menu wiring is part of the test.
# Each has its own config folder and save folder; nothing real is written. The script steps are in lock step through
# waitlog: chat lines (h_checked, j_killed ...), so the checks below compare numbers taken at the same moments:
#
#   CP1  both heroes standing, 3 training dummies alive          (digest of the simulation as each client sees it)
#   CP2  the joiner killed a dummy                               (1 dead)
#   CP3  the host killed another dummy                           (2 dead)
#   CP4  the joiner has left                                     (host alone)
#
# The line "REALM WORLD level=1 heroes=2 monsters_alive=3 monsters_dead=0 digest=<hash of every unit's id, kind,
# type, life, state and final position>" is printed by the game command mpworld. Equal digests at a checkpoint mean
# both clients hold the same world, to the sub-tile.
jsave_src="$HOME/git/d2s-test/Maricon.d2s"
hflow=$tmp/9j-host
jflow=$tmp/9j-join
mp_join_log=$tmp/9j-join.log

d2s_name() { dd if="$1" bs=1 skip=20 count=16 2>/dev/null | tr -d '\0'; }

scenario_env() {
  host_name=$(d2s_name "$D2S_SAMPLE_BODY")
  join_name=$(d2s_name "$jsave_src")
  mkdir -p $hflow/saves $hflow/cfg $jflow/saves $jflow/cfg
  cp "$D2S_SAMPLE_BODY" $hflow/saves/Host.d2s
  cp "$HOME/Library/Application Support/OpenDiablo2/config.json" $hflow/cfg/ 2>/dev/null
  if [ ! -f "$jsave_src" ]; then
    echo 'echo "realm multiplayer: second character missing, only the host runs"'
  else
    cp "$jsave_src" $jflow/saves/Join.d2s
    cp "$HOME/Library/Application Support/OpenDiablo2/config.json" $jflow/cfg/ 2>/dev/null
    local jcmd=$tmp/9j-join.command
    {
      echo '#!/bin/zsh'
      echo "export OD2_PORT=$OD2_PORT OD2_JOIN_RETRY=150 OD2_CONFIG_DIR=\"$jflow/cfg\" OD2_D2S_DIR=\"$jflow/saves\""
      echo "export ${OD2_VERIFY_MUTE_ENV} OD2_AUTOEXIT=1"
      echo "export OD2_AUTOFLOW=\"join:127.0.0.1,select:$join_name,play,difficulty:0\""
      # joiner: sees the host, walks east (and casts north, away from the dummies), tells the host it is there,
      # kills the nearest dummy once the host has looked, and leaves after the host's kill
      local js='waitlog:local=false;say:players;wait:1;move:126,117;wait:6;say:players;say:chat hello_from_joiner'
      js+=';cast:Fire Bolt@126,100;wait:2;say:mpworld;say:chat j_arrived'
      js+=';waitlog:h_checked;say:mpkill;until:by="'$join_name'",45;wait:1;say:mpworld;say:chat j_killed'
      js+=';waitlog:h_killed;wait:1;say:mpworld;say:chat j_done;wait:1;exit'
      echo "export OD2_AUTOSCRIPT='$js'"
      echo "$tmp/od2 2>&1 | tee $mp_join_log"
    } > $jcmd
    chmod +x $jcmd; rm -f $mp_join_log
    launch_game $jcmd
  fi
  echo "unset OD2_AUTOGAME OD2_D2S_WRITEBACK"
  echo "export OD2_CONFIG_DIR=\"$hflow/cfg\" OD2_D2S_DIR=\"$hflow/saves\" OD2_BIND=127.0.0.1"
  echo "export OD2_AUTOFLOW=\"host,select:$host_name,play,difficulty:0\""
  local hs='waitlog:local=false;say:players;say:chat hello_from_host'
  hs+=';waitlog:j_arrived;wait:1;say:players;say:mpworld;say:chat h_checked'
  hs+=';waitlog:j_killed;say:mpworld;say:mpkill;until:by="'$host_name'",45;wait:1;say:mpworld;say:chat h_killed'
  hs+=';waitlog:j_done;wait:4;say:players;say:mpworld;exit'
  echo "export OD2_AUTOSCRIPT='$hs'"
}

scenario_check() {
  local j=$mp_join_log.txt pat="PLAYER (JOIN|ADD|LEAVE|CAST)|PLAYERS n=|CHAT|REALM (WORLD|KILL|MONSTER)|AUTOSCRIPT RESULT"
  [ -f "$mp_join_log" ] && sed 's/\x1b\[[0-9;]*m//g' $mp_join_log > $j
  if [ ! -f "$j" ]; then echo "FAIL: no joiner log (second character missing?)"; fail=1; return; fi
  grep -E "$pat" $log.txt | sed 's/^/host   /' | cut -c1-230
  grep -E "$pat" $j | sed 's/^/joiner /' | cut -c1-230

  # both ran the realm, from the menus
  grep -q "proto=realm" $log.txt || { echo "FAIL: host did not play on the realm"; fail=1; }
  grep -q "proto=realm" $j || { echo "FAIL: joiner did not play on the realm"; fail=1; }
  grep -q "AUTOFLOW step [0-9]* host" $log.txt || { echo "FAIL: host did not come through the Host Game menu"; fail=1; }
  grep -q "AUTOFLOW step [0-9]* join:127.0.0.1" $j || { echo "FAIL: joiner did not come through the Join Game menu"; fail=1; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: host script did not pass"; fail=1; }
  grep -q "AUTOSCRIPT RESULT PASS" $j || { echo "FAIL: joiner script did not pass"; fail=1; }

  # two different heroes, each seen by the other
  local hn jn
  hn=$(grep -o 'PLAYER JOIN name="[^"]*"' $log.txt | head -1 | sed 's/.*name="//; s/"$//')
  jn=$(grep -o 'PLAYER JOIN name="[^"]*"' $j | head -1 | sed 's/.*name="//; s/"$//')
  { [ -n "$hn" ] && [ -n "$jn" ] && [ "$hn" != "$jn" ]; } || { echo "FAIL: two different heroes did not join (host=$hn joiner=$jn)"; fail=1; }
  grep -q "PLAYER ADD name=\"$hn\" .*local=false" $j || { echo "FAIL: joiner never saw $hn"; fail=1; }
  grep -q "PLAYER ADD name=\"$jn\" .*local=false" $log.txt || { echo "FAIL: host never saw $jn"; fail=1; }

  # the same level
  local hf jf
  hf=$(grep -o 'fingerprint=[0-9a-f]*' $log.txt | head -1); jf=$(grep -o 'fingerprint=[0-9a-f]*' $j | head -1)
  { [ -n "$hf" ] && [ "$hf" = "$jf" ]; } || { echo "FAIL: map fingerprints differ (host $hf joiner $jf)"; fail=1; }

  # move sync: the second look of each lists both heroes, and each sees the other where the other stands
  local hl jl hseesj jown jseesh hown first
  hl=$(grep 'PLAYERS n=' $log.txt | sed -n 2p); jl=$(grep 'PLAYERS n=' $j | sed -n 2p)
  { echo "$hl" | grep -q "$jn@" && echo "$hl" | grep -q "$hn@"; } || { echo "FAIL: host does not list both heroes: $hl"; fail=1; }
  { echo "$jl" | grep -q "$jn@" && echo "$jl" | grep -q "$hn@"; } || { echo "FAIL: joiner does not list both heroes: $jl"; fail=1; }
  hseesj=$(echo "$hl" | grep -o " $jn@([0-9,]*)" | sed 's/.*@//'); jown=$(echo "$jl" | grep -o "\*$jn@([0-9,]*)" | sed 's/.*@//')
  jseesh=$(echo "$jl" | grep -o " $hn@([0-9,]*)" | sed 's/.*@//'); hown=$(echo "$hl" | grep -o "\*$hn@([0-9,]*)" | sed 's/.*@//')
  { [ -n "$hseesj" ] && [ "$hseesj" = "$jown" ]; } || { echo "FAIL: host sees the joiner at '$hseesj', the joiner is at '$jown'"; fail=1; }
  { [ -n "$jseesh" ] && [ "$jseesh" = "$hown" ]; } || { echo "FAIL: joiner sees the host at '$jseesh', the host is at '$hown'"; fail=1; }
  first=$(grep 'PLAYERS n=' $log.txt | sed -n 1p | grep -o " $jn@([0-9,]*)" | sed 's/.*@//')
  { [ -n "$first" ] && [ "$first" != "$hseesj" ]; } || { echo "FAIL: the joiner's walk did not show at the host ('$first' -> '$hseesj')"; fail=1; }

  # chat both ways, and the joiner's cast at the host
  grep -q "CHAT <$jn> hello from joiner" $log.txt || { echo "FAIL: host did not get the joiner's chat"; fail=1; }
  grep -q "CHAT <$hn> hello from host" $j || { echo "FAIL: joiner did not get the host's chat"; fail=1; }
  grep -q "PLAYER CAST name=\"$jn\"" $log.txt || { echo "FAIL: the host did not see the joiner cast"; fail=1; }

  # kill sync: both logs hold the same two kills, one by each hero
  local hk jk
  hk=$(grep -o 'REALM KILL unit=[0-9]* name="[^"]*" by="[^"]*"' $log.txt | sed 's/ name="[^"]*"//' | sort)
  jk=$(grep -o 'REALM KILL unit=[0-9]* name="[^"]*" by="[^"]*"' $j | sed 's/ name="[^"]*"//' | sort)
  { [ "$(echo "$hk" | grep -c .)" = 2 ] && [ "$hk" = "$jk" ]; } || { echo "FAIL: the kills differ (host: $hk | joiner: $jk)"; fail=1; }
  echo "$hk" | grep -q "by=\"$jn\"" || { echo "FAIL: no kill by the joiner"; fail=1; }
  echo "$hk" | grep -q "by=\"$hn\"" || { echo "FAIL: no kill by the host"; fail=1; }

  # equal worlds at the checkpoints (numbers taken at the same moments, see the header)
  local i hw jw hd jd
  for i in 1 2 3; do
    hw=$(grep 'REALM WORLD' $log.txt | sed -n ${i}p); jw=$(grep 'REALM WORLD' $j | sed -n ${i}p)
    hd=$(echo "$hw" | grep -o 'digest=[0-9a-f]*'); jd=$(echo "$jw" | grep -o 'digest=[0-9a-f]*')
    { [ -n "$hd" ] && [ "$hd" = "$jd" ]; } || { echo "FAIL: CP$i digests differ (host $hd, joiner $jd)"; fail=1; }
    echo "$hw" | grep -q "heroes=2 monsters_alive=$((4 - i)) monsters_dead=$((i - 1))" || { echo "FAIL: CP$i host world: $hw"; fail=1; }
    echo "$jw" | grep -q "engine_monsters_alive=$((4 - i)) engine_monsters_dead=$((i - 1))" || { echo "FAIL: CP$i joiner engine monsters: $jw"; fail=1; }
  done
  grep 'REALM WORLD' $log.txt | sed -n 4p | grep -q "heroes=1 monsters_alive=1 monsters_dead=2" || { echo "FAIL: CP4: the host does not see itself alone"; fail=1; }

  # clean leave
  grep -q "PLAYER LEAVE name=\"$jn\"" $log.txt || { echo "FAIL: host did not see the joiner leave"; fail=1; }
  grep 'PLAYERS n=' $log.txt | tail -1 | grep -q 'n=1 ' || { echo "FAIL: host still lists the joiner after it left"; fail=1; }
  if grep -E "\[(ERROR|WARNING)\]|panic" $j | grep -v "skipping missing"; then echo "FAIL: errors in the joiner log"; fail=1; fi
}
