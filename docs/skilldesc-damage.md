# skilldesc damage and remaining line kinds (Game.exe 1.14b, base 0x400000)

Addresses and observations only (written to docs/ because the worktree agent cannot write to ~/git/d2-re-notes; copy it there). V = read in the binary, U = inferred. Follows verify-skilldesc.md.

## Corrections to verify-skilldesc.md
* Kind 13 (handler 0x4eb140) is NOT a descdam dispatcher. It loads the word at rec+0xbc of its first argument (U: the skills.txt summon monster id), passes it with calcA and calcB to 0x4e8ed0, which indexes the 0x1a8-stride table at [0x740d78]+0xa78 (U: monstats) and prints "Life: " (string 0x10c6, StrSkill42) + a plain number (0x4e4a30 flag 0). Value = avg + avg*calcA/100 + calcB, avg = (a+b)/2 of the pair returned for the monster at the current difficulty (U: min/max life). 0 gives no line. Shipped kind-13 rows are summons (skeletons, golems, valkyrie, wolves, vines, spirits, shadow).
* The descdam function table is at 0x72bf00 (index = descdam, entries 9..32 filled; 15 = 0x4e6fd0, 26 = 0x4e8e80). It has no code xref; the consumer was not found. 0x4e6fd0 builds a min/max pair from ddam calc1/calc2 (desc rec +0x18/+0x1c via 0x6480c0), the weapon range scaled by SrcDam/128 when SrcDam (skills rec +0x1a5) != 0x80, plus physical and elemental min/max (>>8), then renders via 0x4e6870. So descdam is a separate header-damage path, not a row kind; not modelled (U). Shipped descdam values: 5 (57 rows), 7 (15), 1, 6, 8, 9, 19, 20, 22 (3 each), 17, 2, 3, 4, 10-16, 18, 21, 23, 24.

## Damage rows (V)
* 8 (0x4eaf90 -> 0x4e9380): SKILL_GetToHitBonus 0x645da0; 0 gives no line; else 0x4e4f40 flag 1 with 0x10a6 and 0x10b3: "To Attack Rating: +n percent".
* 9 (0x4eafe9 -> 0x4e9140): texta, textb, then 0x10a0 "Damage: ". lo/hi come from 0x648f90 / 0x6490d0 (physical min/max: weapon range * SrcDam(+0x1a5) >> 7 when the byte is set, + MinDam(+0x1a8), level tiers 0x645f20, synergy calc, << HitShift(+0x1a4)), then >> 8. Each end becomes v + v*calcA/100 + calcB. Both 0: no line. lo != hi: "%d-%d" (0x6dada8); equal: "+%d" (0x6dcec8), "%d" when negative. Newline (0xf9e) after.
* 10 (0x4eb048 -> 0x4e9080 -> 0x4e8f90): elemental min/max 0x646100/0x646200 (flag 1) >> 8; both 0: no line; type byte skills rec +0x1dc picks the label through 0x4e3fa0 (1 fire 0x10a1, 2 lightning 0x10a3, 3 magic 0x10c3, 4 cold 0x10a2, 5 poison 0x10a4); "%d-%d", or "%d" when equal (no plus); calcs unused; texta+textb first. Kind 24 (0x4eb3de) is the same helper with an extra argument (U) after textb.
* 11 (0x4eb0ab -> 0x4eab70): ELen 0x6462f0; etype 4 uses 0x10a9 "Cold Length: ", etype 5 uses 0x10aa "Poison Length: ", other types no line; the value goes through 0x4eaa50 (frames/25, one decimal, " second(s)").
* Magic Arrow rows 1, 11, 10, 9, 8 render in that order; there is no synthesized line.

## Other kinds
* 15 (0x4eb1cd): texta ": " textb.
* 16 (0x4eb211 -> 0x4e4680): textb first, label 0x10b0 "Duration: ", calcA and calcB as frames; each split whole=v/25, tenth=(v%25)*10/25; "%d-%d", or "%d.%d-%d.%d" (0x6dcf00) when either end has a tenth; always plural 0x10ac; 0 at both ends no line.
* 18 (0x4eb272): texta only, wrapped. 25 (0x4eb429): texta then textb concatenated, wrapped. 17 (0x4eb234): textb then range helper 0x4e5020 with unit 0x10be (U). 20/21 (0x4eb33e/0x4eb362): 0x4e52e0 with 0x10b3 (U).
* 23 (0x4eb397): word at 0x3c of the first argument selects a record via 0x466660 (U: missile); value = word[+0x98]*level + word[+0x96] frames as seconds via 0x4eaa50, label texta. 22 (0x4eb366 -> 0x4e94b0) not decoded.
* 31 (0x4eb645 -> 0x4eb3c2): calcA frames divided by field +0x1c of the 0x58-byte record from 0x610fd0 (index = difficulty byte from 0x449240; used when > 0; U: AiCurseDiv of DifficultyLevels.txt), shown as seconds with texta, no textb. Used by Dim Vision, Confuse, Attract.
* 63 and 67 share 0x4e96b0 (value, textb id, plus flag, percent flag): 63 = (1,1), 67 (0x4ebd56) = (1,0): [texta ": "] "+n" ["%"] " " textb. No zero check in 67.
* 71 (0x4ebde5): needs texta and textb both set; texta ": " then textb used as a printf format with the value (0x524310).
* Not decoded (1-8 shipped rows each): 14, 22, 24 (partly), 26-30, 32-36, 39, 41-43, 47-62, 66, 68, 70, 72, 73.

## One box or two (V)
0x4a86b0 sets up one text box (0x4f3ad0) and calls exactly one of 0x4ec180 (with Next Level) or 0x4ec6d0 (chosen by an argument equal to -1); name, description, dsc2, Current, Next and dsc3 are all appended to one buffer, so the block is ONE box. A second tiny box (0x4ffb70, a "%d" number) is the hotkey or assignment number, drawn after the text.
