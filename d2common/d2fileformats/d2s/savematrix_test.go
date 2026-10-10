package d2s

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Save matrix: synthetic saves generated in memory (nothing is committed) for every class, difficulty,
// act, flag combination, file version and a range of item layouts. Each is written, parsed and written
// again; the two files must be byte identical, and the parsed fields must match what was put in.

type matrixTables struct {
	name string
	tb   *ItemTables
}

// matrixTableSets always has the synthetic tables; the real ones are added when $D2_TABLES is set.
func matrixTableSets(t *testing.T) []matrixTables {
	sets := []matrixTables{{"synth", synthTables(t)}}
	if tb := robustTables(); tb != nil {
		sets = append(sets, matrixTables{"real", tb})
	}

	return sets
}

// matrixBase returns a parsed minimal expansion save to edit.
func matrixBase(t testing.TB, tb *ItemTables) *Character {
	t.Helper()

	var skills [numSkills]byte

	skills[0] = 1
	data := buildBody(t, map[int]uint64{StatStrength: 30, StatLevel: 5, StatGold: 100}, skills)
	data = append(data, 'J', 'M', 0, 0)
	data = append(data, mercTag...)
	data = fixup(data)

	c, err := Parse(data, tb)
	if err != nil {
		t.Fatalf("base: %v", err)
	}

	return c
}

func roundTripExact(t *testing.T, label string, c *Character, tb *ItemTables) []byte {
	t.Helper()

	out, err := Write(c, tb)
	if err != nil {
		t.Fatalf("%s: write: %v", label, err)
	}

	back, err := Parse(out, tb)
	if err != nil {
		t.Fatalf("%s: parse: %v", label, err)
	}

	again, err := Write(back, tb)
	if err != nil {
		t.Fatalf("%s: rewrite: %v", label, err)
	}

	if !bytes.Equal(out, again) {
		t.Fatalf("%s: round trip differs at 0x%X (len %d vs %d)", label, firstDiff(out, again), len(out), len(again))
	}

	if got := binary.LittleEndian.Uint32(out[checksumOffset:]); got != Checksum(out) {
		t.Fatalf("%s: checksum stale", label)
	}

	if int(binary.LittleEndian.Uint32(out[8:])) != len(out) {
		t.Fatalf("%s: size field stale", label)
	}

	if !reflect.DeepEqual(c.Header.Mercenary, back.Header.Mercenary) || c.Header.Name != back.Header.Name ||
		c.Header.Class != back.Header.Class || c.Header.Status != back.Header.Status ||
		c.Header.Version != back.Header.Version || c.Header.Difficulty != back.Header.Difficulty ||
		c.Header.Level != back.Header.Level || c.Header.MapSeed != back.Header.MapSeed {
		t.Fatalf("%s: header changed: %+v -> %+v", label, c.Header, back.Header)
	}

	if len(back.Items) != len(c.Items) || len(back.Corpse) != len(c.Corpse) || len(back.MercItems) != len(c.MercItems) {
		t.Fatalf("%s: item counts %d/%d/%d -> %d/%d/%d", label,
			len(c.Items), len(c.Corpse), len(c.MercItems), len(back.Items), len(back.Corpse), len(back.MercItems))
	}

	return out
}

func TestMatrixHeaderCombinations(t *testing.T) {
	for _, ts := range matrixTableSets(t) {
		tb := ts.tb
		n := 0

		for class := Amazon; class <= Assassin; class++ {
			for _, version := range []uint32{0x5C, 0x5D, 0x5E, 0x5F, 0x60} {
				for flags := uint32(0); flags < 16; flags++ { // expansion, ladder, hardcore, dead
					status := uint32(0)
					if flags&1 != 0 {
						status |= StatusExpansion
					}

					if flags&2 != 0 {
						status |= StatusLadder
					}

					if flags&4 != 0 {
						status |= StatusHardcore
					}

					if flags&8 != 0 {
						status |= StatusDied
					}

					for diff := 0; diff < 3; diff++ {
						act := (int(class) + int(flags) + diff) % 5

						label := fmt.Sprintf("%s/%v/v%X/f%d/d%d", ts.name, class, version, flags, diff)
						c := matrixBase(t, tb)
						c.Header.Class = class
						c.Header.Version = version
						c.Header.Status = status
						c.Header.Level = uint8(1 + n%99)
						c.Header.Difficulty = [3]byte{}
						c.Header.Difficulty[diff] = activeFlag | byte(act)
						c.Header.MapSeed = uint32(n) * 2654435761
						c.Header.Name = "Hero" + strings.Repeat("x", n%12)

						out := roundTripExact(t, label, c, tb)

						h, err := ParseHeader(out)
						if err != nil {
							t.Fatalf("%s: %v", label, err)
						}

						d, a, ok := h.ActiveDifficulty()
						if !ok || d != diff || a != act {
							t.Fatalf("%s: difficulty %d act %d ok=%v", label, d, a, ok)
						}

						if h.IsHardcore() != (flags&4 != 0) || h.IsDead() != (flags&8 != 0) ||
							h.IsLadder() != (flags&2 != 0) || h.IsExpansion() != (flags&1 != 0) {
							t.Fatalf("%s: flags lost", label)
						}

						n++
					}
				}
			}
		}
	}
}

