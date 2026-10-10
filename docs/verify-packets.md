# verify-packets: S2C/C2S layouts checked against Game.exe 1.14b handlers

Addresses and observations only (copy to ~/git/d2-re-notes/verify-packets.md). Bit reader: 0x40ca50 -> 0x40c9a0 reads LSB first (first field in the low bits); context init 0x40c880(ctx, buf, len). Reads past the end give zero bits.

## S2C
| Id | Handler / unpack | Layout (verified) |
|---|---|---|
| 0x95 (13) | 0x459240, unpack 0x4591b0 | id:8, life:15, mana:15, stamina:15, x:16, y:16, dx:8, dy:8 (101 bits, 3 spare). Stats 6/8/10 set to value<<8. dx/dy: byte >0x80 -> byte-0x100 (0x80 stays +128). Then 0x47c320(unit, x, y, 0, x+dx, y+dy). If life != 0 and unit mode == 0x11 (dead) the unit mode is reset (0x45ffd0). |
| 0x18 (15) | 0x4590c0, unpack 0x459010 | id:8, life:15, mana:15, stamina:15, A:7, B:7, x:16, y:16, dx:8, dy:8 (115 bits). Same stat writes plus stat 0x4a = A, stat 0x1a = B (unscaled). Stat names UNVERIFIED. |
| 0x59 (26) | 0x459c00 -> 0x4619e0 (UNIT_CreatePlayerUnit) | u32 id @1, u8 class @5, name[16] @6, u16 x @22, u16 y @24. The last two go to the room lookup, so they are a POSITION, not level/party as in the public docs. |
| 0x0f (16) | deferred handler 0x4584b0 | byte @6 mode -> 0x47ca50; u16 @7,@9 target x,y; byte @0xb copied into the move record (meaning unknown); u16 @0xc,@0xe current x,y -> 0x47c320. Unit type/id at 1..5 are consumed by the dispatcher's unit lookup (registers, not traced). |
| 0x0d | pre-hook | dispatcher looks the unit up only when byte @1 == 1 (unit type). |

Dispatcher 0x45af40 pre-hooks: 0x0d, 0x18, 0x95, 0x96 unpack the packet into a local struct before the handler (0x96 unpacker not read here).

## C2S (server handlers, return 0 ok, 2 soft reject, 3 bad)
| Id | Addr | Verified |
|---|---|---|
| 0x38 | 0x549ad0 | len 13; u32 @1 action, u32 @5 npc id (checked as unit type 1 within 0x32 by 0x546e70), u32 @9 param; all three passed to 0x577b70. Action/param meaning UNVERIFIED. |
| 0x3a | 0x549b40 | len 3; u16 @1: low byte = stat (<= 0x0f) passed to allocate-one-point 0x56ec40, high byte = count-1 (<= 99). Loops count times, rc 2 at first failure. Stat byte to attribute mapping UNVERIFIED. |
| 0x3b | 0x549bc0 | len 3; u16 @1 = skill id; used by 0x547370 (can add), 0x644dc0 (level lookup), 0x4a6ff0, 0x56df60 (prereqs). |
| 0x58 | 0x54a7d0 | len 3; u16 @1 index < 0x2a else rc 2; sets bit 0xc in the current-act quest record entry (0x65e870). |
| 0x59 | 0x54a820 | len 17; u32 type @1 (<6), u32 id @5 (range 0x32), u32 @9, u32 @0xd stored as unit values 2 and 3 via 0x58ca10; value 1 set to 0x28 by the handler. |
| 0x5f | 0x54ab60 | len 5; u16 x @1, u16 y @3; validated move (uses 0x620500, 0x642c30 distance, 0x54a920, 0x54aa60). |
| 0x06/0x07 | 0x547c70 / 0x547d00 | len 9; identical except the flag passed to the set-target call 0x620440: 1 for 0x06, 0 for 0x07. Also increments unit stat 0x148. Then 0x547a90 start skill on unit. No skill id on the wire; the selected left skill is used. |
| 0x09/0x0a | 0x547df0 / 0x547e50 | wrappers: len 9, type < 6, range check, then run the 0x06 / 0x07 body with len 9. |
| 0x0d/0x0e | 0x547f50 / 0x547fe0 | right-hand twins (set-target 0x620480; flag 1 / 0). 0x10 (0x5480d0) wraps 0x0d. 0x11 (0x548130) NOT read; assumed to wrap 0x0e. |

Shared helper 0x5475b0 (location packet parse): requires len 5 else rc 3; clears both output words; needs the player alive (0x622380) else rc 2; reads u16 x @1, u16 y @3; range check 0x546de0 (limit 0x32); on failure and more than 0x19 frames since the last reply (game+0xa8 vs client+0x168) queues a correction packet (id 0x15 via 0x539800); on success stores the frame and outputs x, y.

## Code
d2networking/d2gs/verified_msgs.go (+ verified_test.go). The engine's PlayerInGame (0x59 with level/party) differs from the real layout; AssignPlayer is the real one.

## Not done / open
S2C 0x58, 0x5f handlers (0x459bd0, 0x459d10) not read; 0x96 unpack; C2S 0x11 wrapper; meaning of stats 0x4a/0x1a/0x148; byte @0xb of S2C 0x0f.
