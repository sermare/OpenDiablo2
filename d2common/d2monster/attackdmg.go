package d2monster

// Monster attack damage selection (audited 2026-10 against Game.exe 1.14b).
//
// VERIFIED (MONSTER_SetupLevelScaledStats 0x5a2960, called by
// MONAI_ExecuteAiCommand with the requested unit mode before every attack and
// skill): the unit's physical damage stats (0x15 min, 0x16 max) and attack
// rating (0x13) are loaded from one monstats column set chosen by the MODE:
//
//	mode 5 (A2)          -> A2MinD / A2MaxD / A2TH   (table selector 0x10)
//	modes 7 (SC), 8 (S1) -> S1MinD / S1MaxD / S1TH   (selector 0x20)
//	every other mode     -> A1MinD / A1MaxD / A1TH   (selector 0x08)
//
// so S2, S3, S4 and SQ casts that reach a melee or "source damage" effect use
// the A1 numbers, not nothing. monstats has no S2..S4 damage columns.
// Elemental El1..El3 columns are added when the El#Mode byte equals the mode
// (and a roll under El#Pct passes, no roll at 100 or more).
//
// VERIFIED (0x569ee0, 0x56cde0, 0x56a260): mana cost, the skills.txt delay
// cooldown (state 0x79) and ranged ammo are checked for players only; a unit
// of type 1 (monster) always passes, so monsters have no mana, cooldown or
// ammo bookkeeping: their pacing is the AI (aidel, aip percentages).

// Column is a monstats damage column set.
type Column int

// Damage columns.
const (
	ColumnA1 Column = iota
	ColumnA2
	ColumnS1
)

// ColumnForMode is the VERIFIED mode to damage column rule of 0x5a2960.
func ColumnForMode(m Mode) Column {
	switch m {
	case ModeAttack2:
		return ColumnA2
	case ModeSpecialCast, ModeSkill1:
		return ColumnS1
	}

	return ColumnA1
}

// ElementFires reports whether an El#Mode column applies to a mode (the exe
// compares the stored mode byte with the requested mode, 0 = unused).
func ElementFires(elMode, mode Mode) bool { return elMode != 0 && elMode == mode }

// EffectKind says what a skills.txt row does when a monster casts it.
type EffectKind int

// Effect kinds.
const (
	// EffectNone: no direct damage (curse, summon, teleport, resurrect, aura,
	// spawner...). The cast only plays its animation and side effects.
	EffectNone EffectKind = iota
	// EffectMelee: a hit at the target with the unit damage of the mode
	// (SrcDam percent of it) plus the row's own elemental part.
	EffectMelee
	// EffectMissile: the row launches srvmissile; damage is the missile's.
	EffectMissile
	// EffectArea: a nova, ring or ground effect whose damage is the row's
	// own elemental/physical columns.
	EffectArea
)

// meleeDoFuncs are the srvdofunc values whose handlers call
// MONSTER_SetupLevelScaledStats (VERIFIED callers: Jab 7, Charge 67, Smite
// 150, FireHit 83 via its start func, Leap start 77/78) plus the generic
// Attack (1) and Bash-like (2) handlers (UNVERIFIED: shared with players).
var meleeDoFuncs = map[int]bool{1: true, 2: true, 7: true, 67: true, 77: true, 78: true, 83: true, 89: true, 103: true, 150: true}

// areaDoFuncs are srvdofunc values of damaging ground/nova skills in
// skills.txt used by monsters (UNVERIFIED mapping, from the row layout: nova
// 22, ring/pulse 95, chain 26, cold nova 22, meteor 28, firewall 24).
var areaDoFuncs = map[int]bool{22: true, 24: true, 26: true, 28: true, 95: true, 100: true, 134: true, 136: true, 139: true}

// ClassifyEffect picks the effect kind of a skills.txt row from its srvdofunc
// and whether it names a srvmissile.
func ClassifyEffect(srvDo int, hasMissile bool) EffectKind {
	switch {
	case hasMissile:
		return EffectMissile
	case meleeDoFuncs[srvDo]:
		return EffectMelee
	case areaDoFuncs[srvDo]:
		return EffectArea
	}

	return EffectNone
}
