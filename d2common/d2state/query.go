package d2state

// State names with built-in meaning. Everything else is just a named bag of
// stat modifiers (the skills.txt aurastate / auratargetstate names).
const (
	// Stun and Freeze stop the unit acting for their length.
	// The names are the states.txt rows: stunned (id 21), freeze (1) and
	// cold (11, the chill of cold length).
	Stun   = "stunned"
	Freeze = "freeze"
	Chill  = "cold"
	// Terror (Terror, Howl, fear): the unit runs away.
	Terror = "terror"
)

// Synthetic stat names for effects that have no ItemStatCost entry in the
// shipped skills.txt (the engine reads them through the query methods).
const (
	// StatIronMaiden: percent of the damage the unit takes that its attacker
	// takes as well (Iron Maiden calc1, ln56; U: the reflected amount).
	StatIronMaiden = "x_ironmaiden"
	// StatLifeTap: percent of the damage dealt to the unit given back as
	// life to the attacker (Life Tap calc1).
	StatLifeTap = "x_lifetap"
	// StatBlind: Dim Vision, the unit cannot see its target.
	StatBlind = "x_blind"
	// StatConfused: Confuse, the unit wanders or attacks its own side.
	StatConfused = "x_confused"
	// StatAttract: Attract, enemies of the unit turn on it.
	StatAttract = "x_attract"
	// StatTaunted: Taunt.
	StatTaunted = "x_taunted"
)

// ChillSpeedPct is the slow of cold damage when the monster's ColdEffect is
// not known: monstats.txt coldeffect is -50 for almost every monster. The
// real value is the monster's ColdEffect of the difficulty (Hit.ColdEffect):
// chill sets velocitypercent (stat 0x43), attackrate (0x44) and
// other_animrate (0x45) to it (skills-combat.md, COMBAT_ApplyColdSlowState
// 0x578ca0, mostly verified).
const (
	ChillSpeedPct       = -50
	ChillAttackSpeedPct = ChillSpeedPct

	// MaxStunFrames caps a stun length (verified: 0x578830, 250 frames).
	MaxStunFrames = 250
)

// LengthAfterResist is a cold, freeze or poison length after the target's
// resist for that length row (the row uses the damage type's resist stats;
// poison length also has item_poisonlengthresist). A resist of 100 or more
// removes the effect, a negative resist lengthens it (U: the exe's rounding
// and whether the negative side is clamped).
func LengthAfterResist(frames, resistPct int) int {
	if frames <= 0 || resistPct >= 100 {
		return 0
	}

	return frames * (100 - resistPct) / 100
}

// CurseLength scales a curse or other timed skill state by the target's
// curse_resistance (stat 0x6d): 100 or more rejects the state, otherwise the
// length shrinks by that percent (SKILL_CreateTimedStateStatList 0x56c740;
// U: rounding).
func CurseLength(frames, curseResist int) int {
	if curseResist <= 0 {
		return frames
	}

	return LengthAfterResist(frames, curseResist)
}

// StunLength clamps a stun length to MaxStunFrames.
func StunLength(frames int) int {
	if frames > MaxStunFrames {
		return MaxStunFrames
	}

	return frames
}

// Hit is the part of a damage struct that creates states and streams.
type Hit struct {
	ColdLen, FreezeLen, StunLen int
	// Poison is the per-frame 8.8 damage and PoisonLen its length in frames.
	Poison, PoisonLen int
	// Burn is the per-frame 8.8 damage and BurnLen its length.
	Burn, BurnLen int
	Source        string
	SkillID       int
	// ChillImmune etc. let the engine pass monster immunities (stat 0x6d is
	// the "cannot be slowed" family for the game); they skip the state.
	CannotChill, CannotFreeze, CannotStun bool
	// ColdEffect is the monster's monstats ColdEffect for the difficulty
	// (negative = slow percent) and HasColdEffect says it was supplied. When
	// supplied, a 0 skips chill and freeze (skills-combat.md says the exe skips
	// them; U for stun) and any other value is the chill slow.
	ColdEffect    int
	HasColdEffect bool
}

