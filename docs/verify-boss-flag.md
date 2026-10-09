# Monstats flag bits, Align, classic gate (Game.exe, observations only)

## Loader table (VERIFIED)
MONTBL_LoadMonstatsTable 0x00652b00 builds a stack table of 5-dword entries {name string, type, bit index, byte offset, extra}.
For flag columns the byte offset is 0xc (the flag dword) and the third field is the bit index.
Bits (name string address -> bit): isSpawn 0x6edc70 -> 0; isMelee 0x6edc68 -> 1; noRatio 0x6edc60 -> 2; opendoors 0x6edc2c -> 3;
SetBoss 0x6edc58 -> 4; BossXfer 0x6edc4c -> 5; boss 0x6edc44 -> 6; primeevil 0x6edc38 -> 7; npc 0x6edc28 -> 8; interact 0x6edc1c -> 9;
inTown 0x6edc08 -> 10; deathDmg 0x6edba4 -> 19; inventory 0x6edc10 -> 24.
Mask dwords: 0x6cf250=1, 0x6cf254=2, 0x6cf258=4, 0x6cf25c=8, 0x6cf268=0x40.
So 0x6cf268 (mask 0x40 on byte +0xc) = BOSS, not primeevil (the earlier note was wrong; the string addresses are not in bit order).
The earlier "misfit" test (byte +0xf mask 1 = bit 24) is inventory; byte +0xd mask 2 = interact (9); byte +0xe mask 8 = deathDmg (19, sets unit flag 0x40000000).

## Level rule at 0x00571c4f (VERIFIED)
Area level (LEVEL_GetMonsterLevel 0x0061dc00) replaces the monstats Level only if game+0x70 != 0 AND difficulty > 0 AND
flag noRatio (bit 2) clear AND flag boss (bit 6) clear. Radament/Summoner/Izual (boss=1, primeevil blank) keep monstats Level.

## Align (VERIFIED)
Loader entry for "Align" (string 0x6edc90, code 0x00652b35) has byte offset 0x4c. Align==0 gets the player-count bonus (0x00571760);
Align==1 is skipped by the classic adjustment 0x0063ff30.

## game+0x70 (VERIFIED as the expansion flag)
0x0061dc00 third arg (game+0x70 from 0x00571c4f) selects levels row +0x16 (Ex) vs +0x10 (MonLvl); 0x0063ff30 gets it as param_2 and
runs only when it is 0. So "classic game" == not expansion. In classic games the area level is never used; level = monstats Level[diff],
then 0x0063ff30 sets Level normal + 25*diff.

## Go changes
d2common/d2monstats: LevelExclusion switch removed; ResolveLevel(diff, area, expansion) uses Boss + NoRatio and requires expansion.
Open: d2common/d2monster.ResolveLevel (used by d2core/d2monsters/vitals.go) already uses Boss but does not gate on expansion.
