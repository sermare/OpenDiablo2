package d2itemdesc

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Source returns the bytes of a game table by file name, for example
// "armor.txt" or "uniqueitems.txt". The caller decides where the files
// come from (the MPQ layers of the game, or a folder of extracted tables
// in the tests). ok is false when the file does not exist.
type Source func(name string) (data []byte, ok bool)

// Translator maps a string table key ("ModStr1a") to its text. It returns
// "" when the key is unknown.
type Translator func(key string) string

// StatDef is one row of ItemStatCost.txt, the columns that drive how a
// stat is described.
type StatDef struct {
	ID       int
	Name     string
	Priority int // descpriority: higher is listed first
	Func     int // descfunc
	Val      int // descval: 0 none, 1 value before the text, 2 value after
	StrPos   string
	StrNeg   string
	Str2     string

	Grp                          int
	GrpFunc, GrpVal              int
	GrpStrPos, GrpStrNeg, GrpStr string

	Op, OpParam int
}

// PropSlot is one stat slot of a properties.txt row.
type PropSlot struct {
	Func int
	Stat string
}

// PropSpec is a property as the item tables write it (a property code with
// a parameter and a value range), for example the bonuses of a set.
type PropSpec struct {
	Code     string
	Param    string
	Min, Max int
}

// BaseItem is the part of armor.txt, weapons.txt and misc.txt that the
// description needs.
type BaseItem struct {
	Code, Name, NameStr string
	Type, Type2         string
	Weapon, Armor       bool

	MinAC, MaxAC      int
	Block             int
	MinDam, MaxDam    int // one hand; for armor.txt the smite damage of paladin shields
	TwoMin, TwoMax    int
	MissMin, MissMax  int
	OneOrTwo, TwoOnly bool
	ReqStr, ReqDex    int
	ReqLevel          int
	Durability        int
	NoDurability      bool
	Stackable         bool
	Cost              int
	MaxSockets        int
	Speed             int
	Throwable         bool
	Width, Height     int
}

// AffixDef is a row of magicprefix.txt / magicsuffix.txt.
type AffixDef struct {
	Name     string
	LevelReq int
}

// UniqueDef is a row of uniqueitems.txt.
type UniqueDef struct {
	Index    string // name key (also the English name)
	Code     string
	LevelReq int
}

// SetItemDef is a row of setitems.txt.
type SetItemDef struct {
	Index    string
	Set      string
	Code     string
	LevelReq int
	// Partial are the bonus lists of the item (aprop1a/1b .. aprop5a/5b):
	// list i is active when i+2 pieces of the set are worn.
	Partial [5][]PropSpec
}

// SetDef is a row of sets.txt.
type SetDef struct {
	Index string
	Name  string
	// Partial are the set wide bonuses (PCode2a.. PCode5b): list i is
	// active when i+2 pieces are worn.
	Partial [4][]PropSpec
	Full    []PropSpec // active when the whole set is worn
	Items   []string   // setitems.txt index names, file order
}

// RunewordDef is a row of runes.txt.
type RunewordDef struct {
	Key     string // "Runeword130"
	Display string
	Runes   []string // rune item codes
}

// GemDef is a row of gems.txt: the effect of a gem or rune per item class.
type GemDef struct {
	Name, Code string
	Weapon     []PropSpec
	Helm       []PropSpec
	Shield     []PropSpec
}

// ClassDef is a row of charstats.txt.
type ClassDef struct {
	Name      string
	AllSkills string // string key of "to Amazon Skill Levels"
	SkillTabs [3]string
	ClassOnly string
}

// SkillDef is a skill of skills.txt.
type SkillDef struct {
	Code  string // internal skill name of skills.txt
	Name  string
	Class int // charstats row, -1 for none
}

// Tables carries every game table the description needs. Build it once
// with Load and reuse it.
type Tables struct {
	Tr Translator

	Stats      map[int]*StatDef
	StatByName map[string]*StatDef
	Props      map[string][]PropSlot
	Bases      map[string]*BaseItem
	Gems       map[string]*GemDef
	Skills     map[int]*SkillDef
	Classes    []ClassDef

	// types maps an item type code to its parent type codes (ItemTypes.txt)
	types map[string][]string
	// typeClass maps an item type code to its class restriction code
	typeClass map[string]string

	Prefixes, Suffixes       []AffixDef
	RarePrefixes, RareSuffix []string
	Uniques                  []UniqueDef
	SetItems                 []SetItemDef
	Sets                     map[string]*SetDef
	Runewords                []RunewordDef // ordered by their save id minus runewordIDBase

	// EtherealText is the line printed for ethereal items; the original
	// string is not in the string tables (unverified), so it can be replaced.
	EtherealText string
}

