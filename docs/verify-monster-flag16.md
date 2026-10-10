# Attack-rating "+0x16" flags (Game.exe, observations only)

## Result (VERIFIED by disassembly)
The +0x16 tested at 0x57b8b0 is NOT a monstats column. Helper 0x59dd60(unit, mask in EDX) requires unit type (dword [unit]) == 1,
takes unit+0x14 (per-spawn monster data) and returns word [data+0x16] & mask. So it is a per-spawn type mask
(unique / super unique style), not part of the monstats loader table at 0x652b00. The loader table has no bit-field entry at
byte offset 0x16 (only a plain field at offset 0x16, name string 0x6edd38 "UMonSound", type 0x14, plus a flag entry with
bit index 0x16 at flag-dword offset 0xc, which is unrelated).

## Conditions at 0x57b8b0
- stat 0x73 (zero defense): only if the defender is a monster and NONE of: +0x16 & 0xa (0x59dd60, EDX=0xa);
  0x63fa40(0, unit) = monstats row byte +0xc & dword 0x6cf268 (0x40 = boss column, bit 6, see verify-boss-flag.md;
  row = [0x740d78]+0xa78 + index*0x1a8); 0x63fed0(unit) nonzero.
- stat 0x74 (target AC percent) is halved if boss (same 0x63fa40 test), or +0x16 & 2 (0x59dd60 EDX=2), or 0x63fed0 nonzero.
- 0x63fed0 returns by monster class index: 0x10f -> 1, 0x152 -> 2, 0x167 -> 3, 0x230 and 0x231 -> 4, else 0.
- Mask 2 is the super unique bit and mask 8 the unique bit by the usual layout (bit naming UNVERIFIED; the masks are verified).

## Go
d2core/d2monsters/arops.go uses the boss column + Monster.TypeFlags (new; MonTypeSuperUnique set on the leader of a
super unique pack in groups.go; natural uniques/champions are not spawned with a flag yet) + the special classes.
Real-row table test: d2core/d2monsters/arops_flag16_test.go (D2_TABLES).
