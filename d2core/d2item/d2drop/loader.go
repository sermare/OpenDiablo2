package d2drop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// TableSource gives the raw text of a game table by its bare file name
// ("weapons.txt", "UniqueItems.txt", "skills.txt" ...). The game reads the
// tables from the MPQs, the first of patch_d2, d2exp, d2data that holds the
// file wins; the source must return the effective version.
type TableSource interface {
	Table(name string) ([]byte, error)
}

// DirTables reads the extracted game tables of a folder: weapons.txt, armor.txt,
// misc.txt, ItemTypes.txt and ItemStatCost.txt directly in it, skills.txt in
// skills/patch_d2 and every other table in itemgen/{patch_d2,d2exp,d2data}.
// The tests and tools use it; the game reads its MPQs.
type DirTables struct {
	Root string
}

func (d DirTables) Table(name string) ([]byte, error) {
	rootLevel := map[string]bool{
		"ItemTypes.txt": true, "weapons.txt": true, "armor.txt": true, "misc.txt": true, "ItemStatCost.txt": true,
	}

	var cands []string

	switch {
	case rootLevel[name]:
		cands = []string{filepath.Join(d.Root, name)}
	case name == "skills.txt":
		cands = []string{filepath.Join(d.Root, "skills", "patch_d2", name)}
	default:
		for _, mpq := range []string{"patch_d2", "d2exp", "d2data"} {
			cands = append(cands, filepath.Join(d.Root, "itemgen", mpq, name))
		}
	}

	var err error

	for _, p := range cands {
		var raw []byte

		if raw, err = os.ReadFile(p); err == nil {
			return raw, nil
		}
	}

	return nil, err
}

// tb is the part of testing.TB the loaders use, so the same code serves the
// tests and the game. LoadCreator turns failures into an error.
type tb interface {
	Helper()
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

type loadFailure struct{ msg string }

type failer struct{}

func (failer) Helper() {}

func (failer) Fatal(args ...interface{}) { panic(loadFailure{fmt.Sprint(args...)}) }

func (failer) Fatalf(format string, args ...interface{}) {
	panic(loadFailure{fmt.Sprintf(format, args...)})
}

// ErrLoad wraps a table loading failure.
var ErrLoad = errors.New("d2drop: loading tables")

// LoadCreator builds the item Creator from the game tables: weapons, armor,
// misc, ItemTypes, ItemStatCost, Properties, skills, QualityItems,
// LowQualityItems, ItemRatio, MagicPrefix/Suffix, AutoMagic, RarePrefix/Suffix,
// UniqueItems, SetItems and Sets.
func LoadCreator(src TableSource) (c *Creator, err error) {
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprint(r)
			if lf, ok := r.(loadFailure); ok {
				msg = lf.msg
			}

			c, err = nil, fmt.Errorf("%w: %s", ErrLoad, msg)
		}
	}()

	return loadCreatorFrom(failer{}, src), nil
}

func loadCreatorFrom(t tb, src TableSource) *Creator {
	pt := loadPropTablesFrom(t, src)
	pt.Affix = loadTestAffixesFrom(t, src, pt)
	pt.Quality = loadTestQualityFrom(t, src, pt)

	return &Creator{
		Items: loadItemTablesFrom(t, src), Quality: loadQualityTablesFrom(t, src), Affixes: loadAffixTablesFrom(t, src),
		Props: pt, Uniques: loadUniqueTablesFrom(t, src, pt),
		Runes: loadRuneTablesFrom(t, src, pt),
	}
}

// loadRuneTablesFrom reads Runes.txt.
func loadRuneTablesFrom(t tb, src TableSource, pt *PropTables) *RuneTables {
	t.Helper()

	tab := cReadTable(t, src, "Runes.txt")
	skills := cSkillByName(t, src)
	out := &RuneTables{}

	for _, r := range tab.rows {
		if tab.s(r, "Name") == "" {
			continue
		}

		w := Runeword{Name: tab.s(r, "Name"), Complete: tab.n(r, "complete") == 1}

		for k := 1; k <= 6; k++ {
			w.IType = append(w.IType, tab.s(r, fmt.Sprintf("itype%d", k)))

			if code := tab.s(r, fmt.Sprintf("rune%d", k)); code != "" {
				w.Runes = append(w.Runes, code)
			}
		}

		for k := 1; k <= 3; k++ {
			w.EType = append(w.EType, tab.s(r, fmt.Sprintf("etype%d", k)))
		}

		for k := range w.Props {
			n := strconv.Itoa(k + 1)
			w.Props[k] = cPropInst(t, pt, skills, tab, r, "t1code"+n, "t1param"+n, "t1min"+n, "t1max"+n)
		}

		out.Words = append(out.Words, w)
	}

	return out
}

