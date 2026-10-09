# Skill tree rules, verified in Game.exe 1.14b (addresses and observations only)

## Effective level clamp
- 0x645680 SKILL_GetTotalLevel(unit, skill, includeBonus): base points at skill+0x28; bonus added only when includeBonus != 0 and skill+0x34 == -1 (via 0x645560); floor 0; upper clamp = 0x610ae0(0).
- 0x610ae0 reads element [class] of the table pointed to by 0x9648e8 (loaded by DATATBL_LoadExperienceTable). That table is Experience.txt: first row MaxLvl (99). So the bound is the character level cap (99), NOT skills.txt maxlvl. Items can lift a 20-point skill past 20. The class argument is always 0 here.
- 0x645b20 (next-point level requirement) = skills record +0x174 (reqlevel) + base points.

## Spending a point (packet 0x3b, server handler 0x549bc0)
- Checks: 0x56a440 (player unit; record +0xc class equals unit class); 0x645b90; 0x645ce0; base points (includeBonus 0) below SKILLS_GetMaxSkillLevel (record +0x12c maxlvl, 20 when < 1); then prereqs again; then spends 1 point (0x55d320).
- 0x645b90: char level (stat 0xc) >= reqlevel + base points (EVERY point, not only the first); record +0x17e/+0x180/+0x182 (reqskill1-3): each, if set, needs base points >= 1.
- 0x645ce0: record byte +5 bit 4 must be set; level again; strength(stat 0) >= +0x176, energy(2) >= +0x178, dexterity(1) >= +0x17a, vitality(3) >= +0x17c (reqstr, reqint, reqdex, reqvit).
- Client (0x4a7f40 mouse down, 0x4a8380 up, draw 0x4a8980) only gates by unused points and greys icons with the same two functions plus a cost calc at record +0x170 (default 1; meaning unverified). The server decides.

## Tree icon label (UI_DrawSkillTreeIcon 0x4a86b0)
- Label is the includeBonus=1 total (base + items), drawn when total or base is non-zero; colour from 0x6456e0 (item bonus for the skill): positive blue (3), negative red (1), zero white. Locked icons still show the total.

## Hotkeys (16 entries at client record +0x3dc, 8 bytes each)
- Packet 0x51 handler 0x54a690: slot = high 16 bits of the first dword (< 16), skill = low 15 bits, bit 15 = hand; skill must be -1 or owned (with the item id); 0x5330d0 then 0x536af0 stores it. NO duplicate removal server-side.
- Client arrays 0x7b8750/0x7b8840/0x7b87f0 are written by the slot packet (0x4a67c0) and by 0x4a6490, which clears keys of skills whose level fell to 0 and sends a clear packet. The client sender of a normal assignment, so any duplicate rule, was NOT found: UNVERIFIED.
