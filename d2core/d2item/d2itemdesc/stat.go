package d2itemdesc

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Stat is one item stat as a save stores it.
type Stat struct {
	ID    int
	Param int
	Value int
}

// Stat ids with special description rules (the engine builds these lines
// itself instead of using descfunc/descstr).
const (
	statEnhDmgMax = 17 // item_maxdamage_percent
	statEnhDmgMin = 18 // item_mindamage_percent
	statFireMin   = 48
	statLightMin  = 50
	statMagicMin  = 52
	statColdMin   = 54
	statPoisonMin = 57
)

// pairs are the damage stats shown as one "Adds a-b ..." line:
// min stat -> (rangeKey, singleKey, number of following stats).
var damagePairs = map[int]struct {
	rangeKey, singleKey string
	follow              int
}{
	statFireMin:   {"strModFireDamageRange", "strModFireDamage", 1},
	statLightMin:  {"strModLightningDamageRange", "strModLightningDamage", 1},
	statMagicMin:  {"strModMagicDamageRange", "strModMagicDamage", 1},
	statColdMin:   {"strModColdDamageRange", "strModColdDamage", 2},
	statPoisonMin: {"strModPoisonDamageRange", "strModPoisonDamage", 2},
}

const (
	poisonFramesPerSecond = 25
	poisonDamageDivisor   = 256
	percentOf128          = 128
)

// Merge adds up stats with the same id and parameter, the way the game
// shows one line for the sum of an item's own, runeword and socket stats.
func Merge(lists ...[]Stat) []Stat {
	type key struct{ id, param int }

	idx := make(map[key]int)

	var out []Stat

	for _, l := range lists {
		for _, s := range l {
			k := key{s.ID, s.Param}
			if i, ok := idx[k]; ok {
				out[i].Value += s.Value
				continue
			}

			idx[k] = len(out)
			out = append(out, s)
		}
	}

	return out
}

type statLine struct {
	prio int
	id   int
	text string
}

// StatLines turns merged stats into description lines, ordered by
// descpriority (highest first) as the original does.
func (t *Tables) StatLines(stats []Stat, ctx *Context) []string {
	byID := make(map[int]Stat)

	for _, s := range stats {
		if s.Param == 0 {
			byID[s.ID] = s
		}
	}

	used := make(map[int]bool)

	var lines []statLine

	// groups ("All Resistances", "to all Attributes"): every member present
	// with the same value
	lines = append(lines, t.groupLines(byID, used)...)

	// damage pairs and enhanced damage
	lines = append(lines, t.pairLines(byID, used)...)

	for _, s := range stats {
		if used[s.ID] && s.Param == 0 {
			continue
		}

		def := t.Stats[s.ID]
		if def == nil || def.Func == 0 {
			continue
		}

		text := t.describe(def, s, ctx)
		if text == "" {
			continue
		}

		lines = append(lines, statLine{def.Priority, s.ID, text})
	}

	sort.SliceStable(lines, func(i, j int) bool {
		if lines[i].prio != lines[j].prio {
			return lines[i].prio > lines[j].prio
		}

		return lines[i].id < lines[j].id
	})

	out := make([]string, 0, len(lines))
	seen := make(map[string]bool)

	for _, l := range lines {
		// stats like mindamage / secondary_mindamage share one text
		if seen[l.text] {
			continue
		}

		seen[l.text] = true

		out = append(out, l.text)
	}

	return out
}

func (t *Tables) groupLines(byID map[int]Stat, used map[int]bool) []statLine {
	members := make(map[int][]*StatDef)

	for _, d := range t.Stats {
		if d.Grp != 0 {
			members[d.Grp] = append(members[d.Grp], d)
		}
	}

	var out []statLine

	groups := make([]int, 0, len(members))
	for g := range members {
		groups = append(groups, g)
	}

	sort.Ints(groups)

	for _, g := range groups {
		ms := members[g]

		val, ok := 0, true

		prio := 0

		for i, d := range ms {
			s, has := byID[d.ID]
			if !has || (i > 0 && s.Value != val) {
				ok = false
				break
			}

			val = s.Value
			if d.Priority > prio {
				prio = d.Priority
			}
		}

		if !ok || len(ms) < 2 {
			continue
		}

		d := ms[0]
		for _, m := range ms {
			if m.ID < d.ID {
				d = m
			}

			used[m.ID] = true
		}

		text := t.format(d.GrpFunc, d.GrpVal, d.GrpStrPos, d.GrpStrNeg, d.GrpStr, val, 0, d)
		out = append(out, statLine{prio, d.ID, text})
	}

	return out
}