type tsv struct {
	cols map[string]int
	rows [][]string
}

func readTab(t tb, src TableSource, name string) *tsv {
	t.Helper()

	raw, err := src.Table(name)
	if err != nil {
		t.Fatalf("missing table %s: %v", name, err)
	}

	return readTSV2(t, raw)
}

func readTSV2(t tb, raw []byte) *tsv {
	t.Helper()

	lines := strings.Split(strings.ReplaceAll(string(raw), "\r", ""), "\n")
	out := &tsv{cols: map[string]int{}}

	for i, h := range strings.Split(lines[0], "\t") {
		if _, dup := out.cols[strings.ToLower(h)]; !dup {
			out.cols[strings.ToLower(h)] = i
		}
	}

	for _, l := range lines[1:] {
		if l != "" {
			out.rows = append(out.rows, strings.Split(l, "\t"))
		}
	}

	return out
}

func (d *tsv) s(row []string, col string) string {
	i, ok := d.cols[strings.ToLower(col)]
	if !ok || i >= len(row) {
		return ""
	}

	return strings.TrimSpace(row[i])
}

func (d *tsv) n(row []string, col string) int {
	v, _ := strconv.Atoi(d.s(row, col))

	return v
}

var heroIndex = map[string]int{"ama": 0, "sor": 1, "nec": 2, "pal": 3, "bar": 4, "dru": 5, "ass": 6}