// ErrMissingTable is returned by Load when a required table is absent.
var ErrMissingTable = errors.New("d2itemdesc: missing table")

// the save id of the first runeword (Runeword1 is not id 0).
const runewordIDBase = 27

// Load reads every table through src. Tables a game version does not have
// (for example the Lord of Destruction ones in a classic install) leave the
// matching lookups empty; ItemStatCost, armor, weapons and misc are required.
func Load(src Source, tr Translator) (*Tables, error) {
	if tr == nil {
		tr = func(string) string { return "" }
	}

	t := &Tables{
		Tr:           tr,
		Stats:        make(map[int]*StatDef),
		StatByName:   make(map[string]*StatDef),
		Props:        make(map[string][]PropSlot),
		Bases:        make(map[string]*BaseItem),
		Gems:         make(map[string]*GemDef),
		Skills:       make(map[int]*SkillDef),
		Sets:         make(map[string]*SetDef),
		types:        make(map[string][]string),
		typeClass:    make(map[string]string),
		EtherealText: "Ethereal (Cannot be Repaired)",
	}

	need := func(name string) (*tsv, error) {
		d, ok := src(name)
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrMissingTable, name)
		}

		return parseTSV(d), nil
	}

	opt := func(name string) *tsv {
		if d, ok := src(name); ok {
			return parseTSV(d)
		}

		return &tsv{cols: map[string]int{}}
	}

	isc, err := need("ItemStatCost.txt")
	if err != nil {
		return nil, err
	}

	t.loadStats(isc)

	for _, f := range []struct {
		name string
		kind int
	}{{"armor.txt", 0}, {"weapons.txt", 1}, {"misc.txt", 2}} {
		tb, err := need(f.name)
		if err != nil {
			return nil, err
		}

		t.loadBases(tb, f.kind)
	}

	t.loadTypes(opt("ItemTypes.txt"))
	t.loadProps(opt("properties.txt"))
	t.loadGems(opt("gems.txt"))
	t.loadAffixes(opt("magicprefix.txt"), opt("magicsuffix.txt"), opt("rareprefix.txt"), opt("raresuffix.txt"))
	t.loadUniques(opt("uniqueitems.txt"))
	t.loadSets(opt("sets.txt"), opt("setitems.txt"))
	t.loadRunewords(opt("runes.txt"))
	t.loadClasses(opt("charstats.txt"))
	t.loadSkills(opt("skills.txt"), opt("skilldesc.txt"))

	return t, nil
}

func (t *Tables) loadStats(tb *tsv) {
	for _, r := range tb.rows {
		name := tb.str(r, "Stat")
		if name == "" {
			continue
		}

		id, err := strconv.Atoi(tb.str(r, "ID"))
		if err != nil {
			continue
		}

		s := &StatDef{
			ID: id, Name: name,
			Priority: tb.num(r, "descpriority"), Func: tb.num(r, "descfunc"), Val: tb.num(r, "descval"),
			StrPos: tb.str(r, "descstrpos"), StrNeg: tb.str(r, "descstrneg"), Str2: tb.str(r, "descstr2"),
			Grp: tb.num(r, "dgrp"), GrpFunc: tb.num(r, "dgrpfunc"), GrpVal: tb.num(r, "dgrpval"),
			GrpStrPos: tb.str(r, "dgrpstrpos"), GrpStrNeg: tb.str(r, "dgrpstrneg"), GrpStr: tb.str(r, "dgrpstr2"),
			Op: tb.num(r, "op"), OpParam: tb.num(r, "op param"),
		}
		t.Stats[id] = s
		t.StatByName[name] = s
	}
}

func (t *Tables) loadBases(tb *tsv, kind int) {
	for _, r := range tb.rows {
		code := tb.str(r, "code")
		if code == "" {
			continue
		}

		b := &BaseItem{
			Code: code, Name: tb.str(r, "name"), NameStr: tb.str(r, "namestr"),
			Type: tb.str(r, "type"), Type2: tb.str(r, "type2"),
			Weapon: kind == 1, Armor: kind == 0,
			MinAC: tb.num(r, "minac"), MaxAC: tb.num(r, "maxac"), Block: tb.num(r, "block"),
			MinDam: tb.num(r, "mindam"), MaxDam: tb.num(r, "maxdam"),
			TwoMin: tb.num(r, "2handmindam"), TwoMax: tb.num(r, "2handmaxdam"),
			MissMin: tb.num(r, "minmisdam"), MissMax: tb.num(r, "maxmisdam"),
			OneOrTwo: tb.num(r, "1or2handed") == 1, TwoOnly: tb.num(r, "2handed") == 1,
			ReqStr: tb.num(r, "reqstr"), ReqDex: tb.num(r, "reqdex"), ReqLevel: tb.num(r, "levelreq"),
			Durability: tb.num(r, "durability"), NoDurability: tb.num(r, "nodurability") == 1,
			Stackable: tb.num(r, "stackable") == 1, Cost: tb.num(r, "cost"),
			MaxSockets: tb.num(r, "gemsockets"), Speed: tb.num(r, "speed"),
			Throwable: tb.num(r, "throwable") == 1,
			Width:     tb.num(r, "invwidth"), Height: tb.num(r, "invheight"),
		}
		t.Bases[code] = b
	}
}

