scenario_name="multiplayer (two processes over TCP with the d2gs protocol: join, see each other, walk, cast, chat, leave)"
# The runner starts one process per scenario (the host, with $save). This scenario starts the joiner itself from
# scenario_env (it retries until the host listens) with a second character, writes its own log to $tmp/96-join.log
# and checks both logs. Both use OD2_PROTO=d2gs (Huffman compressed D2GS packets) on 127.0.0.1:$OD2_PORT.
# Timeline (script seconds; the joiner's clock starts a few seconds after the host's):
#   joiner: 10 look, walk 6 tiles east, 20 look, cast, chat, 36 look, leave
#   host:    7 look (joiner still at the start), 26 look (joiner arrived), chat, 61 look (joiner gone), leave
mp_second_save="$HOME/git/d2s-test/Maricon.d2s"
mp_join_log=$tmp/96-join.log

scenario_env() {
  if [ ! -f "$mp_second_save" ]; then
    echo 'echo "multiplayer: second character missing, only the host runs"'
  else
    local jsave=$tmp/join.d2s jcmd=$tmp/96-join.command
    cp "$mp_second_save" $jsave
    {
      echo '#!/bin/zsh'
      echo "export OD2_PORT=$OD2_PORT OD2_PROTO=d2gs OD2_JOIN=127.0.0.1:$OD2_PORT OD2_JOIN_RETRY=120"
      echo "export OD2_AUTOGAME=\"$jsave\" OD2_AUTOTEST_MUTE=1 OD2_AUTOEXIT=1 OD2_D2S_WRITEBACK=$tmp"
      echo "export OD2_AUTOSCRIPT='wait:10;say:players;move:126,117;wait:10;say:players;cast:Fire Bolt@128,117;wait:2;say:chat hello_from_joiner;wait:14;say:players;exit'"
      echo "$tmp/od2 2>&1 | tee $mp_join_log"
    } > $jcmd
    chmod +x $jcmd; rm -f $mp_join_log
    launch_game $jcmd
  fi
  echo "export OD2_PROTO=d2gs OD2_HOST=1 OD2_BIND=127.0.0.1 OD2_D2S_WRITEBACK=$tmp"
  echo "export OD2_AUTOSCRIPT='wait:7;say:players;wait:19;say:players;say:chat hello_from_host;wait:35;say:players;exit'"
}

