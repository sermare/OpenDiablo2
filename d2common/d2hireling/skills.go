package d2hireling

// maxSkillLevel caps a merc skill level (V).
const maxSkillLevel = 32

// SkillLevel is the level of skill slot i at merc level mlvl:
// clamp(LevelN + (LvlPerLvlN*d >> 5), 1..32) with d = mlvl - row.Level
// (V formula; the grouping was checked against the table: rogue Fire Arrow
// Level 1 at row level 3 and 7 at row level 25 with LvlPerLvl 10).
func (r *Record) SkillLevel(i, mlvl int) int {
	d := mlvl - r.Level
	s := r.Skills[i]
	l := s.Level + (s.LvlPerLvl*d)>>5

	if l < 1 {
		l = 1
	}

	if l > maxSkillLevel {
		l = maxSkillLevel
	}

	return l
}

// SkillEnv answers the questions the skill choice asks about the unit and the
// skills table. Nil funcs mean "no restriction".
type SkillEnv struct {
	// ReqLevel is the skills.txt reqlevel of a skill name.
	ReqLevel func(name string) int
	// StateActive reports that a buff/aura skill's state is already up on
	// the unit (V: only for skills whose flag byte +0x230 is 1). It also
	// answers false for skills that are not such buffs.
	StateActive func(name string) bool
	// TooFar is the skill 0x29 rule (target farther than skillLevel/2+4).
	TooFar func(name string, skillLevel int) bool
}

// Choice is the outcome of ChooseSkill.
type Choice struct {
	// Default means the plain attack (roll below DefaultChance, or nothing
	// eligible).
	Default bool
	Slot    int // skill slot 0..5 when !Default
	Skill   SkillSlot
	Level   int
}

// ChooseSkill is MERC_ChooseAndQueueSkill's selection (V): acc starts at
// DefaultChance and each eligible skill adds Chance + ChancePerLevel*d/4;
// r = roll(acc+1); r < DefaultChance is the default attack, otherwise the
// first skill whose cumulative weight is >= r. roll(n) returns [0,n).
func (r *Record) ChooseSkill(mlvl int, env SkillEnv, roll func(n int) int) Choice {
	d := mlvl - r.Level
	if d < 0 {
		d = 0
	}

	acc := r.DefaultChance

	type cand struct {
		slot, level, cum int
	}

	var cands []cand

	for i, s := range r.Skills {
		if !s.Used() {
			break // V: stops at the first empty slot
		}

		if env.ReqLevel != nil && env.ReqLevel(s.Name) > mlvl {
			continue
		}

		lvl := r.SkillLevel(i, mlvl)
		if env.StateActive != nil && env.StateActive(s.Name) {
			continue
		}

		if env.TooFar != nil && env.TooFar(s.Name, lvl) {
			continue
		}

		acc += s.ChancePerLevel*d/4 + s.Chance
		cands = append(cands, cand{i, lvl, acc})
	}

	v := roll(acc + 1)
	if v < r.DefaultChance {
		return Choice{Default: true}
	}

	for _, c := range cands {
		if c.cum >= v {
			return Choice{Slot: c.slot, Skill: r.Skills[c.slot], Level: c.level}
		}
	}

	return Choice{Default: true}
}
