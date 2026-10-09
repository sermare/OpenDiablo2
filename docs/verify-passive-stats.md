# Passive / aura stats and state groups, checked in Game.exe (1.14b)

Addresses and observations only. Branch: feat/verify-passive-stats (code in d2common/d2state, d2statlist, d2combat, d2core/d2hero). Copy to ~/git/d2-re-notes/verify-passive-stats.md (the agent could not write outside its worktree).

## State group conflict: resolved
- 0x56c740 SKILL_CreateTimedStateStatList does not read the States.txt group (confirmed again). The group exclusion is FUN_0056a480 (ecx unit, edx state id, stack flag): for a state whose states record word at +0x1e (group) is nonzero it scans all state ids, and for each other state with the same group that the unit has it toggles the bit off (0x63aef0), finds its statlist (0x6258f0), removes and frees it (0x627b30, 0x627020). With the flag nonzero (as the buff cast passes it) the state itself is also ended. Group 0 returns 0 at once.
- Callers of 0x56a480: SRVDO_FrozenArmorState 0x5c7540 (srvdofunc table entry 18 at 0x72fa90, used by the armors, Fade, Burst of Speed and the other generic buffs), SRVDO_116_Wearwolf 0x5c4e80, SRVST_028_BladeShield, SRVST_038_Whirlwind, 0x5bc990, 0x559b8b/0x559b9e, 0x5c4bf8. The call in 0x5c7540 is at 0x5c75ab, before the call to 0x56c740 (0x5c7619).
- Group data (patch_d2 States.txt): group 1 = frozenarmor, bonearmor, chillingarmor, shiverarmor, justhit; group 2 = quickness, fade; group 3 = maul, feralrage, wolf, bear, monsterset, delerium.
- So the earlier conclusion "group is not an exclusion rule" was only true of 0x56c740; the armors do replace each other (and Burst of Speed replaces Fade) via the cast code. 0x5ea850 (monster AI only) is a separate reader.
- Order inside 0x5c7540: flag 0x40 on the unit, 0x56a480, eval auralencalc (+0x60), 0x56c740 (callback 0x5c74e0 as the stat-list param38), 0x5c4c60 (aura stats), 0x5c4d70 (passive stats), stats 0x15e/0x15f = skill/level, unit events auraevent1..3, then 0x63af70 state bit on.

## 0x5c4d70 / 0x5c4c60
- Neither tests a skill flag. Aura: 6 slots, stat ids at rec +0x54, calcs at +0x68. Passive: 5 slots, stat ids +0x98, calcs +0xa4. Per slot in order: stat id valid, calc evaluated (0x648050), a result of 0 is skipped, otherwise STATS_SetBaseStatValue on the new statlist (set, so a repeated stat id in one list is overwritten). If the stat is 0x44 stat 0x45 is set to the same value.
- 0x5c4d70 callers: 0x5c7540, SRVDO_023_Blaze 0x5c7d10, SRVST_038_Whirlwind 0x5d7a00 only. The routine that applies true passives (Iron Skin, Natural Resistance, Warmth) on load or skill change was not found; 0x56db80 is the aura re-run on skill assignment and does not apply passives.
- 0x5c4c60 callers: 17 (buff casts, shapeshifts, Armageddon, Fenris Rage 0x56e490, Berserk, Bash, Blade Shield...).

## Stat getters and ops
- 0x6256e0 (unit) and 0x625760 -> 0x625680 (list) both end in 0x6251d0: they return the stored aggregate value of the stat in the unit's expanded list (with a minimum clamp from the record when flag bit 4 at +5 is set). No op arithmetic at read time.
- Op arithmetic happens when a list is attached/changed: 0x627160 -> 0x626c80 -> 0x625320 (plain add) or 0x626a00 for stats with op records; 0x626430 computes a derived value: ops 1, 11, 13 add MulDiv(total of the op stat, this stat's value, 100) (op 1 and 13 only for item lists, unit type 4, op 11 for players and monsters); ops 2-5 use per-level/shift terms; 6, 7 item-modifier terms; 8, 9 charstats per-point terms. Percent sources therefore sum before the multiply (additive).

## 0x6225a0 COMBAT_GetDefense (from the disassembly)
- base = stat 31 + dex(stat 2)/4 (arithmetic shift with sign fix, truncating). pct = full stat 171 + full stat 16.
- If the unit has state 0x65 (holyshield) and its statlist has skill id (0x15e) > 0 and level (0x15f) > 0 and a hand-item lookup (0x457c20 / 0x63d9d0) succeeds: pct += eval of skills.txt record +0x138 (calc1) at that skill and level. Which item the lookup wants is not decoded.
- pct*base is divided by 100 truncating toward zero; for base > 0 it is added, for base <= 0 it is subtracted (a bonus makes a negative defense less negative).
- Then if stat 0xb6 (182 armor_override_percent) is nonzero: total += MulDiv(total, stat, 100).

## Open
- The true-passive application path; the item test for Holy Shield; how skill-state stat lists are merged into the player's totals on the client (not needed for the server numbers).
