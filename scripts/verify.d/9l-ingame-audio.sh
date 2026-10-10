scenario_realtime=1
scenario_unmuted=1
scenario_timeout=240
scenario_name="in-game audio (real Blood Moor: music, ambience, footsteps, skill, hit and monster sounds start, have volume and reach the device)"
# Numeric audit, no audio files: the AUDIOSTAT line printed at exit counts, per category, the channels that
# reached a player (started), the loudest volume given (vol, 0..1, after master volume and distance), the bytes
# the audio device pulled (flow) and the loudest sample among them (amp, 0..32767).
scenario_env() {
  echo 'export OD2_REALMAPS=1 OD2_AUTOLEVEL=2 OD2_SOUNDLOG=1'
  echo 'export OD2_AUTOMONSTER="zombie1,2" OD2_AUTOMONSTER_SECONDS=25 OD2_AUTOMONSTER_DIFF=0 OD2_AUTOCAST="Fire Bolt,4"'
}
# audio_field <category> <started|vol|flow|amp> prints one number of the AUDIOSTAT line (0 when absent)
audio_field() {
  local n=$(grep -a "AUDIOSTAT" $log.txt | tail -1 | sed -n "s/.* $1=\([0-9]*\)(vol \([0-9.]*\) flow \([0-9]*\)B amp \([0-9]*\)).*/\1 \2 \3 \4/p")
  case $2 in started) echo ${${=n}[1]:-0};; vol) echo ${${=n}[2]:-0};; flow) echo ${${=n}[3]:-0};; amp) echo ${${=n}[4]:-0};; esac
}
scenario_check() {
  grep -a "AUDIOSTAT" $log.txt | cut -c1-600
  grep -aq "AUDIOSTAT" $log.txt || { echo "FAIL: no AUDIOSTAT line"; fail=1; return; }
  local c
  for c in music ambience footstep skill hit monster; do
    echo "AUDIO $c started=$(audio_field $c started) vol=$(audio_field $c vol) flow=$(audio_field $c flow) amp=$(audio_field $c amp)"
    [ "$(audio_field $c started)" -ge 1 ] || { echo "FAIL: no $c sound started"; fail=1; }
    [ "$(audio_field $c vol)" != "0" ] && [ "$(audio_field $c vol)" != "0.000" ] || { echo "FAIL: $c sounds had zero volume"; fail=1; }
    [ "$(audio_field $c flow)" -gt 0 ] || { echo "FAIL: the audio device pulled no $c data"; fail=1; }
    [ "$(audio_field $c amp)" -gt 0 ] || { echo "FAIL: $c data was digital silence"; fail=1; }
  done
  grep -aqE "AUDIOSTAT music_master=(0\.[1-9]|1)" $log.txt || { echo "FAIL: music master volume is zero"; fail=1; }
  grep -aqE "AUDIOSTAT .*sfx_master=(0\.[1-9]|1)" $log.txt || { echo "FAIL: sound master volume is zero"; fail=1; }
  grep -aq "AMBIENT env change env=ESOUNDENVIRON_WILDERNESS" $log.txt || { echo "FAIL: the Blood Moor sound environment was not selected"; fail=1; }
  grep -a "could not load sound" $log.txt && { echo "FAIL: sounds that could not be loaded"; fail=1; }
}
