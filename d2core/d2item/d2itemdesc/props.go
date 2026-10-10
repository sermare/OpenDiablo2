package d2itemdesc

import (
	"strconv"
	"strings"
)

// property function ids of properties.txt (the "func" columns)
const (
	pfValue       = 1  // value to stat
	pfArmorPct    = 2  // item_armor_percent
	pfRepeat      = 3  // repeat the previous function with min and max
	pfDmgMin      = 5  // dmg-min
	pfDmgMax      = 6  // dmg-max
	pfDmgPct      = 7  // dmg%
	pfSpeed       = 8  // swing/move/cast/balance/block speed
	pfRepeatParam = 9  // repeat with param, min and max
	pfSkillTab    = 10 // skilltab
	pfProc        = 11 // att-skill, hit-skill ...
	pfDurability  = 13
	pfMin         = 15 // damage min
	pfMax         = 16 // damage max
	pfLength      = 17 // damage length (param)
	pfIndestruct  = 20
	pfClassSkills = 21
	pfSingleSkill = 22
)

const (
	statMinDamage   = 21
	statMaxDamage   = 22
	statIndestruct  = 152
	skillLevelShift = 6
)

// skillByCode finds a skill id from the internal skill name of skills.txt.
func (t *Tables) skillByCode(name string) (int, bool) {
	if n, err := strconv.Atoi(name); err == nil {
		return n, true
	}

	for id, s := range t.Skills {
		if strings.EqualFold(s.Code, name) {
			return id, true
		}
	}

	return 0, false
}

// EvalProp turns a property of the item tables (a gem effect, a set bonus,
// ...) into stats. Only the fixed value (the maximum) is used: gems, runes
// and set bonuses do not roll. The property functions that are not needed
// for them (charges, random skills, states) produce nothing.
func (t *Tables) EvalProp(p PropSpec) []Stat {
	slots := t.Props[p.Code]
	val := p.Max

	var (
		out  []Stat
		last int
	)

	for _, sl := range slots {
		fn := sl.Func
		if fn == pfRepeat || fn == pfRepeatParam {
			fn = last
		} else {
			last = fn
		}

		def := t.StatByName[sl.Stat]

		switch fn {
		case pfValue, pfArmorPct, pfSpeed, pfDurability:
			if def != nil {
				out = append(out, Stat{ID: def.ID, Value: val})
			}
		case pfDmgMin:
			out = append(out, Stat{ID: statMinDamage, Value: val})
		case pfDmgMax:
			out = append(out, Stat{ID: statMaxDamage, Value: val})
		case pfDmgPct:
			out = append(out, Stat{ID: statEnhDmgMax, Value: val}, Stat{ID: statEnhDmgMin, Value: val})
		case pfMin:
			if def != nil {
				out = append(out, Stat{ID: def.ID, Value: p.Min})
			}
		case pfMax:
			if def != nil {
				out = append(out, Stat{ID: def.ID, Value: p.Max})
			}
		case pfLength:
			if def != nil {
				n, _ := strconv.Atoi(p.Param)
				out = append(out, Stat{ID: def.ID, Value: n})
			}
		case pfSkillTab:
			if def != nil {
				n, _ := strconv.Atoi(p.Param) // class*3 + tab
				out = append(out, Stat{ID: def.ID, Param: n%3 | (n/3)<<3, Value: val})
			}
		case pfClassSkills:
			if def != nil {
				out = append(out, Stat{ID: def.ID, Param: t.classOfProp(p.Code), Value: val})
			}
		case pfSingleSkill:
			if def != nil {
				if id, ok := t.skillByCode(p.Param); ok {
					out = append(out, Stat{ID: def.ID, Param: id, Value: val})
				}
			}
		case pfProc:
			if def != nil {
				if id, ok := t.skillByCode(p.Param); ok {
					out = append(out, Stat{ID: def.ID, Param: id<<skillLevelShift | p.Max, Value: p.Min})
				}
			}
		case pfIndestruct:
			out = append(out, Stat{ID: statIndestruct, Value: 1})
		}
	}

	return out
}

// classOfProp is the class index of the class skill properties (ama, sor ...).
func (t *Tables) classOfProp(code string) int {
	if c, ok := classCodes[code]; ok {
		return c
	}

	return 0
}
