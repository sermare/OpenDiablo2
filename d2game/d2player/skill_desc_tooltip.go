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

// elemType is the skills.txt EType as the numeric element of the label helper
// 0x4e3fa0: 1 fire, 2 lightning, 3 magic, 4 cold, 5 poison, 0 none.
func elemType(s string) int {
	return map[string]int{"fire": 1, "ltng": 2, "mag": 3, "cold": 4, "pois": 5}[s]
}

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

// labelFallback is the English text of the fixed labels the damage rows use
// (string ids 0x10a0, 0x10a1..0x10a4, 0x10c3, 0x10a6, 0x10b3, 0x10a9, 0x10aa,
// 0x10b0, 0x10c6, 0x10ab, 0x10ac, VERIFIED against string.tbl).
var labelFallback = map[string]string{
	"StrSkill4": "Damage: ", "StrSkill5": "Fire Damage: ", "StrSkill6": "Cold Damage: ",
	"StrSkill7": "Lightning Damage: ", "StrSkill8": "Poison Damage: ", "StrSkill39": "Magic Damage: ",
	"StrSkill10": "To Attack Rating: ", "StrSkill23": " percent", "StrSkill13": "Cold Length: ",
	"StrSkill14": "Poison Length: ", "StrSkill20": "Duration: ", "StrSkill42": "Life: ",
	"StrSkill15": " second", "StrSkill16": " seconds",
}

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
	points, heroLevel int, pts map[int]int, mana func(level int) (string, bool), weapon [2]int) descTexts {
	var out descTexts

	if sk == nil {
		return out
	}

	// Rows print fixed labels (damage, element, duration, life ...) by string
	// key; fall back to the original English when a table lacks the key.
	rawTr := tr
	tr = func(key string) string {
		if v := rawTr(key); v != key {
			return v
		}

		if fb, ok := labelFallback[key]; ok {
			return fb
		}

		return key
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

	// ctxAt supplies the values of the kind 1, 8-11 rows at a level. Damage is
	// not a separate line: the original prints it where a row of kind 8-11
	// sits (0x4e9380, 0x4e9140, 0x4e9080, 0x4eab70). Kind 13 (summon life)
	// has no hook here: the summoned monster's life table is not reachable
	// from the tooltip, so those rows give no line.
	ctxAt := func(level int) *d2skilldesc.Ctx {
		env := d2skill.NewEnv(sk, level, unit, reg)
		c := &d2skilldesc.Ctx{
			ToHit: func() int { return sk.ToHitBonus(env, level) },
			Phys: func() (int, int) {
				return int(sk.PhysMin(env, level, weapon[0], true) >> 8),
					int(sk.PhysMax(env, level, weapon[1], true) >> 8)
			},
			Elem: func() (int, int, int) {
				return int(sk.ElemMin(env, level) >> 8), int(sk.ElemMax(env, level) >> 8), elemType(sk.EType)
			},
			ElemLen: func() int { return sk.ElemLen(env, level) },
		}

		if mana != nil {
			c.Mana = func() (string, bool) { return mana(level) }
		}

		return c
	}

	// 0x4ec6d0 evaluates the dsc2 and descline blocks at max(level, 1).
	shown := points
	if shown < 1 {
		shown = 1
	}

	out.Top = d2skilldesc.BlockCtx(desc.Lines2, tr, evalAt(shown), ctxAt(shown))

	if points >= 1 {
		body := d2skilldesc.BlockCtx(desc.Lines, tr, evalAt(points), ctxAt(points))

		if len(body) > 0 {
			head := d2skilldesc.CurrentLevel(label(tr, keyCurrentLevel, "Current Skill Level: "), points)
			out.Current = append([]string{head}, body...)
		}
	}

	// Both builders evaluate the dsc3 block at level+1 (0x4ec2a0, 0x4ec720).
	out.Synergy = d2skilldesc.BlockCtx(desc.Lines3, tr, evalAt(points+1), ctxAt(points+1))

	nextLabel := label(tr, keyNextLevel, "Next Level")
	if points < 1 {
		nextLabel = label(tr, keyFirstLevel, "First Level")
	}

	out.Next = d2skilldesc.NextLevel(nextLabel, desc, points, sk.MaxLvl, tr, evalAt(points+1), ctxAt(points+1))

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

	// The equipped weapon is not reachable from here yet, so the weapon part
	// of kind-9 damage (SrcDam/128 of the weapon) is 0 in the live tooltip.
	t := skillDescTexts(tr, reg, reg.ByID(sk.ID), sk.Desc, sk.SkillPoints, heroLevel, pts, mana, [2]int{})

	var lines []string

	lines = append(lines, t.Top...)
	lines = append(lines, t.Current...)
	lines = append(lines, t.Next...)
	lines = append(lines, t.Synergy...)

	return lines
}