// matrixItems builds a varied item set; the returned lists are the main, corpse and mercenary lists.
func matrixItems(t testing.TB, tb *ItemTables) (main, corpse, merc []Item, golem Item) {
	t.Helper()

	mk := func(it Item) Item {
		t.Helper()

		if it.Quality == 0 {
			it.Quality = QualityNormal
		}

		out, err := NewItem(it, tb)
		if err != nil {
			t.Fatalf("NewItem %+v: %v", it, err)
		}

		return out
	}

	bowCode := "bow" // the real weapons.txt only knows the type "bow"; its short bow is "sbw"
	if tb.ItemKindOf(bowCode) == 0 {
		bowCode = "sbw"
	}

	props := []Property{{ID: 0, Value: 5}, {ID: 97, Param: 300, Value: 3}}

	var id uint32

	next := func() uint32 { id++; return id * 0x01010101 }

	// equipped (slots 1..12) incl. socketed, ethereal, runeword, personalised
	for slot := uint8(1); slot <= 12; slot++ {
		it := Item{Identified: true, Location: LocationEquipped, Equipped: slot, Code: "cap", ID: next(),
			Level: 60, Quality: QualityMagic, MagicPrefix: uint16(slot), MagicSuffix: 100 + uint16(slot),
			Defense: 40 + int(slot), MaxDurability: 30, Durability: 12, Properties: props}

		switch slot % 4 {
		case 0:
			it.Ethereal = true
		case 1:
			it.Personalized, it.PersonalName = true, "Name-"+fmt.Sprint(slot)
		case 2:
			it.Quality, it.RunewordID, it.TotalSockets = QualityNormal, uint16(40+slot), 2
			it.MagicPrefix, it.MagicSuffix = 0, 0
			it.RunewordProperties = []Property{{ID: 0, Value: 9}}
			it.Properties = nil
		}

		if it.TotalSockets == 0 && slot%3 == 0 {
			it.TotalSockets = 3
		}

		for s := uint8(0); s < it.TotalSockets; s++ {
			it.Children = append(it.Children, mk(Item{Code: "key", Location: LocationSocketed, Quality: QualityNormal,
				Identified: true, ID: next(), Quantity: 1}))
		}

		it.SocketCount = uint8(len(it.Children))
		main = append(main, mk(it))
	}

	// every quality, plus a weapon
	for q := QualityLow; q <= QualityCrafted; q++ {
		it := Item{Identified: true, Location: LocationStored, Page: 1, X: q, Y: 0, Code: bowCode, ID: next(),
			Level: 30, Quality: q, Defense: 0, MaxDurability: 250, Durability: 100, Properties: props}

		switch q {
		case QualityLow:
			it.LowQualityID = 3
		case QualityHigh:
			it.HighQuality = 5
		case QualitySet:
			it.SetID, it.SetListMask = 77, 0b00101
			it.SetProperties = [][]Property{{{ID: 0, Value: 1}}, {{ID: 0, Value: 2}}}
		case QualityUnique:
			it.UniqueID = 4000
		case QualityRare, QualityCrafted:
			it.RareName1, it.RareName2, it.RareMask = 12, 200, 0b101011
			it.RareAffixes = [6]uint16{1, 2, 0, 4, 0, 6}
		}

		main = append(main, mk(it))
	}

	// stash, cube, belt: many items (large inventory)
	for i := 0; i < 120; i++ {
		page := []uint8{5, 4, 1}[i%3]
		x := uint8(i % 10)
		y := uint8(i / 10 % 10)

		it := Item{Identified: true, Location: LocationStored, Page: page, X: x, Y: y, Code: "key", ID: next(),
			Quality: QualityNormal, Quantity: uint16(1 + i%255)}
		if i%7 == 0 {
			it = Item{Code: "cap", ID: next(), Level: uint8(i), Quality: QualityNormal, Identified: true,
				Location: LocationStored, Page: page, X: x, Y: y, Defense: i, MaxDurability: 20, Durability: 20}
		}

		main = append(main, mk(it))
	}

	for i := uint8(0); i < 16; i++ {
		main = append(main, mk(Item{Identified: true, Location: LocationBelt, X: i, Code: "key", ID: next(),
			Quality: QualityNormal, Quantity: 5}))
	}

	main = append(main, Item{Identified: true, Ear: true, Code: "ear", EarInfo: &EarInfo{Class: 2, Level: 40, Name: "EarName"},
		Location: LocationStored, Page: 1, X: 8, Y: 3, Version: DefaultItemVersion})

	for i := 0; i < 4; i++ {
		corpse = append(corpse, mk(Item{Identified: true, Location: LocationEquipped, Equipped: uint8(1 + i), Code: "cap",
			ID: next(), Level: 5, Quality: QualityNormal, Defense: 10, MaxDurability: 10, Durability: 10}))
	}

	for i := 0; i < 4; i++ {
		merc = append(merc, mk(Item{Identified: true, Location: LocationEquipped, Equipped: uint8(1 + i), Code: bowCode,
			ID: next(), Level: 5, Quality: QualityRare, RareName1: 1, RareName2: 2, RareMask: 0b11,
			RareAffixes: [6]uint16{3, 4}, MaxDurability: 50, Durability: 49, Properties: props}))
	}

	golem = mk(Item{Identified: true, Code: "cap", ID: next(), Level: 1, Quality: QualityNormal, Defense: 5,
		MaxDurability: 5, Durability: 5})

	return main, corpse, merc, golem
}

