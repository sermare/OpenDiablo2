# True passive skills in Game.exe (1.14b): found

Addresses and observations only. Copy to ~/git/d2-re-notes/true-passives.md (the agent could not write outside its worktree). Branch feat/true-passives (engine: d2common/d2skill TruePassiveStats, d2core/d2skills Engine.PassiveTotals, hooked into HeroStatsState.SkillStats in d2game/d2gamescreen/game_skills.go).

## The path (verified from decompile)
- 0x648130 SKILL_Func_648130 (unit, skill id) is the per-skill passive applier/remover. Called from skill add/set-level (SKILL_AddOrSetSkillLevel 0x648662/0x64869b/0x6486c5), 0x6482e0, 0x648390, 0x6484d0, 0x648e70, 0x648ef0 (item-granted skill level changes), SKILL_ValidateSkillPrereqs 0x56e05b, and 0x559928.
- 0x56bd50 SKILL_ApplyPassiveStatesForUnit: loops the unit's skill list; for each skill with passivestate > 0 (0x644ae0 reads record +0x94) toggles the state on and calls 0x648130. Callers: 0x5cd074, 0x559a8f, SERVER_HandleC2S41_RespawnPlayer 0x549f81 (join / respawn).
- Gate is the passivestate column (record +0x94, short) > 0. The skills.txt `passive` flag is NOT tested. So Resist Fire/Cold/Lightning and Blessed Aim (passive flag blank, passivestate set) go through the same path.
- 0x648130: finds the unit's skill entry, total level = SKILL_GetTotalLevel(unit, entry, 1) (includes +skills). Level 0 / missing entry: the state's statlist is found, removed and freed. Else a state-keyed statlist is allocated/attached (0x644a70); if it already holds stat 0x15f equal to the level it returns early. Then for slots 1..5: stop at the first invalid stat id (rec +0x98 shorts), value = calc eval at +0xa4, STATS_SetBaseStatValueIfList(list, stat, value, param). Zero values are NOT skipped here (the aura writer 0x5c4d70 skips them). Then stats 0x15e (skill id) and 0x15f (level) are set and the passive state bit is turned on.
- Param: the short at +0x96 (passiveitype, item type index). If >0 it is passed as the stat param/layer, else 0. So passiveitype is NOT an equip gate in this routine: mastery stats are always applied, keyed by item type. The weapon check must happen at the reader (combat code looking up the stat with the equipped weapon's type). Reader side NOT checked (unverified).
- Before applying, if record +0x80 (short; probably aurastate, unverified) > 0 and the unit has that state, the passive list is removed instead.
- Result lives in a stat list on the unit keyed by the passivestate id, with 0x15e/0x15f marking skill/level; separate from aura/buff lists.

## Engine
Hero totals now include true passives with no weapon key (Natural Resistance, Increased Speed, Iron Skin, Resist x, Warmth...). Weapon-keyed mastery stats are left out of plain totals. Not modelled: the +0x80 removal rule, reading mastery stats per equipped weapon.
