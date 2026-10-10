package herogen

import (
	"fmt"
	"os"
	"path/filepath"

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

	return t, nil
}
