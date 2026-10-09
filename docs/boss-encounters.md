# Boss encounters (Game.exe 1.14b, read-only Ghidra)

V = verified in the decompilation, U = unverified/interpreted. Code: `d2common/d2boss` (triggers),
`d2common/d2monster/ai_baal.go` and `ai_shaman.go` (AIs), `d2common/d2quest/boss.go` (kill -> quest bits),
`d2game/d2gamescreen/autoboss.go` (`OD2_AUTOBOSS`), scenario `scripts/verify.d/9c-bosses.sh`.

## Numbering facts (V)
- The exe's monster class equals the monstats.txt row for rows before the "Expansion" separator (about row 410); after it the exe
  class is row-1. Proof: the Ancient AI (0x5ee290) tests classes 0x21c..0x21e, which are rows 541..543 "Ancient Barbarian 1..3".
  So BaalThrone morphs into exe class 0x22f = 559 = file row 560 "Baal Crab to Stairs".
- In this engine `stat.ID` follows the exe numbering (observed in the OD2_AUTOBOSS log): Duriel 211, Mephisto 242, Diablo 243,
  Storm Caster 306 (key fingermage3), Oblivion Knight 312 (doomknight3), Venom Lord 362 (megademon3), Warped Shaman 62
  (fallenshaman5), baalthrone 543, baalcrab 544, baalcrabstairs 559.
- Objects (objects.txt Id): 152 orifice (OperateFn 25), 100 Duriel's Lair portal (43), 341 mephisto bridge (4), 342 hellgate (46),
  seals 392..396 (OperateFn 54, 52, 55, 52, 56), 255 diablo start point, 563 "The Worldstone Chamber" portal (70).
- Skills: 284 Baal Taunt, 285 Baal Corpse Explode, 286 Baal Monster Spawn, 315 Tentacle, 316 Nova, 317 Inferno, 318 Cold Missiles.
- Quest id / record slot: Seven Tombs 13/14, Guardian 20/22, Terror's End 23/26, Eve of Destruction 36/40.

## Baal throne (BaalThrone 0x5ee400, V)
AiGeneral: +0x14 wave counter, +0x18 flags (bit0 announced, bit1 sent), +0x1c next frame. With no hero within 0x40: flag0 clear ->
announce (counter < 5), cast skill 285 on itself, flag0, next = frame + 250; flag0 set -> counter > 4: morph into class 0x22f
with stat list 0x8e and wait 5; otherwise FUN_005ee2f0 casts 286 at throne + (0,13), counter++, next = frame + 100, flag1 set,
flag0 clear. A hero within 0x40 gets Skill1 with aip1%, else the throne waits 10.
Waves = SuperUniques "Baal Subject 1..5" (rows 62..66): classes 62, 105, 121, 122, 135, groups 5, 3, 5, 8, 5 (V from the table;
that skill 286 uses exactly these is U). The engine gates the next wave on the previous being dead (U: the exe think function is
purely timed).
- BaalToStairs 0x5ee720 (V): scan radius 25 for object 563, none -> wait 25; within aip1 -> state 0x92 and the unit leaves the level;
  otherwise walk to it.
- BaalTaunt 0x5ee810 (V flow, U for the pull): hero idle count > aip2 -> cast 284 (mode 4) at him; hero farther than aip3 -> moved
  by SERVER_MoveUnitToLevelPosition; farther than aip1 -> walk; sleep 25.
- BaalTentacle 0x5ee920 (V): lifetime (aip3 + roll) * 25 frames, no target -> dismissed; aip1% attack A2 else wait aip2.
- BaalCrab 0x5fc200 / BaalCrabClone 0x5fc440 (AiBaal.cpp, V weights): three 16-entry weight tables (reach 0x5fb380, engaged 0x5fb630,
  idle 0x5fb4d0): 1 wait (35/15/5 frames by difficulty), 2/6 walk to 12, 3 circle, 4 wander 16, 5 home, 8 buff, 9 tentacles (315),
  10 melee A2, 11 nova (316), 12 inferno (317), 13 cold missiles (318), 14 teleport, 15 clone (at most 2, a third of life and mana);
  then a 25-frame sleep. The inputs of unread helpers (threat score FUN_005fac60, room test, FUN_005fb280) are BaalSituation fields (U).

