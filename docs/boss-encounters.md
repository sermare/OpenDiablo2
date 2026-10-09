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

## Other AIs
- FallenShaman 0x5f04d0 (V, re-read): melee first (aip3), corpse scan for classes 19/58 within aip4, alert broadcast aip1, resurrect
  aip1, Skill2 fire aip2 within aip5, circle. The engine's Director has no corpse finder yet, so it does not resurrect in game.
- State 13 (0x5e4be0): anchor guard, partly U. State 16 (0x5e1c60): imp after Imp Teleport (unit state 0x8f), partly U.

## Oracle audit (feat/boss-oracle)
Tests: `d2common/d2monster/boss_oracle_test.go` (real monstats via `D2_TABLES`, skipped when unset) pins Skill1-8 (name, mode,
level), aip1-8, aidel, aidist, threat, Level and the six resists per difficulty for Andariel, Duriel, Mephisto, Diablo, Summoner,
Izual, Blood Raven, Griswold, Radament and the Baal rows (throne, crab, clone, taunt, to-stairs, tentacles, minion);
`d2common/d2quest/boss_oracle_test.go` pins the monster class constants of the quest nodes to the file's hcIdx column.

Facts from the table (V = read from the file):
- Andariel aip 30/10/30/50 (N), 35/8/32/55, 35/6/34/60; Skill1 AndrialSpray, Skill2 AndyPoisonBolt. No Skill3+.
- Duriel aip1 5/5/6 (aura level), aip2 33 (Smite), aip3 50 (Jab); aip4 and aip5 are empty, so no A2 mix and no Charge ever; Skill4 Holy Freeze (NU).
- Mephisto aip1 15/20/25, aip2/3 25/33/33 (unused by the ported think); aidist 0/40/46; six skills, all mode A2.
- Diablo has no aip at all; seven skills (DiabLight SC, DiabCold S2, DiabFire S1, DiabWall S3, DiabRun seq, PrimeFirewall S3, DiabPrison S3).
- Summoner aip 85/5/63/40/120/33/5/40 (N); 93/.../100/20/8 (NM); 98/.../80/10/11 (H); Skill5 Weaken.
- Izual aip4 0/75/100 and aip5 20/5/0 vary by difficulty; one skill, Frost Nova.
- Baal: throne aip1 25, taunt aip 3/10/20, to-stairs aip1 4, tentacles aip1 70..90 / aip2 24..16 / aip3 10; crab and clone carry 7
  skills (clone: "Baal Clone Teleport" instead of "Baal Teleport"). Monster levels differ per difficulty (crab 60/75/99).
- Immunities are plain resists >= 100: Griswold Hell poison 120, Baal Minion Hell fire 120, tentacles Hell cold 110..130. No boss
  in the act list is fully immune otherwise. Resists are used as written (no difficulty penalty for non-merc monsters, verify-resist.md).
- The hcIdx column already carries the exe numbering: baalthrone 543, baalcrab 544, baalcrabstairs 559, baalclone 570.

Known divergence (not changed here to avoid touching tick.go/ai_baal.go, which feat/verify-monster-ai owns): Baal AI target
modes. Exe: Throne 2, Crab 0, Taunt 1, ToStairs 1, Tentacle 1, Clone 0. This tree: TargetOnly for all but ToStairs (None).
`TestBaalTargetModesDivergence` pins today's values and must change with that merge.