func TestMatrixItemsAllClasses(t *testing.T) {
	for _, ts := range matrixTableSets(t) {
		tb := ts.tb

		for class := Amazon; class <= Assassin; class++ {
			for _, expansion := range []bool{false, true} {
				for _, withMerc := range []bool{false, true} {
					label := fmt.Sprintf("%s/%v/exp=%v/merc=%v", ts.name, class, expansion, withMerc)
					c := matrixBase(t, tb)
					c.Header.Class = class
					c.Header.Status = 0

					if expansion {
						c.Header.Status = StatusExpansion
					}

					main, corpse, merc, golem := matrixItems(t, tb)
					c.Items = main
					c.Corpse, c.HasCorpse = corpse, true

					if withMerc {
						c.Header.Mercenary = Mercenary{ID: 0x1234, NameID: 3, Type: 7, Experience: 99999, Dead: class%2 == 0}
						c.MercItems = merc
					}

					if class == Necromancer && expansion {
						g := golem
						c.Golem = &g
					}

					if withMerc && !expansion {
						if _, err := Write(c, tb); !errors.Is(err, ErrWouldDrop) {
							t.Fatalf("%s: mercenary items without expansion: %v", label, err)
						}

						c.MercItems = nil
					}

					out := roundTripExact(t, label, c, tb)

					back, err := Parse(out, tb)
					if err != nil {
						t.Fatal(err)
					}

					for i := range main {
						if !itemsEqual(main[i], back.Items[i]) {
							t.Fatalf("%s: item %d changed:\n%+v\n%+v", label, i, main[i], back.Items[i])
						}
					}

					if class == Necromancer && expansion && (back.Golem == nil || !itemsEqual(golem, *back.Golem)) {
						t.Fatalf("%s: golem lost", label)
					}
				}
			}
		}
	}
}

// itemsEqual compares the fields a caller can set.
func itemsEqual(a, b Item) bool {
	a.Flags, b.Flags = 0, 0
	a.Properties, b.Properties = normProps(a.Properties), normProps(b.Properties)
	a.RunewordProperties, b.RunewordProperties = normProps(a.RunewordProperties), normProps(b.RunewordProperties)

	for _, it := range []*Item{&a, &b} {
		sets := make([][]Property, len(it.SetProperties))
		for i := range sets {
			sets[i] = normProps(it.SetProperties[i])
		}

		it.SetProperties = sets
	}

	if len(a.Children) != len(b.Children) {
		return false
	}

	for i := range a.Children {
		if !itemsEqual(a.Children[i], b.Children[i]) {
			return false
		}
	}

	a.Children, b.Children = nil, nil

	return reflect.DeepEqual(a, b)
}