func (t *Tables) pairLines(byID map[int]Stat, used map[int]bool) []statLine {
	var out []statLine

	if mx, hasMax := byID[statEnhDmgMax]; hasMax {
		if mn, hasMin := byID[statEnhDmgMin]; hasMin && mn.Value == mx.Value {
			used[statEnhDmgMax], used[statEnhDmgMin] = true, true

			p := maxInt(t.Stats[statEnhDmgMax].Priority, t.Stats[statEnhDmgMin].Priority)
			out = append(out, statLine{p, statEnhDmgMax, fmt.Sprintf("%s%d%% %s", sign(mx.Value), mx.Value, t.enhancedDamageText())})
		} else {
			used[statEnhDmgMax] = true
			out = append(out, statLine{t.Stats[statEnhDmgMax].Priority, statEnhDmgMax,
				fmt.Sprintf("%s%d%% %s", sign(mx.Value), mx.Value, t.tr(t.Stats[statEnhDmgMax].StrPos))})
		}
	}

	if mn, ok := byID[statEnhDmgMin]; ok && !used[statEnhDmgMin] {
		used[statEnhDmgMin] = true
		out = append(out, statLine{t.Stats[statEnhDmgMin].Priority, statEnhDmgMin,
			fmt.Sprintf("%s%d%% %s", sign(mn.Value), mn.Value, t.tr(t.Stats[statEnhDmgMin].StrPos))})
	}

	ids := make([]int, 0, len(damagePairs))
	for id := range damagePairs {
		ids = append(ids, id)
	}

	sort.Ints(ids)

	for _, id := range ids {
		p := damagePairs[id]

		mn, ok1 := byID[id]
		mx, ok2 := byID[id+1]

		if !ok1 || !ok2 {
			continue
		}

		prio := 0
		if d := t.Stats[id]; d != nil {
			prio = d.Priority
		}

		used[id], used[id+1] = true, true

		var text string

		switch id {
		case statPoisonMin:
			ln := byID[id+2].Value
			used[id+2] = true
			// poison damage is stored per frame (x256) over `ln` frames
			lo, hi := mn.Value*ln/poisonDamageDivisor, mx.Value*ln/poisonDamageDivisor
			secs := ln / poisonFramesPerSecond

			if lo == hi {
				text = fmt.Sprintf(t.tr(p.singleKey), lo, secs)
			} else {
				text = fmt.Sprintf(t.tr(p.rangeKey), lo, hi, secs)
			}
		default:
			if id == statColdMin {
				used[id+2] = true // freeze length: not part of the line
			}

			if mn.Value == mx.Value {
				text = fmt.Sprintf(t.tr(p.singleKey), mn.Value)
			} else {
				text = fmt.Sprintf(t.tr(p.rangeKey), mn.Value, mx.Value)
			}
		}

		out = append(out, statLine{prio, id, text})
	}

	return out
}

// enhancedDamageText is "Enhanced Damage": the string tables only have the
// "Enhanced Maximum Damage" / "Enhanced Minimum Damage" halves, the engine
// joins them when both bonuses are equal.
func (t *Tables) enhancedDamageText() string {
	s := t.tr(t.Stats[statEnhDmgMax].StrPos)

	return strings.Replace(s, " Maximum", "", 1)
}

func sign(v int) string {
	if v >= 0 {
		return "+"
	}

	return "" // the number carries its own minus
}

// describe renders one stat.
func (t *Tables) describe(def *StatDef, s Stat, ctx *Context) string {
	v := s.Value

	// "per character level" stats store value / 2^OpParam per level
	if def.Op >= 2 && def.Op <= 5 && def.Func >= 6 && def.Func <= 9 {
		clvl := 1
		if ctx != nil && ctx.CharLevel > 0 {
			clvl = ctx.CharLevel
		}

		v = (v * clvl) >> uint(def.OpParam)
	}

	return t.format(def.Func, def.Val, def.StrPos, def.StrNeg, def.Str2, v, s.Param, def)
}

