package d2drop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// Real-data loader for the affix tables (the game loads them from the first of
// patch_d2, d2exp, d2data that has them) and the comparison with the golden
// recorded from the real Game.exe (testdata/create_b.json, written by
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
func loadAffixTables(t *testing.T) *AffixTables {
	t.Helper()

	// Properties whose first stat is the number of sockets.
	sockets := map[string]bool{}
	props := readTSV(t, tablePath(t, "Properties.txt"))

	for _, r := range props.rows {
		if props.s(r, "stat1") == "item_numsockets" {
			sockets[props.s(r, "code")] = true
		}
	}

	at := &AffixTables{}

	load := func(file string) int {
		tab := readTSV(t, tablePath(t, file))
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
		tab := readTSV(t, tablePath(t, file))

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

func TestLoadAffixTables(t *testing.T) {
	at := loadAffixTables(t)

	// Sizes of the game's combined table (read from the emulated game).
	if at.NSuffix != 747 || at.NPrefix != 669 || len(at.Rows) != 1452 {
		t.Errorf("affix table: %d suffixes, %d prefixes, %d rows", at.NSuffix, at.NPrefix, len(at.Rows))
	}

	if len(at.RareNames) != 201 || at.NRareSuffixName != 155 {
		t.Errorf("rare names: %d, %d suffix names", len(at.RareNames), at.NRareSuffixName)
	}

	// Rows read from the game: id 1 is "of Health", id 1417 "Fletcher's".
	if at.row(1).Name != "of Health" || at.row(1417).Name != "Fletcher's" || at.row(1452).Name != "of Anthrax" {
		t.Errorf("rows: %q %q %q", at.row(1).Name, at.row(1417).Name, at.row(1452).Name)
	}
}

type cbSnap struct {
	Q   int       `json:"q"`
	Fl  uint32    `json:"fl"`
	Pre [3]int    `json:"pre"`
	Suf [3]int    `json:"suf"`
	Au  int       `json:"au"`
	Rn  [2]int    `json:"rn"`
	I   [2]uint32 `json:"i"`
	Il  int       `json:"il"`
	Vw  int       `json:"vw"`
}

type cbEvent struct {
	F  string `json:"f"`
	RV uint32 `json:"rv"`
	B  cbSnap `json:"b"`
	A  cbSnap `json:"a"`
}

type cbCase struct {
	C   string    `json:"c"`
	Il  int       `json:"il"`
	Q   int       `json:"q"`
	Df  int       `json:"df"`
	Gs  uint32    `json:"gs"`
	V   int       `json:"v"`
	X   int       `json:"x"`
	Fq  int       `json:"fq"`
	Fil int       `json:"fil"`
	Ev  []cbEvent `json:"ev"`
}

type cbGolden struct {
	V     int      `json:"v"`
	Cases []cbCase `json:"cases"`
}

func TestOracleAffixes(t *testing.T) {
	items := loadItemTables(t)
	affixes := loadAffixTables(t)

	raw, err := os.ReadFile(filepath.Join("testdata", "create_b.json"))
	if err != nil {
		t.Fatal(err)
	}

	var g cbGolden
	if err = json.Unmarshal(raw, &g); err != nil {
		t.Fatal(err)
	}

	cr := &Creator{Items: items, Affixes: affixes}
	stats := map[string][2]int{} // events compared, whose item generator was compared too
	bad := 0

	for ci := range g.Cases {
		cs := &g.Cases[ci]
		base := items.ByCode[cs.C]

		if base == nil {
			t.Fatalf("unknown item %q", cs.C)
		}

		tempered := 0

		for ei := range cs.Ev {
			ev := &cs.Ev[ei]
			st := &itemState{
				c: cr, base: base, ilvl: cs.Fil,
				req: Request{Code: cs.C, ILvl: cs.Il, Version: cs.V, Expansion: cs.X == 1, Difficulty: cs.Df},
			}
			st.quality = Quality(ev.B.Q)
			st.flags = ev.B.Fl
			st.prefix, st.suffix, st.auto, st.rareNames = ev.B.Pre, ev.B.Suf, ev.B.Au, ev.B.Rn
			st.item = d2rand.Seed{Lo: ev.B.I[0], Hi: ev.B.I[1]}

			var ok bool

			full := !cr.HasStaffMods(st) // the staffmods rolls are not ported
			names := false
			rv := ev.RV != 0

			switch ev.F {
			case "m":
				ok = cr.rollMagic(st)
			case "r":
				ok, names = cr.rollRare(st), true
			case "c":
				ok, names = cr.rollCrafted(st), true
			case "t":
				id := cr.pickRareName(st, tempered == 0)
				tempered++
				ok, rv = id == int(ev.RV), true

				if id != int(ev.RV) {
					t.Errorf("case %d %s il %d v %d tempered name %d: got %d want %d", ci, cs.C, cs.Il, cs.V, tempered, id, ev.RV)
					bad++
				}
			case "a":
				id := cr.pickAutomagic(st)
				ok, rv = id == int(ev.RV), true

				if !ok {
					t.Errorf("case %d %s il %d v %d auto: got %d want %d", ci, cs.C, cs.Il, cs.V, id, ev.RV)
					bad++
				}
			case "k": // ITEMGEN_RollCharmAffixes, called directly by the generator
				ok = cr.rollCharm(st)
				full = true
			default:
				t.Fatalf("event %q", ev.F)
			}

			if ev.F == "t" || ev.F == "a" {
				full = true // no staffmods roll after these
			}

			if ev.F != "t" && ev.F != "a" && ok != rv {
				t.Errorf("case %d %s il %d q %d v %d: %s returned %v, game %d", ci, cs.C, cs.Il, cs.Q, cs.V, ev.F, ok, ev.RV)
				bad++
			}

			s := stats[ev.F]
			s[0]++
			stats[ev.F] = s

			if st.prefix != ev.A.Pre || st.suffix != ev.A.Suf || (names && st.rareNames != ev.A.Rn) {
				t.Errorf("case %d %s il %d q %d v %d x %d: %s affixes %v %v %v %v, game %v %v %v %v", ci, cs.C, cs.Il, cs.Q, cs.V, cs.X,
					ev.F, st.prefix, st.suffix, st.auto, st.rareNames, ev.A.Pre, ev.A.Suf, ev.A.Au, ev.A.Rn)
				bad++
			}

			if full {
				s[1]++
				stats[ev.F] = s

				if got := [2]uint32{st.item.Lo, st.item.Hi}; got != ev.A.I {
					t.Errorf("case %d %s il %d q %d v %d x %d: %s generator %v, game %v", ci, cs.C, cs.Il, cs.Q, cs.V, cs.X, ev.F, got, ev.A.I)
					bad++
				}

				if ev.F != "t" && ev.F != "a" && st.flags != ev.A.Fl {
					t.Errorf("case %d %s: %s flags %#x, game %#x", ci, cs.C, ev.F, st.flags, ev.A.Fl)
					bad++
				}
			}

			if bad > 25 {
				t.Fatal("too many mismatches")
			}
		}
	}

	t.Logf("events compared (and with generator state): %v", stats)
}

func TestAffixLevelLoD(t *testing.T) {
	cases := []struct{ ilvl, qlvl, ml, want int }{
		{50, 20, 0, 40},
		{90, 40, 0, 81}, // 90 >= 99-20: 2*90-99
		{1, 60, 0, 30},  // the item level is raised to the base level first
		{99, 0, 0, 99},
		{50, 20, 3, 53}, // magic level is added
		{99, 0, 10, 99}, // clamped
		{1, 0, 0, 1},
	}

	for _, c := range cases {
		if got := affixLevel(c.ilvl, c.qlvl, c.ml); got != c.want {
			t.Errorf("affixLevel(%d,%d,%d) = %d, want %d", c.ilvl, c.qlvl, c.ml, got, c.want)
		}
	}
}