func normProps(p []Property) []Property {
	if len(p) == 0 {
		return nil
	}

	out := make([]Property, len(p))
	for i, q := range p {
		q.Name = ""
		out[i] = q
	}

	return out
}

func TestMatrixNames(t *testing.T) {
	tb := synthTables(t)

	for _, name := range []string{"ab", "Abcdefghijklmno", "A-b", "a_b", "Zz"} {
		c := matrixBase(t, tb)
		c.Header.Name = name
		roundTripExact(t, "name "+name, c, tb)
	}

	// the 15 character name must fill the field with a terminator
	c := matrixBase(t, tb)
	c.Header.Name = strings.Repeat("A", MaxCharacterName)
	out := roundTripExact(t, "max name", c, tb)

	if out[nameOffset+15] != 0 {
		t.Fatal("name not terminated")
	}

	for _, bad := range []string{"", strings.Repeat("A", 16), "a\x00b", "x\xffy"} {
		c := matrixBase(t, tb)
		c.Header.Name = bad

		out, err := Write(c, tb)
		if err != nil {
			if bad == "" || !errors.Is(err, ErrBadName) {
				t.Fatalf("name %q: %v", bad, err)
			}

			continue
		}

		// accepted by Write: it must at least parse back to the same name
		back, err := Parse(out, tb)
		if err != nil || back.Header.Name != bad {
			t.Fatalf("name %q: wrote a file that does not parse back (%v)", bad, err)
		}
	}

	for name, wantOK := range map[string]bool{"ab": true, "a": false, "a-b": true, "-ab": false, "ab-": false,
		"a--b": false, "a1b": false, "a b": false, "ÄÖ": false} {
		if (ValidateName(name) == nil) != wantOK {
			t.Errorf("ValidateName(%q) = %v, want ok=%v", name, ValidateName(name), wantOK)
		}
	}
}

// TestMatrixNewCharacters writes a header-only file for every class and flag, round trips it and
// checks the status bits and the checksum.
func TestMatrixNewCharacters(t *testing.T) {
	for class := Amazon; class <= Assassin; class++ {
		for f := 0; f < 8; f++ {
			fl := NewCharacterFlags{Expansion: f&1 != 0, Hardcore: f&2 != 0, Ladder: f&4 != 0}

			data, err := NewCharacter("Newbie", class, fl, DefaultAppearance(class))
			if err != nil {
				t.Fatal(err)
			}

			c, err := Parse(data, nil)
			if err != nil {
				t.Fatalf("%v/%d: %v", class, f, err)
			}

			if !c.Header.IsNewCharacter() || c.Header.IsExpansion() != fl.Expansion ||
				c.Header.IsHardcore() != fl.Hardcore || c.Header.IsLadder() != fl.Ladder {
				t.Fatalf("%v/%d: flags %x", class, f, c.Header.Status)
			}

			out, err := Write(c, nil)
			if err != nil || !bytes.Equal(out, data) {
				t.Fatalf("%v/%d: new character round trip differs (%v)", class, f, err)
			}
		}
	}
}

// TestMatrixVersionBounds: versions outside 0x5C..0x60 are refused by both Parse and Write.
func TestMatrixVersionBounds(t *testing.T) {
	tb := synthTables(t)

	for _, v := range []uint32{0, 0x47, 0x57, 0x59, 0x5B, 0x61, 0x62, 0x63, 0xFFFFFFFF} {
		c := matrixBase(t, tb)
		c.Header.Version = v

		if _, err := Write(c, tb); !errors.Is(err, ErrBadVersion) {
			t.Fatalf("write v%X: %v", v, err)
		}

		data := buildBody(t, nil, [numSkills]byte{})
		binary.LittleEndian.PutUint32(data[4:], v)
		data = fixup(data)

		if _, err := Parse(data, tb); !errors.Is(err, ErrBadVersion) {
			t.Fatalf("parse v%X: %v", v, err)
		}
	}
}

// TestMatrixHardcoreDead: a dead hardcore character keeps its inventory through a write.
func TestMatrixHardcoreDead(t *testing.T) {
	tb := synthTables(t)
	c := matrixBase(t, tb)
	main, _, _, _ := matrixItems(t, tb)
	c.Items = main
	c.Header.Status = StatusExpansion | StatusHardcore | StatusDied | StatusLadder
	roundTripExact(t, "hc dead", c, tb)
}