// ApplyHit turns the lengths of a hit into states and streams on the set and
// returns the names applied (for logs).
func (s *Set) ApplyHit(frame int, h Hit) []string {
	var out []string

	if h.HasColdEffect && h.ColdEffect == 0 {
		h.CannotChill, h.CannotFreeze = true, true
	}

	if h.StunLen > 0 && !h.CannotStun {
		s.Apply(frame, Instance{Name: Stun, Until: frame + StunLength(h.StunLen), Source: h.Source, SkillID: h.SkillID})
		out = append(out, Stun)
	}

	if h.FreezeLen > 0 && !h.CannotFreeze {
		s.Apply(frame, Instance{Name: Freeze, Until: frame + h.FreezeLen, Source: h.Source, SkillID: h.SkillID})
		out = append(out, Freeze)
	}

	if h.ColdLen > 0 && !h.CannotChill {
		slow := ChillSpeedPct
		if h.HasColdEffect {
			slow = h.ColdEffect
		}

		s.Apply(frame, Instance{Name: Chill, Until: frame + h.ColdLen, Source: h.Source, SkillID: h.SkillID,
			Mods: []StatMod{{"velocitypercent", slow}, {"attackrate", slow}, {"other_animrate", slow}}})
		out = append(out, Chill)
	}

	if h.Poison > 0 && h.PoisonLen > 0 {
		s.AddStream(frame, "poison", h.Poison, h.PoisonLen, h.Source, h.SkillID)
		out = append(out, "poison")
	}

	if h.Burn > 0 && h.BurnLen > 0 {
		s.AddStream(frame, "burn", h.Burn, h.BurnLen, h.Source, h.SkillID)
		out = append(out, "burn")
	}

	return out
}

// CanAct is false while the unit is stunned or frozen.
func (s *Set) CanAct(frame int) bool { return !s.Active(frame, Stun) && !s.Active(frame, Freeze) }

// Fleeing reports whether the unit is afraid (Terror, Howl).
func (s *Set) Fleeing(frame int) bool { return s.Active(frame, Terror) }

// SpeedPct is the movement speed change in percent (negative slows), from
// velocitypercent mods (curses, Holy Freeze, chill, Vigor...). Clamped so a
// unit never moves backwards: at least -100.
func (s *Set) SpeedPct(frame int) int {
	v := s.Stat(frame, "velocitypercent") + s.Stat(frame, "item_fastermovevelocity")
	if v < -100 {
		v = -100
	}

	return v
}

// AttackSpeedPct is the attack speed change in percent: stat attackrate
// (0x44), which chill, Decrepify and Holy Freeze lower and Frenzy and
// Fanaticism raise. other_animrate (0x45, the other animations: hit recovery,
// block...) is OtherAnimPct; they were summed before, which counted a chill
// three times.
func (s *Set) AttackSpeedPct(frame int) int {
	v := s.Stat(frame, "attackrate")
	if v < -100 {
		v = -100
	}

	return v
}

// OtherAnimPct is the speed change of the non-attack animations in percent
// (stat other_animrate).
func (s *Set) OtherAnimPct(frame int) int {
	v := s.Stat(frame, "other_animrate")
	if v < -100 {
		v = -100
	}

	return v
}

// ResistDelta is the change to a resist percent of a damage kind ("phys",
// "fire", "ltng", "cold", "pois", "mag") from states: Amplify Damage is
// damageresist -100, Lower Resist and Conviction fire/cold/light/poison
// resist penalties, Resist/Salvation auras bonuses. The engine adds it to
// the unit's base resist before d2combat.EffectiveResist.
func (s *Set) ResistDelta(frame int, kind string) int {
	switch kind {
	case "phys":
		return s.Stat(frame, "damageresist")
	case "fire":
		return s.Stat(frame, "fireresist")
	case "ltng":
		return s.Stat(frame, "lightresist")
	case "cold":
		return s.Stat(frame, "coldresist")
	case "pois":
		return s.Stat(frame, "poisonresist")
	case "mag":
		return s.Stat(frame, "magicresist")
	}

	return 0
}

// DamagePct is the damage dealt change in percent (Might, Weaken is
// negative, Decrepify, Concentration, Fanaticism, Maul...).
func (s *Set) DamagePct(frame int) int { return s.Stat(frame, "damagepercent") }

// DefensePct is the defense change in percent (Defiance, Shout/Battle
// Orders' skill_armor_percent, Frozen Armor's, Conviction's penalty).
// armor_override_percent -100 (Berserk) sets the defense to zero.
func (s *Set) DefensePct(frame int) int {
	if s.Stat(frame, "armor_override_percent") <= -100 {
		return -100
	}

	return s.Stat(frame, "skill_armor_percent") + s.Stat(frame, "item_armor_percent")
}

// AttackRatingPct is the attack rating change in percent (Blessed Aim,
// Fanaticism, Taunt's and Weaken-like penalties).
func (s *Set) AttackRatingPct(frame int) int { return s.Stat(frame, "item_tohit_percent") }

// ReflectPct is the Iron Maiden percent: damage dealt to the cursed unit also
// hurts the attacker.
func (s *Set) ReflectPct(frame int) int { return s.Stat(frame, StatIronMaiden) }

// ThornsPct is the percent of melee damage taken that is returned (Thorns).
func (s *Set) ThornsPct(frame int) int { return s.Stat(frame, "thorns_percent") }

// LifeTapPct is Life Tap's percent of damage received that heals the attacker.
func (s *Set) LifeTapPct(frame int) int { return s.Stat(frame, StatLifeTap) }