// loadItemTables reads weapons.txt, armor.txt, misc.txt and ItemTypes.txt.
func loadItemTablesFrom(t tb, src TableSource) *ItemTables {
	t.Helper()

	it := &ItemTables{ByCode: map[string]*BaseItem{}, Types: map[string]*ItemType{}}

	types := readTab(t, src, "ItemTypes.txt")

	skipped := 0 // the "Expansion" separator row is not part of the game's table

	for i, r := range types.rows {
		if types.s(r, "ItemType") == "Expansion" {
			skipped++

			continue
		}

		i -= skipped

		code := types.s(r, "Code")
		if code == "" && i != 0 {
			continue
		}

		ty := &ItemType{
			Index: i, Code: code, Equiv1: types.s(r, "Equiv1"), Equiv2: types.s(r, "Equiv2"),
			Normal: types.n(r, "Normal") == 1, Magic: types.n(r, "Magic") == 1, Rare: types.n(r, "Rare") == 1,
			Charm: types.n(r, "Charm") == 1, Gem: types.n(r, "Gem") == 1, Beltable: types.n(r, "Beltable") == 1,
			MaxSock1: types.n(r, "MaxSock1"), MaxSock25: types.n(r, "MaxSock25"), MaxSock40: types.n(r, "MaxSock40"),
			TreasureClass: types.n(r, "TreasureClass") == 1, Rarity: types.n(r, "Rarity"),
			Class: -1, VarInvGfx: types.n(r, "VarInvGfx"), Throwable: types.n(r, "Throwable") == 1,
			Quiver: types.s(r, "Quiver") != "", AutoStack: types.n(r, "AutoStack") == 1,
			StaffMods: types.s(r, "StaffMods"), CostFormula: types.n(r, "CostFormula"),
		}

		if c, ok := heroIndex[strings.ToLower(types.s(r, "Class"))]; ok {
			ty.Class = c
		}

		it.Types[code] = ty
	}

	for _, ty := range it.Types {
		seen := map[string]bool{}

		var walk func(c string)

		walk = func(c string) {
			if c == "" || seen[c] {
				return
			}

			seen[c] = true
			ty.Ancestors = append(ty.Ancestors, c)

			if p := it.Types[c]; p != nil {
				walk(p.Equiv1)
				walk(p.Equiv2)
			}
		}

		walk(ty.Code)
	}

	for kind, file := range []string{"weapons", "armor", "misc"} {
		tab := readTab(t, src, file+".txt")

		for _, r := range tab.rows {
			code := tab.s(r, "code")
			if code == "" {
				continue
			}

			b := &BaseItem{
				Class: len(it.Items), Kind: BaseKind(kind), Code: code,
				NormCode: tab.s(r, "normcode"), UberCode: tab.s(r, "ubercode"), UltraCode: tab.s(r, "ultracode"),
				Type: tab.s(r, "type"), Type2: tab.s(r, "type2"), Version: tab.n(r, "version"),
				Level: tab.n(r, "level"), LevelReq: tab.n(r, "levelreq"), Rarity: tab.n(r, "rarity"),
				Spawnable: tab.n(r, "spawnable") == 1, Quest: tab.n(r, "quest"),
				QuestDiffCheck: tab.n(r, "questdiffcheck"), Unique: tab.n(r, "unique") == 1,
				MagicLevel: tab.n(r, "magic lvl"), AutoPrefix: tab.n(r, "auto prefix"),
				MinAC: tab.n(r, "minac"), MaxAC: tab.n(r, "maxac"), Block: tab.n(r, "block"),
				Absorbs: tab.n(r, "absorbs"), Speed: tab.n(r, "speed"), Durability: tab.n(r, "durability"),
				NoDurability: tab.n(r, "nodurability") == 1,
				MinDam:       tab.n(r, "mindam"), MaxDam: tab.n(r, "maxdam"),
				TwoHandMinDam: tab.n(r, "2handmindam"), TwoHandMaxDam: tab.n(r, "2handmaxdam"),
				MinMisDam: tab.n(r, "minmisdam"), MaxMisDam: tab.n(r, "maxmisdam"),
				StrBonus: tab.n(r, "StrBonus"), DexBonus: tab.n(r, "DexBonus"),
				ReqStr: tab.n(r, "reqstr"), ReqDex: tab.n(r, "reqdex"),
				Stackable: tab.n(r, "stackable") == 1, MinStack: tab.n(r, "minstack"),
				MaxStack: tab.n(r, "maxstack"), SpawnStack: tab.n(r, "spawnstack"),
				HasInv: tab.n(r, "hasinv") == 1, GemSockets: tab.n(r, "gemsockets"),
				Throwable: tab.n(r, "throwable") == 1, Useable: tab.n(r, "useable") == 1,
				TwoHanded: tab.n(r, "2handed") == 1, OneOrTwoHanded: tab.n(r, "1or2handed") == 1,
				Cost: tab.n(r, "cost"), InvWidth: tab.n(r, "invwidth"), InvHeight: tab.n(r, "invheight"),
			}

			it.Items = append(it.Items, b)
			it.ByCode[code] = b
		}
	}

	return it
}

// Loaders of the slice C tables from the extracted game tables (D2_TABLES).
// The game drops the "Expansion" separator rows of its tables, so do these.

// gameTab is a table read like the game does, with the information which
// rows come after the "Expansion" separator.
type gameTab struct {
	*tsv
	exp []bool
	// stateParams: non numeric parameters that are not skills are state names
	// (Sets.txt "fullsetgeneric"), read as 0.
	stateParams bool
}