scenario_check() {
  local j=$mp_join_log.txt pat="PLAYER (JOIN|ADD|LEAVE|CAST)|PLAYERS n=|CHAT|GAMEINFO|MAP generated|AUTOSCRIPT RESULT"
  [ -f "$mp_join_log" ] && sed 's/\x1b\[[0-9;]*m//g' $mp_join_log > $j
  if [ ! -f "$j" ]; then echo "FAIL: no joiner log (second character missing?)"; fail=1; return; fi
  grep -E "$pat" $log.txt | sed 's/^/host   /' | cut -c1-200
  grep -E "$pat" $j | sed 's/^/joiner /' | cut -c1-200

  grep -q "proto=d2gs" $log.txt || { echo "FAIL: host did not run the d2gs protocol"; fail=1; }
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: host script did not pass"; fail=1; }
  grep -q "AUTOSCRIPT RESULT PASS" $j || { echo "FAIL: joiner script did not pass"; fail=1; }

  # both joined: the host's server saw two different heroes, each client knows the other
  local hnq jnq hn jn
  hnq=$(grep -o 'PLAYER JOIN name="[^"]*"' $log.txt | sed -n 1p | sed 's/.*name=//')
  jnq=$(grep -o 'PLAYER JOIN name="[^"]*"' $log.txt | sed -n 2p | sed 's/.*name=//')
  hn=${hnq//\"/}; jn=${jnq//\"/}
  { [ -n "$hn" ] && [ -n "$jn" ] && [ "$hn" != "$jn" ]; } || { echo "FAIL: two different heroes did not join (host=$hn joiner=$jn)"; fail=1; }
  grep -q "PLAYER ADD name=$hnq .*local=false" $j || { echo "FAIL: joiner never saw $hn"; fail=1; }
  grep -q "PLAYER ADD name=$jnq .*local=false" $log.txt || { echo "FAIL: host never saw $jn"; fail=1; }

  # the same level: identical map fingerprint, and the host's seed reached the joiner
  local hf jf
  hf=$(grep -o 'fingerprint=[0-9a-f]*' $log.txt | head -1); jf=$(grep -o 'fingerprint=[0-9a-f]*' $j | head -1)
  { [ -n "$hf" ] && [ "$hf" = "$jf" ]; } || { echo "FAIL: map fingerprints differ (host $hf joiner $jf)"; fail=1; }
  grep -q "GAMEINFO host map seed=" $j || { echo "FAIL: joiner got no game info (seed/difficulty)"; fail=1; }

  # each sees the other's name at the other's position (2nd look of each: both heroes listed; the position the
  # host has for the joiner equals the joiner's own, the joiner's view of the host equals the host's own)
  local hl jl hseesj jown jseesh hown
  hl=$(grep 'PLAYERS n=' $log.txt | sed -n 2p); jl=$(grep 'PLAYERS n=' $j | sed -n 2p)
  { echo "$hl" | grep -q "$jn@" && echo "$hl" | grep -q "$hn@"; } || { echo "FAIL: host does not list both heroes: $hl"; fail=1; }
  { echo "$jl" | grep -q "$jn@" && echo "$jl" | grep -q "$hn@"; } || { echo "FAIL: joiner does not list both heroes: $jl"; fail=1; }
  hseesj=$(echo "$hl" | grep -o " $jn@([0-9,]*)" | sed 's/.*@//'); jown=$(echo "$jl" | grep -o "\*$jn@([0-9,]*)" | sed 's/.*@//')
  jseesh=$(echo "$jl" | grep -o " $hn@([0-9,]*)" | sed 's/.*@//'); hown=$(echo "$hl" | grep -o "\*$hn@([0-9,]*)" | sed 's/.*@//')
  { [ -n "$hseesj" ] && [ "$hseesj" = "$jown" ]; } || { echo "FAIL: host sees the joiner at '$hseesj', the joiner is at '$jown'"; fail=1; }
  { [ -n "$jseesh" ] && [ "$jseesh" = "$hown" ]; } || { echo "FAIL: joiner sees the host at '$jseesh', the host is at '$hown'"; fail=1; }

  # the joiner's walk appears at the host: its position differs between the host's first and second look
  local first
  first=$(grep 'PLAYERS n=' $log.txt | sed -n 1p | grep -o " $jn@([0-9,]*)" | sed 's/.*@//')
  { [ -n "$first" ] && [ "$first" != "$hseesj" ]; } || { echo "FAIL: the joiner's walk did not show at the host ('$first' -> '$hseesj')"; fail=1; }

  # the joiner's cast and chat reach the host; the host's chat reaches the joiner
  grep -q "PLAYER CAST name=$jnq" $log.txt || { echo "FAIL: the host did not see the joiner cast"; fail=1; }
  grep -q "CHAT <$jn> hello from joiner" $log.txt || { echo "FAIL: host did not get the joiner's chat"; fail=1; }
  grep -q "CHAT <$hn> hello from host" $j || { echo "FAIL: joiner did not get the host's chat"; fail=1; }

  # clean leave: the joiner left first, the host saw it and lists only itself afterwards
  grep -q "PLAYER LEAVE name=$jnq" $log.txt || { echo "FAIL: host did not see the joiner leave"; fail=1; }
  grep 'PLAYERS n=' $log.txt | tail -1 | grep -q 'n=1 ' || { echo "FAIL: host still lists the joiner after it left"; fail=1; }
  if grep -E "\[(ERROR|WARNING)\]|panic" $j | grep -v "skipping missing"; then echo "FAIL: errors in the joiner log"; fail=1; fi
}