func (t *Tables) loadTypes(tb *tsv) {
	for _, r := range tb.rows {
		code := tb.str(r, "Code")
		if code == "" {
			continue
		}

		for _, c := range []string{"Equiv1", "Equiv2"} {
			if p := tb.str(r, c); p != "" {
				t.types[code] = append(t.types[code], p)
			}
		}

		if c := tb.str(r, "Class"); c != "" {
			t.typeClass[code] = c
		}
	}
}

// IsA reports whether item type code is, or derives from, target.
func (t *Tables) IsA(code, target string) bool {
	return t.isA(code, target, 0)
}

func (t *Tables) isA(code, target string, depth int) bool {
	if code == target {
		return true
	}

	if depth > 16 {
		return false
	}

	for _, p := range t.types[code] {
		if t.isA(p, target, depth+1) {
			return true
		}
	}

	return false
}

func (t *Tables) loadProps(tb *tsv) {
	for _, r := range tb.rows {
		code := tb.str(r, "code")
		if code == "" {
			continue
		}

		var slots []PropSlot

		for i := 1; i <= 7; i++ {
			f, st := tb.num(r, "func"+strconv.Itoa(i)), tb.str(r, "stat"+strconv.Itoa(i))
			if f == 0 && st == "" {
				continue
			}

			slots = append(slots, PropSlot{Func: f, Stat: st})
		}

		t.Props[code] = slots
	}
}

func (t *Tables) loadGems(tb *tsv) {
	for _, r := range tb.rows {
		code := tb.str(r, "code")
		if code == "" {
			continue
		}

		g := &GemDef{Name: tb.str(r, "name"), Code: code}
		for _, m := range []struct {
			dst    *[]PropSpec
			prefix string
		}{{&g.Weapon, "weaponMod"}, {&g.Helm, "helmMod"}, {&g.Shield, "shieldMod"}} {
			for i := 1; i <= 3; i++ {
				n := strconv.Itoa(i)

				c := tb.str(r, m.prefix+n+"Code")
				if c == "" {
					continue
				}

				*m.dst = append(*m.dst, PropSpec{Code: c, Param: tb.str(r, m.prefix+n+"Param"),
					Min: tb.num(r, m.prefix+n+"Min"), Max: tb.num(r, m.prefix+n+"Max")})
			}
		}

		t.Gems[code] = g
	}
}

func (t *Tables) loadAffixes(pre, suf, rpre, rsuf *tsv) {
	load := func(tb *tsv) []AffixDef {
		var out []AffixDef

		// blank rows count: the save stores the row number (verified
		// against the sample save by the d2hero importer)
		for _, r := range tb.rows {
			out = append(out, AffixDef{Name: tb.str(r, "Name"), LevelReq: tb.num(r, "levelreq")})
		}

		return out
	}

	t.Prefixes, t.Suffixes = load(pre), load(suf)

	for _, r := range rpre.rows {
		t.RarePrefixes = append(t.RarePrefixes, rpre.str(r, "name"))
	}

	for _, r := range rsuf.rows {
		t.RareSuffix = append(t.RareSuffix, rsuf.str(r, "name"))
	}
}

func (t *Tables) loadUniques(tb *tsv) {
	for _, r := range tb.rows {
		idx := tb.str(r, "index")
		if idx == "" || strings.EqualFold(idx, "Expansion") {
			continue // the separator rows are not counted by the save ids
		}

		t.Uniques = append(t.Uniques, UniqueDef{Index: idx, Code: tb.str(r, "code"), LevelReq: tb.num(r, "lvl req")})
	}
}

