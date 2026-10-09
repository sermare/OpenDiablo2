# verify-diablo-hellforge (Game.exe 1.14b, read-only Ghidra; addresses and observations only)

(Written to docs/ because the worktree agent cannot write to ~/git/d2-re-notes; copy it there.)

Branch feat/verify-diablo-hellforge (fork). Code: d2common/d2boss (seals.go, tomb.go), d2common/d2quest (a345.go).
Tables used: operate fn table base 0x730258 (index = objects.txt OperateFn); init fn table base 0x72f578 (index = InitFn).
Several routines below are not defined as functions in Ghidra (read from raw bytes): 0x5b2830, 0x5b31a0, 0x5b3230, 0x5b3820, 0x5b4770, 0x5b4840, 0x59b450.

## (2) Seals and dummies
- Objects: 392 OperateFn 54 = 0x5b4720; 393, 395 OperateFn 52 = 0x5b3240 (plain); 394 OperateFn 55 = 0x5b4770; 396 OperateFn 56 = 0x5b4840.
- 0x5b4720/70/840 (quest 0x17 data): store seal coords (UNIT_GetCoords) at +0x24/0x28, +0x2c/0x30, +0x34/0x38, then add offsets:
  392 (-0xc, -0x34), 394 (-0x27, +0x21), 396 (+0x20, +0x10). Common tail 0x5b4690: find a room (class 0xd for 392, 0xe 394, 0xf 396), create
  object type 2 class 0x83 (Dummy 131) at the stored position, then call 0x5b3240 (flag +0xc..+0x10, mode 1, event 1 after objects.txt delay).
- Dummy operate: QUEST_OnObjectOperated 0x5428c0 -> 0x5b3360: exact match of the dummy coords with position 1/2/3 -> super unique hcIdx
  in game root +0xb28/+0xb2a/+0xb2c. Those are indices 36, 37, 38 of the hcIdx->row short array at root +0xae0 (+0xae0 + 2*36 = +0xb28).
  So 392 -> hcIdx 36 (Infector of Souls), 394 -> 37 (Lord De Seis), 396 -> 38 (Grand Vizier). The earlier model had 392 and 396 swapped.
- 0x65bdd0 mode 2 = monster count + row; 0x5a2480 spawns the super unique and, for hcIdx 0x24..0x26, attaches quest 0x17 (0x5413a0).

## (1) Diablo arrival
- Dummy 255: InitFn 55 = 0x5b31a0 (object created from the level): quest 0x17 data +6 = 1, +8 = own unit id; if +0x11 clear and all five flags
  set and kill counter (+0x48) == 3: 0x5b2e60, then if +1 clear: +0x18 = 0, +2 = 1, +3 = 0, +1 = 1, QUEST_AddTimer(callback 0x5b2830, delay 1).
  The same block ends 0x5b3240 and the kill handler 0x5b2f10 (whichever condition completes last).
- 0x5b2830 (runs while +2): +0x18++ ; below 10 nothing; then if +6: find object (type 2, id +8) via 0x550fc0, take its room and coords,
  0x5b27b0 -> 0x5b0b60 -> 0x5b0600 spawn class 0xf3 with radius -1 (exact subtile), then 5, then 10; on success OR 0x3000000 into unit +0xc4,
  +0x11 = 1, then +2 = 0, +1 = 0. Spawn failure: nothing set, retried on later frames (not fully traced).
- Second branch (+3, +4 set after Diablo's death): GetTickCount compared with +0x18 + 0x17318 (95 s); clears +1, +3, +4.
- No cutscene/lock code found in these routines. 0x3000000 on +0xc4 is also set on quest NPCs spawned by 0x5b2200 and 0x5428c0 (class 0x1cc): purpose not decoded.
- After arrival (+0x13): 0x54ca00 (random population gate) returns 0 for level 0x6c; MONAI_RestoreInactiveMonster also asks 0x5b2e40.
- Attach: 0x5af8c0 (post-create hook) class 0xf3 -> quest 0x17 (+ COMBAT_Helper_5a2320(0x16,1)), 0xf2 -> 0x14, 0xd3 -> 0xd and 9.

## (3) Hellforge
- 0x5af8c0 case 0x199 (409 = monstats "hephasto") -> FUN_005413a0(0x18): attaches the Hellforge node (id 24, slot 0x1b, InitTheHellforge 0x5b41d0).
- Kill handler 0x5b4190: node active (+9): unit +0xb8 = 'hfh ' (0x20686668), 0x557980(7, ...) drop; on success data +0x10++. No state check.
- Forge object 376: InitFn 48 = 0x5b3630 (mode from node data; node inactive -> mode 3), OperateFn 49 = 0x5b3820 (quest 0x18; needs hammer
  'hfh ' on the player, then flags 0x1b bits 0xd, 1, data +4 = +3 = 1, +8 = 4, +0x14 = 1, mode 3). Object tick 0x5b42d0 hands out gems/runes
  per tier (+8: 1 regular gs*, 2/3 flawless, 4 perfect gp*, final rune by difficulty: tables of r01.. at 0x73b624). 0x5b3710 (data ref 0x5b3989) removes
  'hfh ' and 'mss ' and sets bits 0xd, 1 of slot 0x1b. No quest code creates the forge: it is a level preset object.

## (4) Duriel lair byte +0xb
- 0x59b700: node id 0xd (Seven Tombs) active and data +0xb == 0 -> warp blocked (FUN_00543a70 case level 0x49).
- Writer: callback 0x59b450 (quest timer). Registered by 0x59b960 (staff placement event: removes 'hst ', 'vip ', 'msf ' items; slot 10 bits 0, 0xd; sets
  data +0xd, +0xe = 1; QUEST_AddTimer(0x59b450, (short at (root+0xb64)+0x22a88+0x96 - 0x4b) / 20) once via +3). The callback: orifice mode set, portal
  object 100 created at (X-13, Y+3) of the object with id +0x20, then +0xb = 1, +0xd = 0, +3 = 0. Orifice operate = OperateFn 25 = 0x59b850 (needs 'hst ').

## Options added (all off)
Seals.ExeLayout (pairing, dummy offsets, 11-frame delay, SpawnsClosed), Tomb.ExeLairGate, Game.ExeBossBits now also drops the hammer on Hephasto.