## Verified in the bosses pass (feat/verify-bosses, details in d2-re-notes/verify-bosses.md)
- VERIFIED Mephisto moat: his think (0x5f6950) has no teleport; the only teleport helper, MONAI_TryWoundedMonTeleport 0x5aedc0
  (hard-coded skill 0xb8 = MonTeleport 184, not read from the monster's skills), needs AiGeneral flag 0x20, which no MONAI_SetAiFlag
  site sets (sites set 0x1, 0x2, 0x10, 0x40). So Mephisto and the blood guards never teleport; the moat trick works.
- VERIFIED seals: object 392..396 -> flag 0..4 in order (FUN_005b3240); the bosses are spawned by the dummy object 131 in level 108
  (QUEST_OnObjectOperated -> 0x5b3360, position matched against three stored positions -> super unique id; a failed spawn retries
  after 10 frames); group size = MinGrp/MaxGrp of the super unique row (9, 5, 9). Arrival 0x5b2e60 sends a mode-0 record to every
  other living non-pet monster of level 108 (default; `Seals.LegacyNoPurge` removes it). The Diablo spawn itself is quest timer 1 (handler not traced).
- VERIFIED Baal waves: skill 286 is cast with a missile (hit function 54 -> CCMD_Func_54c470 super unique spawn); the super unique
  hcIdx come from the table 0x6e4ab8 = 61..65 = rows "Baal Subject 1..5". The think is gated: no step while a living monster that
  is not hostile to the throne (the previous wave) is within edge distance 0x40 of it (a type-1 scan, callback 0x5db5f0), plus the
  250/100 frame timers. Earlier note "hero within 0x40" was wrong.
- VERIFIED kill bits: Mephisto 0x5b9d00 slot 22 bits 0xd, 0, 0xb; Diablo 0x5b2950 slot 26 bits 0xd, 0 (+6, 7 classic); Baal 0x58bae0
  slot 40 bits 0xd, 0 and only in level 132. None sets reward pending. They are the default (`Game.LegacyBossBits` restores the old reward-pending flow). Baal's death also fires
  missile 625 at the corpse (0x58bce0, effect not identified).
- VERIFIED drops: Mephisto's kill stamps item code "mss " (soulstone) on the unit and drops it (0x557980); Mephistoq has none. The Hellforge
  handler 0x5b4190 stamps "hfh " the same way (which monster is attached: not traced).
- VERIFIED travel: Meshif (class 210) sets slot 15 (and slot 10 if open), Warriv (155) slot 7, class 367 slot 28 (expansion) in 0x5446e0.
- VERIFIED Duriel lair gate: warp into level 73 is blocked while the Seven Tombs node is active and private byte +0xb is 0 (0x59b700);
  the writer of +0xb was not found (`Tomb.LairWarpBlocked` assumes: staff placed).

## Second pass (feat/verify-diablo-hellforge, details in d2-re-notes/verify-diablo-hellforge.md)
- VERIFIED seal pairing: the boss seals 392/394/396 (OperateFn 54/55/56 = 0x5b4720/0x5b4770/0x5b4840) store the seal position plus an
  offset ((-12,-52), (-39,+33), (+32,+16)) in the quest data (+0x24, +0x2c, +0x34), create a Dummy object (131) there and run the seal
  code 0x5b3240. Operating the dummy matches its coordinates (0x5b3360) and spawns hcIdx 36 (seal 392, Infector of Souls), 37 (394, Lord
  De Seis), 38 (396, Grand Vizier of Chaos); the root table +0xb28/2a/2c is the hcIdx->row array at +0xae0 (index 36..38). The earlier
  model (392 = Vizier, 396 = Infector) was wrong. The plain seals 393/395 only set their flag. This is the default (`Seals.LegacyLayout` restores the old pairing).
- VERIFIED Diablo's arrival: the Dummy object 255 (InitFn 55 = 0x5b31a0) records itself in the quest data (+6, +8 = unit id). When all five
  flags are set and the kill counter is 3, 0x5b2e60 runs once and the quest timer callback 0x5b2830 is added with delay 1; it counts 10
  frames, then spawns Diablo (class 0xf3, 0x5b27b0: exact subtile of dummy 255, else radius 5, else 10), ORs 0x3000000 into his unit flags
  and sets +0x11. 11 frames in total (`ExeDiabloDelay`). No cutscene lock was found in this code; after the arrival the level no longer
  populates itself (0x54ca00 -> 0x5b2e40). The Terror's End node is attached to the monsters by class (0x5af8c0: Diablo 0xf3) and to the
  three seal bosses by hcIdx 36..38 (0x5a2480), so the kill counter counts those four.
- VERIFIED Hellforge: the hammer quest node (24) is attached to class 0x199 = 409 "hephasto" in the monster-create hook 0x5af8c0; its kill
  handler 0x5b4190 (node active) stamps "hfh " on the dying unit and drops it (0x557980 mode 7). The forge object 376 (InitFn 48 = 0x5b3630
  sets the mode from the quest data, OperateFn 49 = 0x5b3820 needs the hammer; both are preset level objects, not created by quest code).
  By default the quest package drops the hammer on Hephasto's kill.
- VERIFIED Duriel lair byte +0xb: written (=1) at the end of the quest timer callback 0x59b450, which the staff placement event 0x59b960
  adds; the callback animates the orifice, creates the portal object 100 at (X-13, Y+3) of the object in +0x20 and then sets +0xb. So the
  lair opens when the portal appears (default; `Tomb.LegacyLairGate` restores the staff-time gate).

## Defaults (feat/enable-exe-boss-layout)
The verified behaviours are now the default: exe seal pairing/dummy offsets/11-frame arrival, arrival purge, lair gate at the portal,
exe kill bits (Mephisto bit 11, Baal only in level 132, Diablo classic bits) and the Hephasto hammer. `OD2_LEGACY_BOSS=1` (or the
Legacy* fields) restores the earlier model.
