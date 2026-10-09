# verify-missiles-2: remaining missile points (Game.exe 1.14b; addresses and observations only)

Method: Ghidra read-only (get_functions, disassemble_function, read_memory, xrefs) plus llvm-mc for raw bytes. No disassemble_bytes, no renames. Corrects two inferences of verify-missiles.md (marked CORRECTION).

## (1) Per-cell test 0x6513e0 (VERIFIED)
- Called per entered subtile by the step routine 0x6515b0 (from 0x651ad0). For a missile (path flag 0x40000 set) it reads the cell through 0x650140 -> 0x64ec70(room,x,y,shape,mask) with mask = path+0x50, ORs the result into path+0x54, then: masked flags nonzero and (flags & 5) != 0 -> blocked (returns 0); otherwise passes (returns 1). So bits 0x80/0x100/0x40 in the mask are collected (used by the unit scan) but never block.
- path+0x50 is set by CreateServerMissile at 0x59d92d from table 0x73994c (+8*CollideType) via 0x64a070 (PATH_SetBlockMask). VERIFIED: the mask really is the per-type table value.
- Blocked step: stops at the last free cell, node index := node count, so PATH_AdvanceUnitOneFrame returns 0, SERVER_AdvanceUnitPathFrame 0x552d00 returns 2 and 0x5abd00 calls ProcessHitOrExpire(0,1) (hit function with null target, then destroy).
- CORRECTION: a wall/walk bit OUTSIDE the type's mask is invisible to a moving missile (it never enters path+0x54), so it flies over it. It does NOT "vanish silently". Only a missile with velocity 0 re-reads the cell with mask 0x7fff (0x64a240) and then dies silently on (flags & 5) with return 2.
- Out-of-grid cell reads 0x27; masked it blocks every type whose mask has 0x4 or 0x1 (1,2,3,5,6,8), not 7.

## (2) Guided Arrow retarget (VERIFIED)
- 0x5a8060: returns 1 if data +0x28 bit 4; owner via 0x551030; calls 0x569a80(game, owner, X, Y, radius = rec Param2 (+0x3c), flags 3, -1, 0). 0x569a80 ORs 0xa783 into the flags, installs callback 0x569a40 with a 6-dword context, calls scan 0x569510 (Skills.cpp line 0x323), result = ctx[0] or ctx[2].
- Scan 0x569510: for each room near the centre (0x6194b0, rect test 0x5694a0) walks the room unit list (room+0x74, next at unit+0xe8). Excludes the owner. Position of a unit: PATH_GetXPos/YPos (integer subtile) for types 0/1/3, path+0xc/+0x10 for 2/4/5. Distance via 0x64a620 squared against radius^2; centre is the MISSILE position. Radius unit: subtiles, euclidean.
- Filter 0x569100 with 0xa783 = players (not mode 0 / 0x11) and monsters (not mode 0 / 0xc), unit+0xc4 bit 2 and bit 3 set, not in a town room (0x100 and room loop 0x2000), enemy test 0x552270 (0x8000), line of sight owner -> candidate through 0x64f5f0 with mask 4 (0x200).
- Callback 0x569a40 (not a defined function; bytes 85 d2 8b 41 0c ...): ecx = scan frame, edx = candidate. It compares candidate+0xc (the UNIT ID, confirmed by 0x64b690 storing it as the last-hit id) as unsigned against ctx+0xc (initially -1) and keeps the candidate with the LOWEST unit id (ties: the later); the branch that uses ctx+0x4 is dead because ctx+0x10 = -1. So the pick is the lowest unit id, NOT the nearest and not the first in room order.
- 0x5a7f10 (applies it): life = (lvl-1)*LevRange + Range; mode 5 with target (PATH_SetTargetUnit) or mode 6 ground leg; path recomputed only if UNIT_GetDistanceToUnit/Point < 25.
- Homing window unit (SrvDoFunc 7): UNIT_GetDistanceToUnit 0x642b10 = |dx|,|dy| of integer subtile positions, each reduced by (size1/2 + size2/2) (UNIT_GetSizeX), floored at 0, result = (min + 2*max)/2 (max + min/2). So subtiles, octile-like metric, not euclidean. UNIT_GetDistanceToPoint 0x642c30 is the same without sizes.