// cReadTable reads a table without the rows whose first column is
// "Expansion". Tables without a version column take version 100 for the rows
// after the separator.
func cReadTable(t tb, src TableSource, path string) *gameTab {
	t.Helper()

	tab := readTab(t, src, path)
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
func loadPropTablesFrom(t tb, src TableSource) *PropTables {
	t.Helper()

	pt := &PropTables{ByCode: map[string]int{}, SkillParamShift: 6, SkillParamMask: 0x3f}

	isc := readTab(t, src, "ItemStatCost.txt")
	statID := map[string]int{}

	for i, r := range isc.rows {
		id := isc.n(r, "ID")
		if id != i {
			t.Fatalf("ItemStatCost row %d has ID %d", i, id)
		}

		statID[strings.ToLower(isc.s(r, "Stat"))] = id

		pt.ValShift = append(pt.ValShift, isc.n(r, "ValShift"))
	}

	props := cReadTable(t, src, "Properties.txt")

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

	sk := readTab(t, src, "skills.txt")

	for i, r := range sk.rows {
		if sk.n(r, "Id") != i {
			t.Fatalf("skills row %d has Id %d", i, sk.n(r, "Id"))
		}

		pt.Skills = append(pt.Skills, SkillInfo{ReqLevel: sk.n(r, "reqlevel"), MaxLevel: sk.n(r, "maxlvl")})
	}

	return pt
}

// cSkillByName maps skill names to ids (for the param columns).
func cSkillByName(t tb, src TableSource) map[string]int {
	t.Helper()

	sk := readTab(t, src, "skills.txt")
	out := map[string]int{}

	for i, r := range sk.rows {
		out[strings.ToLower(sk.s(r, "skill"))] = i
	}

	return out
}

// cPropInst parses one prop/par/min/max column group of a row.
func cPropInst(t tb, pt *PropTables, skills map[string]int, tab *gameTab, r []string, prop, par, min, max string) PropInst {
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
		if !ok && tab.stateParams {
			break
		}

		if !ok {
			t.Fatalf("%s: unknown skill %q", code, p)
		}

		pi.Param = id
	}

	return pi
}

