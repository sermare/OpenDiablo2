package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calc"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skilldesc"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// tooltipUnit is the minimal caster a skilldesc calc needs: the hero's level
// and the skill points per skill id. Everything else of d2skill.Unit is left
// nil (a calc that reaches it would panic, so only the fields the Env reads for
// the shipped skilldesc.txt calcs are provided: lvl, ulvl, blvl, skill(...)).
type tooltipUnit struct {
	d2skill.Unit
	level  int
	points map[int]int
}

func (u tooltipUnit) Level() int                { return u.level }
func (u tooltipUnit) SkillLevel(id int) int     { return u.points[id] }
func (u tooltipUnit) BaseSkillLevel(id int) int { return u.points[id] }
func (u tooltipUnit) Stat(string) int           { return 0 }

// descTexts are the lines a skilldesc row adds to the tooltip.
type descTexts struct {
	Current []string // mana-independent lines at the current level
	Next    []string // the "Next Level" block
	Synergy []string // the dsc3 block
}

// tooltipDamageLabel and tooltipNextLabel are fallbacks: the string keys of the
// original's "Damage:" and "Next Level:" labels were not identified.
const (
	tooltipDamageLabel = "Damage: "
	tooltipNextLabel   = "Next Level:"
)

// skillDescTexts renders the skilldesc lines of a skill. points is the
// skill's level (0 = not learned: no current block, the next-level block shows
// level 1), pts maps skill ids to the hero's points (synergies read them) and
// tr translates a string key (returning the key when unknown).
func skillDescTexts(tr func(string) string, reg *d2skill.Registry, sk *d2skill.Skill, desc d2skilldesc.Desc,
	points, heroLevel int, pts map[int]int) descTexts {
	var out descTexts

	if sk == nil {
		return out
	}

	unit := tooltipUnit{level: heroLevel, points: pts}

	evalAt := func(level int) func(string) int {
		env := d2skill.NewEnv(sk, level, unit, reg)

		return func(src string) int {
			if src == "" {
				return 0
			}

			return env.Eval(d2calc.Compile(src, d2calc.KindSkill))
		}
	}

	// UNVERIFIED: damage is the skill's own physical bonus plus elemental
	// range (weapon damage is not added). descdam selects a function in the
	// original (0x4e6fd0 is the one for index 15); here any index > 0 uses the
	// generic table-driven sum.
	damageAt := func(level int) string {
		if desc.DescDam == 0 || level < 1 {
			return ""
		}

		env := d2skill.NewEnv(sk, level, unit, reg)
		lo := int(sk.PhysMin(env, level, 0, false)>>8) + int(sk.ElemMin(env, level)>>8)
		hi := int(sk.PhysMax(env, level, 0, false)>>8) + int(sk.ElemMax(env, level)>>8)
		line, _ := d2skilldesc.Damage(tooltipDamageLabel, lo, hi)

		return line
	}

	if points >= 1 {
		ev := evalAt(points)

		if dmg := damageAt(points); dmg != "" {
			out.Current = append(out.Current, dmg)
		}

		out.Current = append(out.Current, d2skilldesc.Block(desc.Lines, tr, ev)...)
		out.Current = append(out.Current, d2skilldesc.Block(desc.Lines2, tr, ev)...)
		out.Synergy = d2skilldesc.Block(desc.Lines3, tr, ev)
	} else {
		out.Synergy = d2skilldesc.Block(desc.Lines3, tr, evalAt(1))
	}

	out.Next = d2skilldesc.NextLevel(tooltipNextLabel, desc, points, sk.MaxLvl, damageAt(points+1), tr, evalAt(points+1))

	return out
}

// skillDescLines is the skilldesc part of a tooltip: current-level lines, the
// Next Level block and the synergy lines.
func skillDescLines(asset *d2asset.AssetManager, sk *d2hero.HeroSkill, skills map[int]*d2hero.HeroSkill,
	heroLevel int) []string {
	if sk == nil || sk.SkillRecord == nil || sk.SkillDescriptionRecord == nil || asset.Records == nil {
		return nil
	}

	reg := asset.Records.SkillTable()

	pts := make(map[int]int, len(skills))
	for id, s := range skills {
		if s != nil {
			pts[id] = s.SkillPoints
		}
	}

	tr := func(key string) string {
		if key == "" {
			return ""
		}

		return asset.TranslateString(key)
	}

	t := skillDescTexts(tr, reg, reg.ByID(sk.ID), sk.Desc, sk.SkillPoints, heroLevel, pts)

	var lines []string

	lines = append(lines, t.Current...)
	lines = append(lines, t.Next...)
	lines = append(lines, t.Synergy...)

	return lines
}