## (3) Area radius when sHitPar1 is empty (VERIFIED)
- skills.txt record offsets: +0x138 = calc1, +0x64 = aurarangecalc, +0x148.. = Param1..Param8 (so +0x150 = Param3, +0x154 = Param4).
- Hit 1 (0x5a7500): radius = sHitPar1 if > 0 else calc1 of the missile's casting skill (min 1). Hit 14 (0x5a8680, Meteor): sHitPar1 if > 0 else aurarangecalc; flames live Param3 + (lvl-1)*Param4 when > 0.
- Data (patch_d2): fireball sHitPar1 = 4 (table wins); meteorcenter sHitPar1 = 0 -> Meteor aurarangecalc ln12 = Param1 6 + (lvl-1)*Param2 0 = 6 subtiles, Param3 30 + (lvl-1)*15 frames of fire; vampiremeteor Param1 5, Param2 1. Exploding/Freezing Arrow explosions use sHitPar1 5. catapult cold ball (hit 1, no skill, empty) has no calc, radius would fall to the min.

## (4) CollideType 1 and 7 (VERIFIED)
- 1 (0x5a6210): player, or a monster for which 0x625c10 == 2: that function returns 2 for non-units (type > 1), else, for a unit with stat list, the full stat 0xac (172, "alignment" in ItemStatCost) of its state-0x69 (105, "alignment" in States.txt) stat list, else 0. So type 1 missiles hit players and monsters in alignment state with alignment value 2. Meaning of the value 2 unverified.
- 7 (0x5a62b0): target must be a missile (unit type 3) whose missiles.txt flag byte (rec+4) has bit 0x10 (global dword 0x6cf260 = 0x10 = CanDestroy; the dword table at 0x6cf250.. is 1,2,4,8,0x10,0x20 = LastCollide, Explosion, Pierce, CanSlow, CanDestroy, ClientSend). CreateServerMissile (0x59d944) sets unit flag 4 on CanDestroy missiles so the common predicate accepts them. Mask 0x40 has no terrain bits: type 7 flies through walls. No shipped missile uses type 7; used types are 1, 3, 6, 8.

## (5) SrvDoFunc (VERIFIED unless noted)
- 6 (0x5ac1b0): owner (0x551030 chain) and SubMissile1 (rec+0x18 >= 0) and a path must exist, else return 2 (no hit function). If path flag 8 (set by 0x6515b0 when a step entered a cell; cleared at the start of every 0x651cc0) spawn SubMissile1 with create flags 0x21 at the missile subtile (source = dest), skill id and level of the parent, then standard move. The spawn runs BEFORE the move, so it lags the entry by one frame. Rows: firewallmaker (collide 8, SubMissile1 firewall), moltenboulder, vampirefirewallmaker, countessfirewallmaker.
- 2 (0x5abf30): needs SubMissile1; evaluates a value through 0x64c9b0 (an overlay lookup keyed by rec+0x80) and calls 0x5a7170: same flag-8 trigger, create flags 5 (explicit source, explicit velocity left 0, so the sub missile stands still), 0x8000 (explicit range) when the first value > 0, flag 8 (sub-loop) when the second > 0, sub-loop count then overwritten with level*2-2. The exact argument mapping of the two optional values stayed unresolved (decompiler and disassembly disagree on stack slots). Rows: poisonjav (SubMissile1 poisonjavcloud), two trap poison balls, viper_poisjav.
- 5 (0x5ac050): per-frame animation counter in unit+0x44>>8 against SubStart (rec+0x181); stamps bit 0x40 into the cell under the missile (0x64fdd0); then standard move. No gameplay effect beyond the 0x40 marker. Rows: meteorfire, firewall, blaze, immolationfire, countess/vampire variants, moltenboulderfirepath.
- 1 (standard): firebolt, fireball, frostnova, magicarrow, bonespear, hydra, holybolt (hit func 7). 7: guidedarrow. 3 (0x5abfb0): poisonjavcloud (stamps 0x40). 27: tornado (not read).

## Open
- 0x5a7170 optional argument mapping (lifetime / sub-loop count of the poison cloud).
- Meaning of alignment value 2; which skills set state 105.
- Hit 7 (Holy Bolt), SrvDoFunc 27 (Tornado), Fire Blast trap bodies not read.
