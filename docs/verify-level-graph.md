# verify-level-graph: level transition gates, portals, QuestFlag (Game.exe, read-only Ghidra)

Addresses and observations only. Quest node = entry of the per-game list (+0x10F4); node+9 = active, node+0x18 = private data; "id" = quest id, "slot" = record slot (quests.md). (Written to docs/ because the agent could not write to ~/git/d2-re-notes; copy it there.)

## 1. QuestFlag column and the warp gates
- Levels.txt loader DATATBL_LoadLevelsTable_Layout 0x61dd60 registers QuestFlag (name 0x6e9c20) and QuestFlagEx (0x6e9c14) as the first two words of the record. VERIFIED consumer: SERVER_UsePortalObject 0x582700 reads rec[0] (classic) or rec[1] (when game+0x70 is set, taken as the expansion flag, inferred) and calls QUESTREC_GetFlag 0x65e820 with bit 0 ("done"); if the slot is not done the use of a town portal object (class 0x3b) is refused. So the column is a record slot, bit 0.
- The warp-tile handler SERVER_EnterWarpTile 0x553140 does not read the column. It has its own destination list (FUN_00543a70): 0x49 (73), 100, 0x76 (118), 0x80 (128), 0x84 (132). A nonzero result cancels the warp.
  - 73 Duriel's Lair: FUN_0059b700 = Seven Tombs node (id 13, slot 14) exists, is active, and private byte +0xb is 0 -> blocked. What sets +0xb was not traced (the restore routine 0x59b0d0 does not set it). Not slot 13, not slot 10 directly.
  - 100 Durance of Hate 1: FUN_005b9b60 = Blackened Temple node (id 19, slot 21) private byte +0xc is 0 -> blocked, unless the source room is level 0x65 (101). Byte +0xc is restored on join from slot 18 (Khalim's Will) bit 0 (0x5b9050). The Orb/Khalim's Will drives it; the holder is the Blackened Temple node.
  - 118 and 128 from source 120 (Arreat Summit): FUN_0058ae70 = Rite of Passage node (id 35) active and private byte +0 is 0 -> blocked.
  - 132 (Worldstone Chamber): FUN_0058c3f0 = Eve of Destruction node (id 36); allowed only when private byte +0x86 == 1.

## 2. Town portal
- Cast routine FUN_005bbe10: fails in a town room (DRLG_Helper_61a850) and in level 0x88 (136); destination = DRLG_GetActStartLevel 0x61a8b0 of the room's act; table at 0x6e92c8 = 1, 40, 75, 103, 109. VERIFIED: always the act's town.

## 3. Portal factory QUEST_Func_56ae80 (0x56ae80), 11 call sites
Args (room, x, y, destLevel, outUnit, objectClass, permanent). In a town room it refuses unless dest is 0x27 (39) or 0x85..0x88 (133..136) and class is 0x3c (permanent portal); dest act must equal the room's act.
- 0x590b92 Search for Cain: dest 0x26 (Tristram), class 0x3c, permanent. VERIFIED.
- 0x59116f Cain's gibbet: dest 1, class 0x3b.
- 0x598d56 Arcane Sanctuary journal message: dest 0x2e (Canyon of the Magi), class 0x3c. VERIFIED.
- 0x588ea9 and 0x589b7d (Act 5 Betrayal/Anya): dest 0x79 (121), class 0x3c, permanent. VERIFIED.
- 0x59a7b2 Seven Tombs (Tyrael): dest 0x28 (40), class 0x3b.
- 0x5a7429 (missile callback 0x5a73a0, dest from a missile record field +0x44): permanent portal, level not resolved.
- Cow Level (39) and the 133..136 portals: allowed by the whitelist but the creating call site was not found. Unresolved.
- Arcane Sanctuary entry from Palace Cellar 3 and the portal to Duriel's lair: no factory call names 74 or 73; probably objects (D2MOO: 298 and 100). Unresolved.

## 4. Chaos Sanctuary
- 107 -> 108: no gate (not in the EnterWarpTile list, no QuestFlag). Terror's End (id 23, slot 26) OnAreaChange 0x5b2ad0 only watches levels 0x67 and 0x6c for the log. Seal effects were not traced in the exe.

## 5. Levels 125-127 and 133-136
- 125-127: no Vis/Warp in Levels.txt, no factory call or whitelist mentions them: apparently unused. 133-136 are entered by permanent portals made in Harrogath (whitelist VERIFIED), creator unlocated. TP cannot be cast in 136.
