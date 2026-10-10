scenario_name="quest rewards through the NPCs (Akara reset row, Kashya/Qual-Kehk hire, Charsi imbue, Larzuk sockets, Anya personalise, Malah, travel rows, Cain identify): menu rows, speech, cursor-item drop on the NPC"
scenario_warnings_ok=1
# The rewards that used to work only through console commands, played through the NPC menus (OD2_AUTOSCRIPT):
# `say:questpending <act> <quest>` puts a quest into the state after its kill (reward pending), the quest giver's Talk
# row speaks the reward line and claims it, and the item rewards are claimed by clicking the NPC with an item on
# the cursor (`say:pickitem <code>` is the click on the inventory item, `move:npc=` the click on the NPC).
scenario_env() {
  [ -n "${D2S_SAMPLE_BODY:-}" ] && cp "$D2S_SAMPLE_BODY" "$save"
  local s="wait:1;say:resetquests;say:freeinv 12"
  talk() { s+=";move:npc=$1;until:NPC menu opened: npc=\"$1\",60;menu:Talk;wait:3"; }
  # Akara: the Den of Evil line gives the skill point; the reset row follows
  s+=";say:questpending 1 1"
  talk Akara; talk Akara; talk Akara
  s+=";move:npc=Akara;until:NPC menu opened: npc=\"Akara\",60;menu:Reset;wait:2"
  # Kashya: the Burial Grounds reward line, then the hire list
  s+=";say:questpending 1 2"
  talk Kashya; talk Kashya
  s+=";move:npc=Kashya;until:NPC menu opened: npc=\"Kashya\",60;menu:Hire;wait:2;menu:Cancel;wait:1"
  # Charsi: the Malus is handed in (Talk), then an item is imbued by clicking her with it
  s+=";say:giveitem hdm"
  talk Charsi; talk Charsi
  s+=";move:npc=Charsi;until:NPC menu opened: npc=\"Charsi\",60;menu:Cancel;wait:1"
  s+=";say:giveitemq lgl 2;say:pickitem lgl;move:npc=Charsi;until:REWARD imbue done,60;say:putitem"
  # the travel rows: Warriv east (Andariel done), Meshif east (Duriel done), the portal, Tyrael's talk
  s+=";say:completequest 1 6;move:npc=Warriv;until:NPC menu opened: npc=\"Warriv\",60;menu:East;until:TRAVEL act 1 -> 2,20;wait:4"
  s+=";move:npc=Atma;until:NPC menu opened: npc=\"Atma\",60;menu:Talk;wait:3"
  s+=";say:completequest 2 6;move:npc=Meshif;until:NPC menu opened: npc=\"Meshif\",60;menu:East;until:TRAVEL act 2 -> 3,20;wait:4"
  s+=";move:npc=Hratli;until:NPC menu opened: npc=\"Hratli\",60;menu:Talk;wait:3"
  s+=";say:completequest 3 6;travel:4;wait:4"
  s+=";say:completequest 4 2;move:npc=Tyrael;until:NPC menu opened: npc=\"Tyrael\",60;menu:Talk;wait:3;travel:5;until:TRAVEL act 4 -> 5,20;wait:4"
  # Harrogath: Anya and Deckard Cain are not placed by the town map (the quests bring them), stand them next to the hero
  s+=";say:spawnmon drehya;say:spawnmon cain6"
  # Larzuk: sockets
  s+=";say:questpending 5 1"
  talk Larzuk; talk Larzuk
  s+=";say:giveitemq lsd 2;say:pickitem lsd;move:npc=Larzuk;until:REWARD socket done,60;say:putitem"
  # Qual-Kehk: refused before the quest is done, hirable after its reward line
  s+=";move:npc=Qual-Kehk;until:NPC menu opened: npc=\"Qual-Kehk\",60;menu:Hire;wait:1;menu:Cancel;wait:1"
  s+=";say:questpending 5 2"
  talk Qual-Kehk; talk Qual-Kehk
  # Anya: personalisation
  s+=";say:questpending 5 4"
  talk Anya; talk Anya
  s+=";say:giveitemq skp 4;say:pickitem skp;move:npc=Anya;until:REWARD personalize done,60;say:putitem"
  # Malah (Prison of Ice reward line) and Nihlathak
  s+=";say:questpending 5 3"
  talk Malah; talk Malah
  talk Nihlathak
  # Cain: the identify window
  s+=";move:npc=Deckard Cain;until:NPC menu opened: npc=\"Deckard Cain\",60;menu:Identify;wait:2;exit"
  echo "export OD2_AUTOSCRIPT='$s'"
}
scenario_check() {
  grep -E "AUTOSCRIPT RESULT|AUTOSCRIPT step [0-9]+ FAIL|REWARD |QUEST EFFECT|NPC menu opened|MERC offers|TRAVEL act" $log.txt | cut -c1-220
  grep -q "AUTOSCRIPT RESULT PASS" $log.txt || { echo "FAIL: the reward NPC script did not pass"; fail=1; }
  chk() { grep -qE "$1" $log.txt || { echo "FAIL: $2"; fail=1; }; }
  chk 'QUEST EFFECT skill-point \+1' "Akara's skill point"
  chk 'NPC menu opened: npc="Akara" .*[Rr]eset' "Akara's reset row"
  chk 'REWARD respec by Akara' "Akara's reset"
  chk 'QUEST EFFECT hire-rogues' "Kashya's reward"
  chk 'MERC offers seller=150' "Kashya's hire list"
  chk 'QUEST EFFECT imbue-available' "Charsi's imbue was not offered"
  chk 'NPC menu opened: npc="Charsi" .*[Ii]mbue' "Charsi's imbue row"
  chk 'REWARD imbue item=lgl' "Charsi did not imbue"
  chk 'TRAVEL act 1 -> 2 via=npc' "Warriv's row"
  chk 'TRAVEL act 2 -> 3 via=npc' "Meshif's row"
  chk 'TRAVEL act 4 -> 5 via=talk' "Tyrael's talk"
  chk 'NPC menu opened: npc="Larzuk" .*Add Sockets' "Larzuk's sockets row"
  chk 'REWARD socket item=lsd sockets=[1-6]' "Larzuk did not socket the sword"
  chk 'MERC hire refused: quest gate of seller 515|NPC menu opened: npc="Qual-Kehk"' "Qual-Kehk"
  chk 'QUEST EFFECT reward hire-barbarians' "Qual-Kehk's reward"
  chk 'QUEST EFFECT reward personalize' "Anya's reward"
  chk 'REWARD personalize item=skp name=' "Anya did not personalise"
  chk 'QUEST SPEECH npc="Larzuk"' "Larzuk's speech"
  chk 'NPC menu opened: npc="Deckard Cain" .*[Ii]dentify' "Cain's menu"
  chk 'identify window opened' "Cain's identify window"
}
