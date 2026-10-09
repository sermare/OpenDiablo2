# State application rules, checked in Game.exe (1.14b)

Addresses and observations only. Branch: feat/verify-states, package d2common/d2state. (Copy to ~/git/d2-re-notes/verify-states.md; the agent could not write outside its worktree.)

## Per-effect functions
- 0x578830 COMBAT_ApplyStunState: state 0x15. Length cap 250 frames. For a monster target there are early outs: a random 10 percent skip when a data flag (MONSTER_HasDataFlag16) is set, return when MONSTER_IsStatRecordFlag40, return when a short at monstats record +0x32 is 0 (not ColdEffect), and a cut to 13 frames when the length is above 12 and 0x63fed0 is true. If a stun statlist exists its end is overwritten with the new end (shorter replaces longer). Otherwise a statlist is created, state bit set, expiry event 0xc scheduled.
- 0x578990 poison (state 2) and 0x578b00 burn (state 0x73): same shape. One statlist per unit and kind, stat 0x4a (hpregen) = -value. Existing list: end and value are replaced only if new value >= old value; otherwise nothing changes. Poison and burn are two separate lists, so they add.
- 0x578ca0 chill (state 0xb): slow value is monstats ColdEffect (signed byte at record +0x168 + difficulty) for monsters, -50 otherwise; ColdEffect 0 returns. Stats 0x43, 0x44, 0x45 set to it. For a monster with negative ColdEffect the length is divided by DifficultyLevels record +0x18 (integer), minimum 1. Existing list: only the end is extended when later; value untouched. Afterwards a 20 percent roll toggles state 0x6b on (monsters), else off.
- 0x578f50 freeze (state 1): players (unit type 0) go to chill. Monsters: uninterruptable state 0x36 returns; monstats flag40, data flag16 or 0x63fed0 go to chill; otherwise 0x578c50 returns ColdEffect (default -50) and freeze needs it < 0. Length divided by DifficultyLevels +0x14. Existing list: end only extended.

## 0x56c740 SKILL_CreateTimedStateStatList
- Rejects monsters whose record byte +0xd bit 0 is set, and requires a flag in the helper 0x44d630 result.
- States in the curse mask (0x63b530, = states.txt curse column, mask array index 11): curse_resistance (stat 0x6d) >= 100 rejects; nonzero shortens the length by MulDiv(len, r, 100) (truncating), so the result is len - trunc(len*r/100). Having state 0x39 rejects. The existing list is found by flag 0x20 (any curse), so one curse at a time regardless of name.
- Other states: existing list found by state id.
- Existing list with same state, same skill (stat 0x15e) and same level (0x15f): end set to now+length if length nonzero, nothing rebuilt. Same skill, lower level: rejected. Otherwise the old list is removed and freed and a new one is created. The group column is not read here.
- Group column consumer: 0x5ea850, called from monster AI (MONAI_Think_ShadowMaster and one more): "unit has another state of the same group". Not an exclusion rule.

## State bits
- 0x63aef0 STATE_ToggleUnitState: bounds check against state count, set/clear bit via 0x625ce0, queue unit in room list (so clients get it). 0x63af70 does the same on the previous-state snapshot bits.
- Mask arrays: the states loader builds one mask per bit column at 0x963d3c + 4*index; the engine data struct exposes them at offset 0xcc + 4*index (0xf8 curse, 0x100 plrstaydeath, 0x104 monstaydeath, 0x108 bossstaydeath, 0x150 udead, 0x144 exp).

## Death
- 0x57d310 (player death) and 0x5a3f20 (monster death) call STATS_RemoveNonPersistentStatesOnDeath (0x627890): destroys each state statlist (skipping type 4 lists and flag 0x181) unless STATE_StaysOnDeath (0x63b5b0): players use plrstaydeath, all other units including bosses use monstaydeath.
- 0x63b0d0 then clears the state bits with the mask: type 0 plr, monsters mon, bosses (MONSTER_IsStatRecordFlag40) boss.
- Area change: no clear-on-area caller found; 0x63b0d0 has only the two death callers. Left unverified.

## Client colour
- 0x4d68c0 STATE_ClientAddState: if record byte +0x21 (colorshift) is nonzero it calls 0x4d65a0.
- 0x4d65a0: scans state ids upward, keeps the highest colorpri (byte +0x20) with strict greater-than starting from 0, so ties go to the lowest id and priority 0 never wins; id 0 ignored. Unit palette shift (unit +0x6c) = colorshift, except shift 0x68 (104) is stored as 0 for the local player when the video mode is 3D or higher. The blue column is not consulted here.

## Open
- Meaning of data flag16, flag40, 0x63fed0 and state 0x39; the blue column consumer; difficulty divisor column names (probably MonsterFreezeDivisor and MonsterColdDivisor); area-change state removal.