func (t *Tables) skillName(id int) string {
	if d := t.Skills[id]; d != nil {
		return t.tr(d.Name)
	}

	return "Skill " + strconv.Itoa(id)
}

func (t *Tables) class(i int) *ClassDef {
	if i < 0 || i >= len(t.Classes) {
		return nil
	}

	return &t.Classes[i]
}

// format implements the descfunc / descval rules of itemstatcost.txt.
func (t *Tables) format(fn, val int, strPos, strNeg, str2 string, v, param int, def *StatDef) string {
	key := strPos
	if v < 0 && strNeg != "" {
		key = strNeg
	}

	text := t.tr(key)

	withVal := func(num string) string {
		switch val {
		case 1:
			return num + " " + text
		case 2:
			return text + " " + num
		}

		return text
	}

	pct := func(x int) int { return x * 100 / percentOf128 }
	add2 := func(s string) string {
		if str2 != "" {
			return s + " " + t.tr(str2)
		}

		return s
	}

	switch fn {
	case 1, 12:
		return withVal(sign(v) + strconv.Itoa(v))
	case 2:
		return withVal(strconv.Itoa(v) + "%")
	case 3:
		return withVal(strconv.Itoa(v))
	case 4:
		return withVal(sign(v) + strconv.Itoa(v) + "%")
	case 5:
		return withVal(strconv.Itoa(pct(v)) + "%")
	case 6:
		return add2(withVal(sign(v) + strconv.Itoa(v)))
	case 7:
		return add2(withVal(strconv.Itoa(v) + "%"))
	case 8:
		return add2(withVal(sign(v) + strconv.Itoa(v) + "%"))
	case 9:
		return add2(withVal(strconv.Itoa(v)))
	case 10:
		return add2(withVal(strconv.Itoa(pct(v)) + "%"))
	case 11: // repairs durability
		if v >= 100 {
			return fmt.Sprintf(t.tr("ModStre9t"), v/100)
		}

		if v > 0 {
			return fmt.Sprintf(t.tr("ModStre9u"), 1, 100/v)
		}

		return ""
	case 13: // +v to <Class> Skill Levels
		if c := t.class(param); c != nil {
			return sign(v) + strconv.Itoa(v) + " " + t.tr(c.AllSkills)
		}

		return ""
	case 14: // +v to <skill tab> Skills (<Class> Only)
		tab, cls := param&7, (param>>3)&0x1fff

		c := t.class(cls)
		if c == nil || tab >= len(c.SkillTabs) {
			return ""
		}

		return fmt.Sprintf(t.tr(c.SkillTabs[tab]), v) + " " + t.tr(c.ClassOnly)
	case 15: // chance to cast <skill> on <event>
		return fmt.Sprintf(t.tr(key), v, param&0x3f, t.skillName(param>>6))
	case 16: // aura when equipped
		return fmt.Sprintf(t.tr(key), v, t.skillName(param))
	case 19:
		return fmt.Sprintf(t.tr(key), v)
	case 20:
		return withVal(strconv.Itoa(-v) + "%")
	case 21:
		return withVal(strconv.Itoa(-v))
	case 22, 23:
		return withVal(strconv.Itoa(v) + "%")
	case 24: // charges
		return fmt.Sprintf("Level %d %s ", param&0x3f, t.skillName(param>>6)) +
			fmt.Sprintf(t.tr(key), v&0xff, (v>>8)&0xff)
	case 27, 28: // +v to <skill> (<Class> Only)
		s := sign(v) + strconv.Itoa(v) + " to " + t.skillName(param)

		if fn == 27 {
			if d := t.Skills[param]; d != nil {
				if c := t.class(d.Class); c != nil {
					s += " " + t.tr(c.ClassOnly)
				}
			}
		}

		return s
	case 17, 18: // "(Increases near <time>)" stats: the time of day is not modelled
		if fn == 18 {
			return withVal(strconv.Itoa(v) + "%")
		}

		return withVal(sign(v) + strconv.Itoa(v))
	}

	return ""
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}
