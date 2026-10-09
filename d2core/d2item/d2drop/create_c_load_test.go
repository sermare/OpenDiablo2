package d2drop

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Loaders of the slice C tables from the extracted game tables (D2_TABLES).
// The game drops the "Expansion" separator rows of its tables, so do these.

// gameTab is a table read like the game does, with the information which
// rows come after the "Expansion" separator.
type gameTab struct {
	*tsv
	exp []bool
}

// cReadTable reads a table without the rows whose first column is
// "Expansion". Tables without a version column take version 100 for the rows
// after the separator.
func cReadTable(t *testing.T, path string) *gameTab {
	t.Helper()

	tab := readTSV(t, path)
	out := &gameTab{tsv: tab}
	rows := tab.rows[:0:0]
	exp := false

	for _, r := range tab.rows {
		if len(r) > 0 && strings.TrimSpace(r[0]) == "Expansion" {
			exp = true

			continue
		}

		rows = append(rows, r)
		out.exp = append(out.exp, exp)
	}

	tab.rows = rows

	return out
}

var numRe = regexp.MustCompile(`^-?\d+$`)

// loadPropTables reads Properties.txt, ItemStatCost.txt (valshift) and the
// skill levels of skills.txt.
func loadPropTables(t *testing.T) *PropTables {
	t.Helper()

	root := d2Tables(t)
	pt := &PropTables{ByCode: map[string]int{}, SkillParamShift: 6, SkillParamMask: 0x3f}

	isc := readTSV(t, filepath.Join(root, "ItemStatCost.txt"))
	statID := map[string]int{}

	for i, r := range isc.rows {
		id := isc.n(r, "ID")
		if id != i {
			t.Fatalf("ItemStatCost row %d has ID %d", i, id)
		}

		statID[strings.ToLower(isc.s(r, "Stat"))] = id

		pt.ValShift = append(pt.ValShift, isc.n(r, "ValShift"))
	}

	props := cReadTable(t, tablePath(t, "Properties.txt"))

	for i, r := range props.rows {
		row := PropRow{Code: props.s(r, "code")}

		for k := 0; k < 7; k++ {
			n := strconv.Itoa(k + 1)
			row.Set[k] = props.n(r, "set"+n)
			row.Val[k] = props.n(r, "val"+n)
			row.Func[k] = props.n(r, "func"+n)
			row.Stat[k] = -1

			if name := strings.ToLower(props.s(r, "stat"+n)); name != "" {
				id, ok := statID[name]
				if !ok {
					t.Fatalf("Properties %s: unknown stat %q", row.Code, name)
				}

				row.Stat[k] = id
			}
		}

		pt.Props = append(pt.Props, row)

		if _, dup := pt.ByCode[strings.ToLower(row.Code)]; !dup {
			pt.ByCode[strings.ToLower(row.Code)] = i
		}
	}

	sk := readTSV(t, filepath.Join(root, "skills", "patch_d2", "skills.txt"))

	for i, r := range sk.rows {
		if sk.n(r, "Id") != i {
			t.Fatalf("skills row %d has Id %d", i, sk.n(r, "Id"))
		}

		pt.Skills = append(pt.Skills, SkillInfo{ReqLevel: sk.n(r, "reqlevel"), MaxLevel: sk.n(r, "maxlvl")})
	}

	return pt
}

// cSkillByName maps skill names to ids (for the param columns).
func cSkillByName(t *testing.T) map[string]int {
	t.Helper()

	sk := readTSV(t, filepath.Join(d2Tables(t), "skills", "patch_d2", "skills.txt"))
	out := map[string]int{}

	for i, r := range sk.rows {
		out[strings.ToLower(sk.s(r, "skill"))] = i
	}

	return out
}

// cPropInst parses one prop/par/min/max column group of a row.
func cPropInst(t *testing.T, pt *PropTables, skills map[string]int, tab *gameTab, r []string, prop, par, min, max string) PropInst {
	t.Helper()

	code := strings.ToLower(tab.s(r, prop))
	pi := PropInst{Prop: -1}

	if code == "" {
		return pi
	}

	idx, ok := pt.ByCode[code]
	if !ok {
		return pi
	}

	pi.Prop = idx
	pi.Min = tab.n(r, min)
	pi.Max = tab.n(r, max)

	switch p := tab.s(r, par); {
	case p == "":
	case numRe.MatchString(p):
		pi.Param, _ = strconv.Atoi(p)
	default:
		id, ok := skills[strings.ToLower(p)]
		if !ok {
			t.Fatalf("%s: unknown skill %q", code, p)
		}

		pi.Param = id
	}

	return pi
}

