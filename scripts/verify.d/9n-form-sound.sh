scenario_name="Druid sounds by mode (SOUNDLOG: human footsteps light by class, impossible voice on a refused Maul, wolf/bear A1 weapon and S3 bite sounds, form footsteps; w44)"
scenario_realtime=1
scenario_timeout=420
scenario_warnings_ok=1
# w44. A fresh Druid (OD2_AUTOCAST_CLASS, level 20, own temp save). Human: walking plays the class footstep set (druid = light,
# UNITSND_GetStepSoundId 0x4c7970) and the refused Maul (restricted, status 4) says the impossible line (0x13, table 0x7118f8).
# Wolf: walking plays the MonSounds Footstep row (kind form-footstep), Feral Rage (A1) plays the Weapon1 sound
# (weapon_kick_1, always) and Rabies (S3) the Skill3 sound druidwerewolf_bite_1 (UNITSND_PlayMonsterEventSound 0x4c8340).
# Bear: Maul (A1) plays weapon_2hs_small_1.
scenario_env() {
  local s
  s='wait:2;expect:level=2;wait:3;skill:spend=Wearwolf;skill:spend=Feral Rage;skill:spend=Rabies;skill:spend=Wearbear;skill:spend=Maul'
  s="$s;skill:left=Attack;say:heromoved;click:left@hero:140,30;wait:3"
  s="$s;skill:left=Maul;click:left@monster;wait:2"
  s="$s;skill:right=Wearwolf;click:right@hero:60,30;wait:3"
  s="$s;skill:left=Attack;click:left@hero:-140,-30;wait:3"
  s="$s;skill:left=Feral Rage;click:left@monster;wait:4;click:left@monster;wait:4"
  s="$s;skill:left=Rabies;click:left@monster;wait:4;click:left@monster;wait:4"
  s="$s;skill:right=Wearbear;click:right@hero:60,30;wait:3;click:right@hero:60,30;wait:3"
  s="$s;skill:left=Maul;click:left@monster;wait:4;click:left@monster;wait:4;click:left@monster;wait:4;exit"
  echo "export OD2_SOUNDLOG=1 OD2_AUTOCAST_MANA=400 OD2_AUTOGAME=\"$tmp/w44.od2\" OD2_AUTOCAST_CLASS=druid OD2_AUTOCAST_CLVL=20 OD2_AUTOCAST_SKILLPOINTS=8"
  echo 'export OD2_REALMAPS=1 OD2_AUTOLEVEL=2 OD2_AUTOMONSTER="zombie1,10" OD2_AUTOMONSTER_PASSIVE=1 OD2_AUTOMONSTER_HOLD=1 OD2_AUTOMONSTER_DIFF=0 OD2_AUTOMONSTER_SECONDS=300'
  echo "export OD2_AUTOSCRIPT='$s'"
}
scenario_check() {
  grep -E "SOUNDLOG .* kind=(hero-footstep|form-[a-z-]+|hero-voice) " $log.txt | sed -E 's/t=[0-9.]+ //' | cut -c1-120 | sort | uniq -c | sort -rn | head -20
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the script did not pass"; fail=1; }
  grep -qE "SOUNDLOG .* kind=hero-footstep .*handle=light_(walk|run)_[a-z]+_[0-9] " $log.txt || { echo "FAIL: no light class footstep for the human druid"; fail=1; }
  grep -qE "HEROVOICE id=0x13 handle=druid_impossible_1" $log.txt || { echo "FAIL: refused Maul did not say impossible"; fail=1; }
  grep -qE "SOUNDLOG .* kind=form-footstep .*handle=heavy_(walk|run)_[a-z]+_[0-9] " $log.txt || { echo "FAIL: no wolf footstep (MonSounds Footstep row)"; fail=1; }
  grep -qE "SOUNDLOG .* kind=form-weapon .*handle=weapon_kick_1 " $log.txt || { echo "FAIL: Feral Rage (A1) played no wolf weapon sound"; fail=1; }
  grep -qE "SOUNDLOG .* kind=form-skill .*handle=druidwerewolf_bite_[0-9] " $log.txt || { echo "FAIL: Rabies (S3) played no bite sound"; fail=1; }
  grep -qE "SOUNDLOG .* kind=form-weapon .*handle=weapon_2hs_small_1 " $log.txt || { echo "FAIL: Maul (A1) played no bear weapon sound"; fail=1; }
}