func (t *Tables) loadSets(sets, items *tsv) {
	for _, r := range sets.rows {
		idx := sets.str(r, "index")
		if idx == "" || strings.EqualFold(idx, "Expansion") {
			continue
		}

		s := &SetDef{Index: idx, Name: sets.str(r, "name")}

		for i := 0; i < 4; i++ {
			n := strconv.Itoa(i + 2)
			for _, l := range []string{"a", "b"} {
				if c := sets.str(r, "PCode"+n+l); c != "" {
					s.Partial[i] = append(s.Partial[i], PropSpec{Code: c, Param: sets.str(r, "PParam"+n+l),
						Min: sets.num(r, "PMin"+n+l), Max: sets.num(r, "PMax"+n+l)})
				}
			}
		}

		for i := 1; i <= 8; i++ {
			n := strconv.Itoa(i)
			if c := sets.str(r, "FCode"+n); c != "" {
				s.Full = append(s.Full, PropSpec{Code: c, Param: sets.str(r, "FParam"+n),
					Min: sets.num(r, "FMin"+n), Max: sets.num(r, "FMax"+n)})
			}
		}

		t.Sets[idx] = s
	}

	for _, r := range items.rows {
		idx := items.str(r, "index")
		if idx == "" || strings.EqualFold(idx, "Expansion") {
			continue
		}

		si := SetItemDef{Index: idx, Set: items.str(r, "set"), Code: items.str(r, "item"), LevelReq: items.num(r, "lvl req")}

		for i := 0; i < 5; i++ {
			n := strconv.Itoa(i + 1)
			for _, l := range []string{"a", "b"} {
				if c := items.str(r, "aprop"+n+l); c != "" {
					si.Partial[i] = append(si.Partial[i], PropSpec{Code: c, Param: items.str(r, "apar"+n+l),
						Min: items.num(r, "amin"+n+l), Max: items.num(r, "amax"+n+l)})
				}
			}
		}

		t.SetItems = append(t.SetItems, si)

		if s := t.Sets[si.Set]; s != nil {
			s.Items = append(s.Items, idx)
		}
	}
}

func (t *Tables) loadRunewords(tb *tsv) {
	type rw struct {
		n   int
		def RunewordDef
	}

	var all []rw

	for _, r := range tb.rows {
		key := tb.str(r, "Name")
		if !strings.HasPrefix(key, "Runeword") {
			continue
		}

		n, err := strconv.Atoi(strings.TrimPrefix(key, "Runeword"))
		if err != nil {
			continue
		}

		d := RunewordDef{Key: key, Display: tb.str(r, "Rune Name")}

		for i := 1; i <= 6; i++ {
			if c := tb.str(r, "Rune"+strconv.Itoa(i)); c != "" {
				d.Runes = append(d.Runes, c)
			}
		}

		all = append(all, rw{n, d})
	}

	// the id a save stores is the rank of the Runeword<N> number plus 27
	// (verified: Chains of Honor 40, Heart of the Oak 77, Spirit 155 of the
	// sample save; Runeword80 and Runeword96 do not exist)
	sort.Slice(all, func(i, j int) bool { return all[i].n < all[j].n })

	for _, a := range all {
		t.Runewords = append(t.Runewords, a.def)
	}
}

func (t *Tables) loadClasses(tb *tsv) {
	for _, r := range tb.rows {
		name := tb.str(r, "class")
		if name == "" {
			continue
		}

		t.Classes = append(t.Classes, ClassDef{
			Name: name, AllSkills: tb.str(r, "StrAllSkills"),
			SkillTabs: [3]string{tb.str(r, "StrSkillTab1"), tb.str(r, "StrSkillTab2"), tb.str(r, "StrSkillTab3")},
			ClassOnly: tb.str(r, "StrClassOnly"),
		})
	}
}

func (t *Tables) loadSkills(sk, sd *tsv) {
	names := make(map[string]string)

	for _, r := range sd.rows {
		names[sd.str(r, "skilldesc")] = sd.str(r, "str name")
	}

	classIdx := map[string]int{"ama": 0, "sor": 1, "nec": 2, "pal": 3, "bar": 4, "dru": 5, "ass": 6}

	for _, r := range sk.rows {
		id, err := strconv.Atoi(sk.str(r, "Id"))
		if err != nil {
			continue
		}

		cls, ok := classIdx[sk.str(r, "charclass")]
		if !ok {
			cls = -1
		}

		key := names[sk.str(r, "skilldesc")]
		if key == "" {
			key = sk.str(r, "skill")
		}

		t.Skills[id] = &SkillDef{Code: sk.str(r, "skill"), Name: key, Class: cls}
	}
}

// tr translates a key, falling back to the key itself.
func (t *Tables) tr(key string) string {
	if key == "" {
		return ""
	}

	if s := t.Tr(key); s != "" {
		return s
	}

	return key
}
