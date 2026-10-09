# missile-world: Game.exe 1.14b missile points settled (addresses and observations only)

Copy to ~/git/d2-re-notes/missile-world.md (the agent could not write outside its worktree).

Method: Ghidra read-only (read_memory, disassemble_function, get_functions); 0x5ad590 is not a defined function, decoded by hand from read_memory bytes.
Tables: missile records DT+0xb64 stride 0x1a4 (count +0xb6c); skill records DT+0xb98 stride 0x23c. Hit function table 0x739a68 (index*4), SrvDoFunc table 0x739990.

## Hit function 7 = Holy Bolt, 0x5a7a40 (VERIFIED)
- Args: game (ecx), missile (edx), target (stack). No skill/missile record or null target -> return 1.
- If missile rec sHitPar1 (+0x4c) != 0 and an owner exists and the target passes the ally tests 0x552320 / 0x552d80 (exact meaning UNVERIFIED): heal = calc1 (skill +0x138) << 8 + rand((calc2 (+0x13c) << 8) - (calc1 << 8)) via 0x457b60 on the missile seed; if heal <= 0 a fallback from 0x5a6330; added to the target's life (stat 6), clamped to max (0x625f70); a skill word (+0x36) then triggers 0x622020 (visual, not modelled). Returns 1 (kill, no damage).
- Otherwise: sHitPar2 (+0x50) 0 -> 3 for any unit; players pass only for 0 (else 4); monsters: 2 -> 0x63f990 (demon), other nonzero -> 0x63f9e0 (undead); pass 3, fail 4 (bolt keeps flying); non-unit -> 4.
- 0x63f990: monstats flag byte +0xd & dword 0x6cf264 (0x20 = demon). 0x63f9e0: byte +0xd & (0x6cf25c = 8 | 0x6cf260 = 0x10) = lUndead|hUndead. Dword table 0x6cf250: 1,2,4,8,0x10,0x20,0x40,0x80,0x100,0x200,0x400,0x800.
- Data: holybolt sHitPar1 1 ("heals allies"), sHitPar2 1 (undead only), CollideFriend 1, CollideType 3. Skill Holy Bolt calc1/calc2 = min/max heal.

## SrvDoFunc 27 = Tornado, 0x5ad590 (VERIFIED bytes)
- Fails (return 2) without missile record, skill id (0x64b560) or skill record. Period = missile Param1 (+0x38) when > 0 else skill calc4 (+0x144, "damage delay" = par1 = 15), min 1. When (value from 0x64b640(missile) = data short +0xe minus short +0x10, taken as remaining life: UNVERIFIED) % period == 0: damage struct via 0x5a63c0, OR missile HitFlags (+0xa8) / ResultFlags (+0xac), radius = Param2 (+0x3c) when > 0 else skill aurarangecalc (+0x64, par2 = 3), min 1, filter = skill aurafilter (+0x50, 42371 = 0xa583), then the area wrapper 0x569830 (as hit 1). Then tail-jumps to the standard move 0x5abd00.

## Poison cloud lifetime, 0x5abf30 -> 0x5a7170 (RESOLVED)
- 0x5a7170(game, missile, subMissileId, lifetime, subloops) (3 stack args). 0x5abf30 passes lifetime = literal 0 and subloops = 0x64c9b0(missile, owner, rec+0x80, missile id, level), i.e. the missile's SrvCalc1 bytecode ("#subloops"). So SrvDoFunc 2 never overrides the lifetime (poisonjavcloud lives its own Range 60). Flag 8 and the count (overwritten with level*2-2) only when SrvCalc1 > 0: poisonjav 0 (never), viper_poisjav 3, trap poison balls lvl*2 (no SubMissile1, skipped). Effect of flag 8 inside 0x59d5d0 not read.

## Fire Blast (skill "Fire Trauma", id 251, srvdofunc 45 sentries)
- Missiles "bomb in air" (385, SrvHitFunc 36 = 0x5a9ab0) and "bomb on ground" (386, SrvHitFunc 3 = 0x5a7a20 -> 0x5a7890).
- Hit 36: target non-null -> 0. Null target: create HitSubMissile1 (+0x24) at the missile via 0x56cbf0 (parent skill id/level; one value copied 0x64b970 -> 0x64b950, not modelled), return 1.
- Hit 3: target non-null -> 0; null: area damage radius sHitPar1 else skill aurarangecalc (+0x64; Fire Trauma Param1 = 5), 0x5a63c0 damage with HitFlags/ResultFlags ORed, 0x569830; returns 1 (3 if nothing hit).
- 0x5a7890 is the function Ghidra names HitFunc44_DamageStructFromStats; hit 3 shares it.

## World queries in the engine (implemented)
- EnemiesWithin: d2missile.Scan (euclidean subtile radius, living, line of sight with wall bit 4 owner -> unit, lowest unit id; ties go to the later). Engine world: hero owners only, monsters only as candidates, none when the hero is in town. Unit id = brain id (Director.UnitID). Heroes as targets (PvP) not offered.
- Owner.Gone wired for heroes (a dead hero ends Guided Arrow / fire wall makers).
- Unverified: the exe's town test is per room (0x100 / 0x2000), the engine uses the hero's town flag; targetable bits (unit+0xc4 bits 2,3) approximated by Alive.
