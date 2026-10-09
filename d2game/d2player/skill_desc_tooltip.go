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

// descTexts are the lines a skilldesc row adds to the tooltip, in on-screen
// order (VERIFIED, see d2skilldesc.Desc): Top (the dsc2 block), Current (the
// "Current Skill Level: n" label and the descline block), Next (the "Next
// Level" or "First Level" label and the block of level+1) and Synergy (dsc3).
type descTexts struct {
	Top     []string
	Current []string
	Next    []string
	Synergy []string
}

// String-table keys of the labels. VERIFIED from the tooltip builders at
// 0x4ec180/0x4ec6d0 (string ids 0x109d, 0x10ad, 0x109e) and the damage line at
// 0x4e9140 (0x10a0); the English text is only the fallback.
const (
	keyNextLevel    = "StrSkill1"  // "Next Level" (no colon)
	keyFirstLevel   = "StrSkill17" // "First Level", shown instead of Next Level at 0 points
	keyCurrentLevel = "StrSkill2"  // "Current Skill Level: "
	keyDamage       = "StrSkill4"  // "Damage: "
)

// label translates a fixed label key, falling back to the original English
// text when the string tables do not have it.
func label(tr func(string) string, key, fallback string) string {
	if v := tr(key); v != "" && v != key {
		return v
	}

	return fallback
}

// skillDescTexts renders the skilldesc lines of a skill. points is the
// skill's level (0 = not learned: no current block, the next-level block shows
// level 1), pts maps skill ids to the hero's points (synergies read them), tr
// translates a string key (returning the key when unknown) and mana renders the
// kind-1 line for a level (nil: no mana lines).
func skillDescTexts(tr func(string) string, reg *d2skill.Registry, sk *d2skill.Skill, desc d2skilldesc.Desc,
	points, heroLevel int, pts map[int]int, mana func(level int) (string, bool)) descTexts {
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

	manaAt := func(level int) func() (string, bool) {
		if mana == nil {
			return nil
		}

		return func() (string, bool) { return mana(level) }
	}

	// Damage: the original has no single damage line; damage comes from the
	// rows of kinds 8-11 and 13 at their row position (0x4e9140, 0x4e8f90,
	// 0x4e8ed0), which are not modelled. UNVERIFIED: this approximates them
	// with the skill's own physical bonus plus elemental range (weapon damage,
	// SrcDam/128, is not added) placed first in the block. The label key
	// StrSkill4 and the "a-b" wording are VERIFIED.
	damageAt := func(level int) string {
		if desc.DescDam == 0 || level < 1 {
			return ""
		}

		env := d2skill.NewEnv(sk, level, unit, reg)
		lo := int(sk.PhysMin(env, level, 0, false)>>8) + int(sk.ElemMin(env, level)>>8)
		hi := int(sk.PhysMax(env, level, 0, false)>>8) + int(sk.ElemMax(env, level)>>8)
		line, _ := d2skilldesc.Damage(label(tr, keyDamage, "Damage: "), lo, hi)

		return line
	}

	// 0x4ec6d0 evaluates the dsc2 and descline blocks at max(level, 1).
	shown := points
	if shown < 1 {
		shown = 1
	}

	out.Top = d2skilldesc.Block(desc.Lines2, tr, evalAt(shown))

	if points >= 1 {
		body := d2skilldesc.BlockMana(desc.Lines, tr, evalAt(points), manaAt(points))
		if dmg := damageAt(points); dmg != "" {
			body = append([]string{dmg}, body...)
		}

		if len(body) > 0 {
			head := d2skilldesc.CurrentLevel(label(tr, keyCurrentLevel, "Current Skill Level: "), points)
			out.Current = append([]string{head}, body...)
		}
	}

	// Both builders evaluate the dsc3 block at level+1 (0x4ec2a0, 0x4ec720).
	out.Synergy = d2skilldesc.Block(desc.Lines3, tr, evalAt(points+1))

	nextLabel := label(tr, keyNextLevel, "Next Level")
	if points < 1 {
		nextLabel = label(tr, keyFirstLevel, "First Level")
	}

	out.Next = d2skilldesc.NextLevel(nextLabel, desc, points, sk.MaxLvl, damageAt(points+1), tr,
		evalAt(points+1), manaAt(points+1))

	return out
}

// skillDescLines is the skilldesc part of a tooltip: the dsc2 lines, the
// current-level block, the Next Level block and the synergy lines.
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

	var mana func(int) (string, bool)

	if manaLabel := tr(sk.ManaKey); sk.ManaKey != "" && manaLabel != sk.ManaKey {
		mana = func(level int) (string, bool) {
			return d2skilldesc.ManaCost(manaLabel, sk.SkillRecord.PipelineSkill().ManaCost(level))
		}
	}

	t := skillDescTexts(tr, reg, reg.ByID(sk.ID), sk.Desc, sk.SkillPoints, heroLevel, pts, mana)

	var lines []string

	lines = append(lines, t.Top...)
	lines = append(lines, t.Current...)
	lines = append(lines, t.Next...)
	lines = append(lines, t.Synergy...)

	return lines
}