## Terror's End / Chaos Sanctuary (V unless noted)
Private data is 0x4c bytes: 5 seal flags (+0xc..+0x10), three seal positions (+0x24, +0x2c, +0x34) paired with super unique ids
in the game table (+0xb28/2a/2c). Operating the seal at one of the positions spawns that boss there (FUN_005b3360 -> FUN_00543b10
-> FUN_0054c470). Kill counter +0x48; when it is 3 and all five flags are set, FUN_005b2e60 runs once (guard +0x13): every other
living non-pet monster of level 0x6c is sent a record (FUN_005a5860/5a5630, effect U) and quest timer 1 starts (Diablo's arrival).
Diablo's kill (class 0xf3): +0x14 = 1, speech 0x40, quest done. Super uniques: Infector of Souls 36 (class 362, group 9),
Lord De Seis 37 (312, 5), Grand Vizier of Chaos 38 (306, 9). U: which plain seals belong to which boss (modelled: Vizier 392+393,
De Seis 394, Infector 395+396).

## Other kills (V)
- Duriel (0x59ac40): quest timer 8, speech 0x3d, slot 14 bit 5 set when bits 0, 3, 4, 5 are clear, orifice object mode 1.
- Mephisto (0x5ba440): speech 0x3e, QUEST_SetPlayerQuestFlag(0xd), timer 0xc, bridge object mode 1; the bridge object (0x155) handler
  0x5ba630 freezes the hero for the cutscene when the quest state is 2.
- Baal (0x58bce0): slot 0x28; spawns a class 0x271 unit at his corpse (U purpose).
- Bits beyond those listed (primary goal + reward pending for Diablo/Baal, the Andariel convention) are U.

## Pandemonium event and quest rewards (code: `d2common/d2uber`, `d2cube`, `d2reward`; scenario `9g-quest-rewards-uber.sh`)
VERIFIED from the 1.14b patch_d2 tables: monstats rows ubermephisto/uberdiablo/uberizual/uberandariel (display name Lilith)/uberduriel/
uberbaal = exe classes 704..709, level 110, AI names UberMephisto, UberDiablo, UberIzual, Andariel, Duriel, UberBaal (the last one reuses
the BaalCrab think function, U); treasure classes "Uber Andariel/Duriel/Izual" drop dhn/bey/mbr; CubeMain rows pk1+pk2+pk3 ("Pandemonium
Portal") and dhn+bey+mbr ("Pandemonium Finale Portal"); levels 133..135 (Pandemonium 1..3) and 136 (Finale). UNVERIFIED (community
knowledge, nothing in the data): which area a key set opens (the engine walks them in order, the live game picks at random), the Hell
bosses that drop the keys (Andariel pk1, Duriel pk2, Mephisto pk3), the arrival order/delays of the Tristram bosses, the Standard of
Heroes (`std`, not in the tables), the red portal object. Quest rewards: Potion of Life and Malah's scroll change the hero (LifeBonus,
ResistBonus); Larzuk's sockets and Anya's personalisation wait for an item (`rewarditem socket|personalize`); socket count and
eligibility rules are U. The quest system now follows act changes (`TravelToAct2..5` on the NPC, portal and talk trips) and the portal to
Act 4 / Act 5 opens in the Durance of Hate 3 / the Pandemonium Fortress once the quest is done.

## Other AIs
- FallenShaman 0x5f04d0 (V, re-read): melee first (aip3), corpse scan for classes 19/58 within aip4, alert broadcast aip1, resurrect
  aip1, Skill2 fire aip2 within aip5, circle. The engine's Director has no corpse finder yet, so it does not resurrect in game.
- State 13 (0x5e4be0): anchor guard, partly U. State 16 (0x5e1c60): imp after Imp Teleport (unit state 0x8f), partly U.
