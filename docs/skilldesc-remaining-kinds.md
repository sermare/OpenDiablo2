# skilldesc remaining line kinds (Game.exe 1.14b, base 0x400000)

Addresses and observations only (written to docs/ because the worktree agent cannot write to ~/git/d2-re-notes; copy it there). V = read in the binary, U = inferred. Follows skilldesc-damage.md.
Handlers sit in the jump table 0x4ec048 inside fn 0x4eade0 (Ghidra now shows them as part of that function; disassemble_function works).

## Handler register contract (V)
At the jump: EBX = calcA value, [esp+0x10] = calcB value (both via 0x6480c0), ESI = texta id, EDI = textb id, EBP = output buffer, EDX = skills.txt record (word +0x3c = missile id for 22/23). String id 0x1506 = empty. 0x524060 is a plain wide-string append; 0x5222f0 fetches a string by id. Every helper ends with a newline (0xf9e). The text box shows later appends above earlier ones, so a handler that appends two lines shows them in reverse (U, inferred from descline order).
Share of the 1135 shipped line rows: kinds decoded before 92.4%, after this work 96.8%.

## Decoded now (V unless noted)
* 14 (0x4eb16c -> 0x4eabd0): texta, textb raw; then ELen (0x6462f0), per-frame elemental min/max (0x646100/0x646200 flag 1) times ELen >> 8; the length is appended first via 0x4eaa50 with prefix 0x2b0e "over " (patchstring StrSkill63Patch), then label from 0x4e3fa0 (skills rec +0x1dc) + "lo-hi" or "n" when equal. Poison skills (8 rows).
* 20 / 21 (0x4eb33e / 0x4eb362 -> 0x4e52e0 flag 1/0): texta, "+n" or "n", " percent" (0x10b3), textb; 0 no line. No shipped rows.
* 22 (0x4eb366 -> 0x4e94b0 -> 0x4e93b0): missile record from skills rec +0x3c; min/max via 0x64c320/0x64c3f0 (times 0x4b, >>8, helpers 0x4e3ee0/0x4e3f40; exact formula U), element via 0x64c2e0 (1 fire 0x10a1, 2 0x10a3, 3 0x10c3, 4 0x10a2, 5 0x10a4, 6 0xdc4). Output "Average " (0x10e2) + label + "lo-hi"/"n" + " per second" (0x10be). Both 0: nothing. Meteor, Molten Boulder, Armageddon, Immolation Arrow.
* 23 (0x4eb397): missile record by 0x466660 from skills rec +0x3c; frames = word[+0x98]*level + word[+0x96]; 0x4eaa50 flag 0 with texta (e.g. "Duration: ", "Fire Duration: "). Missing record: no line.
* 24 (0x4eb3de -> 0x4e9080): textb appended first, then the kind-10 helper with texta as the label (replaces the element label).
* 51 (0x4ebafc -> 0x4e4990): swprintf(texta as format, calcA) + newline; empty texta gives an empty line. 66 (0x4ebd3a -> 0x4e9b60): same but empty texta gives no line.
* 52 (0x4ebb18 -> 0x4e5020 flag 1): texta + "+lo-hi" (format 0x6dcedc "+%d-%d") + textb; lo == hi goes to the kind-3 helper 0x4e4f40.
* 58 / 62 (0x4ebbe9 / 0x4ebcb6 -> 0x4e5110 flag 1/0): texta, textb, then "lo-hi" (58 with "+"); equal ends fall to 0x4e4f40 (kind 3).
* 57 (0x4ebbe0): 0x4eaa50 with flag 1 (plus sign; tenths branch 0x4e4470 not read, U).
* 72 (0x4ebe67) / 73 (0x4ebf0c): need calcA != 0 and calcB > 0; "+a/b texta" (72, 0x4e4420 flag 1) or "a/b texta" (73); separator is string 0xfa0 "/".
* 0x4eaa50: label id, then n or n.t (frames/25, tenths), unit 0x10ab if exactly 1 else 0x10ac, newline; the flag selects the plus form.

## Not decoded (UNVERIFIED), by shipped rows
17 (1; 0x4eb234), 26 (2), 27, 36, 39, 70 (3 each), 32 (2), 28-30, 33-35, 41, 42, 47-50, 59, 61, 68 (1 each), 53-56 (scroll/book rows, helpers 0x4a6190..0x4a6280 then 0x4ea310), 60/61 look like 0x4e4b00 one decimal with default divisor 10 (U).