// loadUniqueTables reads UniqueItems.txt, SetItems.txt and Sets.txt.
func loadUniqueTablesFrom(t tb, src TableSource, pt *PropTables) *UniqueTables {
	t.Helper()

	skills := cSkillByName(t, src)
	ut := &UniqueTables{}

	u := cReadTable(t, src, "UniqueItems.txt")

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

	sets := cReadTable(t, src, "Sets.txt")
	sets.stateParams = true
	setRow := map[string]int{}

	for i, r := range sets.rows {
		def := SetDef{Name: sets.s(r, "index"), Version: sets.n(r, "version")}

		for k := range def.Partial {
			n := fmt.Sprintf("%d%c", k/2+2, 'a'+k%2)
			def.Partial[k] = cPropInst(t, pt, skills, sets, r, "pcode"+n, "pparam"+n, "pmin"+n, "pmax"+n)
		}

		for k := range def.Full {
			n := strconv.Itoa(k + 1)
			def.Full[k] = cPropInst(t, pt, skills, sets, r, "fcode"+n, "fparam"+n, "fmin"+n, "fmax"+n)
		}

		ut.Sets = append(ut.Sets, def)
		setRow[sets.s(r, "index")] = i
	}

	s := cReadTable(t, src, "SetItems.txt")

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
func loadTestAffixesFrom(t tb, src TableSource, pt *PropTables) [][3]PropInst {
	t.Helper()

	skills := cSkillByName(t, src)

	var out [][3]PropInst

	for _, name := range []string{"MagicSuffix.txt", "MagicPrefix.txt", "AutoMagic.txt"} {
		tab := cReadTable(t, src, name)

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
func loadTestQualityFrom(t tb, src TableSource, pt *PropTables) [][2]PropInst {
	t.Helper()

	skills := cSkillByName(t, src)
	tab := cReadTable(t, src, "QualityItems.txt")

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

// loadQualityTables reads QualityItems, LowQualityItems, ItemRatio and
// Skills (the four tables slice A needs) from D2_TABLES. Books.txt is not
// extracted: its two rows are the ones the game holds in memory (verified
// with the emulator): town portal first, then identify.
func loadQualityTablesFrom(t tb, src TableSource) *QualityTables {
	t.Helper()

	q := &QualityTables{
		Books:     []BookRow{{"tsc", "tbk"}, {"isc", "ibk"}, {}}, // the game's table has a third, empty row
		SkillType: map[int]string{},
	}

	qi := readTab(t, src, "QualityItems.txt")
	for _, r := range qi.rows {
		if qi.s(r, "nummods") == "" {
			continue
		}

		b := func(c string) bool { return qi.n(r, c) == 1 }

		q.Superior = append(q.Superior, SuperiorRow{
			Armor: b("armor"), Weapon: b("weapon"), Shield: b("shield"), Scepter: b("scepter"),
			Wand: b("wand"), Staff: b("staff"), Bow: b("bow"), Boots: b("boots"), Gloves: b("gloves"),
			Belt: b("belt"),
		})
	}

	lq := readTab(t, src, "LowQualityItems.txt")
	for _, r := range lq.rows {
		if lq.s(r, "Name") != "" {
			q.Low++
		}
	}

	ir := readTab(t, src, "ItemRatio.txt")
	for _, r := range ir.rows {
		dr := func(n string) DropRatio {
			return DropRatio{ir.n(r, n), ir.n(r, n+"Divisor"), ir.n(r, n+"Min")}
		}

		q.Ratios = append(q.Ratios, RatioRow{
			Version: ir.n(r, "Version"), Uber: ir.n(r, "Uber") == 1, ClassSpecific: ir.n(r, "Class Specific") == 1,
			Ratio: Ratio{
				Unique: dr("Unique"), Rare: dr("Rare"), Set: dr("Set"), Magic: dr("Magic"),
				HiQuality: DropRatio{ir.n(r, "HiQuality"), ir.n(r, "HiQualityDivisor"), 0},
				Normal:    DropRatio{ir.n(r, "Normal"), ir.n(r, "NormalDivisor"), 0},
			},
		})
	}

	sk := readTab(t, src, "skills.txt")
	for _, r := range sk.rows {
		if sk.s(r, "Id") == "" {
			continue
		}

		id := sk.n(r, "Id")

		if c := heroClass(sk.s(r, "charclass")); c >= 0 {
			q.ClassSkills[c] = append(q.ClassSkills[c], id)
		}

		if it := sk.s(r, "itypea1"); it != "" {
			q.SkillType[id] = it
		}
	}

	return q
}

// gen_create_b.py of the oracle).

func typeCols(tab *tsv, row []string, prefix string, n int) []string {
	out := make([]string, 0, n)

	for i := 1; i <= n; i++ {
		code := tab.s(row, prefix+string(rune('0'+i)))
		if len(code) > 4 { // the game keeps type codes in 4 bytes: "staff" is "staf"
			code = code[:4]
		}

		out = append(out, code)
	}

	return out
}

// loadAffixTables builds the combined table: MagicSuffix, MagicPrefix,
// AutoMagic rows (the "Expansion" separator row is dropped, blank rows stay)
// and the rare names, RareSuffix first.
func loadAffixTablesFrom(t tb, src TableSource) *AffixTables {
	t.Helper()

	// Properties whose first stat is the number of sockets.
	sockets := map[string]bool{}
	props := readTab(t, src, "Properties.txt")

	for _, r := range props.rows {
		if props.s(r, "stat1") == "item_numsockets" {
			sockets[props.s(r, "code")] = true
		}
	}

	at := &AffixTables{}

	load := func(file string) int {
		tab := readTab(t, src, file)
		n := 0

		for _, r := range tab.rows {
			if tab.s(r, "name") == "Expansion" {
				continue
			}

			// The class restriction (the byte at +0x66 of the game's record) is
			// the classspecific column; the class column is a different
			// field (it is not used by the pickers).
			cls := -1
			if c, ok := heroIndex[strings.ToLower(tab.s(r, "classspecific"))]; ok {
				cls = c
			}

			at.Rows = append(at.Rows, AffixRow{
				Name: tab.s(r, "name"), Version: tab.n(r, "version"), Spawnable: tab.n(r, "spawnable") != 0,
				Rare: tab.n(r, "rare") != 0, Level: tab.n(r, "level"), MaxLevel: tab.n(r, "maxlevel"),
				Frequency: tab.n(r, "frequency"), Group: tab.n(r, "group"), Class: cls,
				IType: typeCols(tab, r, "itype", 7), EType: typeCols(tab, r, "etype", 5),
				Mod1Sockets: sockets[tab.s(r, "mod1code")],
			})
			n++
		}

		return n
	}

	at.NSuffix = load("MagicSuffix.txt")
	at.NPrefix = load("MagicPrefix.txt")
	load("AutoMagic.txt")

	for i, file := range []string{"RareSuffix.txt", "RarePrefix.txt"} {
		tab := readTab(t, src, file)

		for _, r := range tab.rows {
			at.RareNames = append(at.RareNames, RareName{
				Name: tab.s(r, "name"), Version: tab.n(r, "version"),
				IType: typeCols(tab, r, "itype", 7), EType: typeCols(tab, r, "etype", 4),
			})
		}

		if i == 0 {
			at.NRareSuffixName = len(at.RareNames)
		}
	}

	return at
}