// loadUniqueTables reads UniqueItems.txt, SetItems.txt and Sets.txt.
func loadUniqueTables(t *testing.T, pt *PropTables) *UniqueTables {
	t.Helper()

	skills := cSkillByName(t)
	ut := &UniqueTables{}

	u := cReadTable(t, tablePath(t, "UniqueItems.txt"))

	for _, r := range u.rows {
		it := UniqueItem{
			Name: u.s(r, "index"), Version: u.n(r, "version"), Enabled: u.n(r, "enabled") != 0,
			Ladder: u.n(r, "ladder") != 0, NoLimit: u.n(r, "nolimit") != 0, Rarity: u.n(r, "rarity"),
			Lvl: u.n(r, "lvl"), LvlReq: u.n(r, "lvl req"), Code: u.s(r, "code"),
		}

		for k := range it.Props {
			n := strconv.Itoa(k + 1)
			it.Props[k] = cPropInst(t, pt, skills, u, r, "prop"+n, "par"+n, "min"+n, "max"+n)
		}

		ut.Uniques = append(ut.Uniques, it)
	}

	sets := cReadTable(t, tablePath(t, "Sets.txt"))
	setRow := map[string]int{}

	for i, r := range sets.rows {
		ut.Sets = append(ut.Sets, SetDef{Name: sets.s(r, "index"), Version: sets.n(r, "version")})
		setRow[sets.s(r, "index")] = i
	}

	s := cReadTable(t, tablePath(t, "SetItems.txt"))

	for i, r := range s.rows {
		it := SetItem{
			Name: s.s(r, "index"), Lvl: s.n(r, "lvl"), LvlReq: s.n(r, "lvl req"),
			Rarity: s.n(r, "rarity"), Code: s.s(r, "item"), AddFunc: s.n(r, "add func"),
		}

		if s.exp[i] {
			it.Version = 100
		}

		si, ok := setRow[s.s(r, "set")]
		if !ok {
			t.Fatalf("set item %q: unknown set %q", it.Name, s.s(r, "set"))
		}

		it.Set = si
		ut.Sets[si].Items = append(ut.Sets[si].Items, i)

		for k := range it.Props {
			n := strconv.Itoa(k + 1)
			it.Props[k] = cPropInst(t, pt, skills, s, r, "prop"+n, "par"+n, "min"+n, "max"+n)
		}

		for k := range it.AProps {
			n := fmt.Sprintf("%d%c", k/2+1, 'a'+k%2)
			it.AProps[k] = cPropInst(t, pt, skills, s, r, "aprop"+n, "apar"+n, "amin"+n, "amax"+n)
		}

		ut.SetItems = append(ut.SetItems, it)
	}

	return ut
}

// loadTestAffixes returns the property lists of the combined affix table of
// the game: magic suffixes, then magic prefixes, then automagic, ids from 1.
func loadTestAffixes(t *testing.T, pt *PropTables) [][3]PropInst {
	t.Helper()

	skills := cSkillByName(t)

	var out [][3]PropInst

	for _, name := range []string{"MagicSuffix.txt", "MagicPrefix.txt", "AutoMagic.txt"} {
		tab := cReadTable(t, tablePath(t, name))

		for _, r := range tab.rows {
			var a [3]PropInst

			for k := range a {
				n := strconv.Itoa(k + 1)
				a[k] = cPropInst(t, pt, skills, tab, r, "mod"+n+"code", "mod"+n+"param", "mod"+n+"min", "mod"+n+"max")
			}

			out = append(out, a)
		}
	}

	return out
}

// loadTestQuality returns the two properties of each QualityItems row.
func loadTestQuality(t *testing.T, pt *PropTables) [][2]PropInst {
	t.Helper()

	skills := cSkillByName(t)
	tab := cReadTable(t, tablePath(t, "QualityItems.txt"))

	var out [][2]PropInst

	for _, r := range tab.rows {
		if tab.s(r, "nummods") == "" {
			continue
		}

		var q [2]PropInst

		for k := range q {
			n := strconv.Itoa(k + 1)
			q[k] = cPropInst(t, pt, skills, tab, r, "mod"+n+"code", "mod"+n+"param", "mod"+n+"min", "mod"+n+"max")
		}

		out = append(out, q)
	}

	return out
}
