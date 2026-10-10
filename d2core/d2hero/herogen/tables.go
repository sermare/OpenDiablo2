package herogen

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2herostats"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// Tables are the game tables the generator reads: the save format tables of
// the d2s package, the item generator and the class and experience tables.
// They come from a folder of extracted tables (the D2_TABLES layout):
//
//	armor.txt weapons.txt misc.txt ItemTypes.txt ItemStatCost.txt (or itemstatcost.bin)
//	CharStats.txt Experience.txt skills/patch_d2/skills.txt itemgen/{patch_d2,d2exp,d2data}/*.txt
type Tables struct {
	Dir     string
	Save    *d2s.ItemTables
	Creator *d2drop.Creator
	Classes map[string]d2statlist.Class
	Exp     map[string]*d2herostats.ExpTable
	Bases   d2statlist.Bases
	// EquipRules and EquipBases are the game's equip rules (body locations, class, strength,
	// dexterity and level requirements, hands) the worn items are checked with.
	EquipRules d2equip.Rules
	EquipBases d2equip.Bases
	// Skills is skills.txt by id; nil when the file is missing (the skill checks are then skipped).
	Skills map[int]SkillRow
	// BeltTypes is the armor.txt "belt" column of the belts by code (it picks the Belts.txt row, so the
	// number of potion cells, see d2inventory.BeltBoxes).
	BeltTypes map[string]int
}

// parseBeltTypes reads the belt column of armor.txt for the rows that have one.
func parseBeltTypes(armor []byte) map[string]int {
	out := map[string]int{}
	lines := strings.Split(strings.ReplaceAll(string(armor), "\r", ""), "\n")

	if len(lines) == 0 {
		return out
	}

	col := map[string]int{}

	for i, h := range strings.Split(lines[0], "\t") {
		if _, dup := col[h]; !dup {
			col[h] = i
		}
	}

	ci, ok1 := col["code"]
	ti, ok2 := col["type"]
	bi, ok3 := col["belt"]

	if !ok1 || !ok2 || !ok3 {
		return out
	}

	for _, l := range lines[1:] {
		f := strings.Split(l, "\t")
		if len(f) <= ci || len(f) <= ti || len(f) <= bi || f[ti] != "belt" {
			continue
		}

		if n, err := strconv.Atoi(f[bi]); err == nil {
			out[f[ci]] = n
		}
	}

	return out
}

// LoadTables reads the tables from dir.
func LoadTables(dir string) (*Tables, error) {
	read := func(names ...string) ([]byte, error) {
		var last error

		for _, n := range names {
			b, err := os.ReadFile(filepath.Join(dir, n))
			if err == nil {
				return b, nil
			}

			last = err
		}

		return nil, last
	}

	files := map[string][]byte{}

	for _, f := range [][]string{
		{"itemstatcost.bin", "ItemStatCost.txt"}, {"armor.txt"}, {"weapons.txt"}, {"misc.txt"}, {"ItemTypes.txt"},
		{"CharStats.txt"}, {"Experience.txt"},
	} {
		b, err := read(f...)
		if err != nil {
			return nil, fmt.Errorf("herogen: %w", err)
		}

		files[f[0]] = b
	}

	t := &Tables{Dir: dir}

	var err error

	if t.Save, err = d2s.NewItemTables(files["itemstatcost.bin"], files["armor.txt"], files["weapons.txt"],
		files["misc.txt"], files["ItemTypes.txt"]); err != nil {
		return nil, fmt.Errorf("herogen: save tables: %w", err)
	}

	if t.Creator, err = d2drop.LoadCreator(d2drop.DirTables{Root: dir}); err != nil {
		return nil, fmt.Errorf("herogen: item generator: %w", err)
	}

	if t.Classes, err = d2statlist.ParseClasses(files["CharStats.txt"]); err != nil {
		return nil, fmt.Errorf("herogen: CharStats: %w", err)
	}

	if t.Exp, err = d2herostats.ParseExperience(files["Experience.txt"]); err != nil {
		return nil, fmt.Errorf("herogen: Experience: %w", err)
	}

	if t.Bases, err = d2statlist.ParseBases(files["armor.txt"], files["weapons.txt"]); err != nil {
		return nil, fmt.Errorf("herogen: bases: %w", err)
	}

	types, err := d2equip.ParseTypes(files["ItemTypes.txt"])
	if err != nil {
		return nil, fmt.Errorf("herogen: ItemTypes: %w", err)
	}

	t.EquipRules = d2equip.Rules{Types: types}

	if t.EquipBases, err = d2equip.ParseBases(files["armor.txt"], files["weapons.txt"], files["misc.txt"]); err != nil {
		return nil, fmt.Errorf("herogen: equip bases: %w", err)
	}

	t.BeltTypes = parseBeltTypes(files["armor.txt"])

	if t.Skills, err = loadSkills(dir); err != nil {
		return nil, err
	}

	return t, nil
}
