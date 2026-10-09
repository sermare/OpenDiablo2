package d2monster

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
)

// TxtClasses is a ClassSource built from monstats.txt contents.
type TxtClasses struct {
	byClass map[int]ClassInfo
	byID    map[string]int
}

// LoadTxtClasses parses the spawn-related columns of monstats.txt. Unset
// minion columns become -1 (class 0 is a real class).
func LoadTxtClasses(monstats []byte) (*TxtClasses, error) {
	d := d2txt.LoadDataDictionary(monstats)
	t := &TxtClasses{byClass: map[int]ClassInfo{}, byID: map[string]int{}}

	type pending struct {
		class  int
		m1, m2 string
	}

	var fix []pending

	for d.Next() {
		c := ClassInfo{
			Class: d.Number("hcIdx"), Key: d.String("Id"),
			MinGrp: d.Number("MinGrp"), MaxGrp: d.Number("MaxGrp"),
			PartyMin: d.Number("PartyMin"), PartyMax: d.Number("PartyMax"),
			Rarity:  d.Number("Rarity"),
			IsSpawn: d.Number("isSpawn") != 0, Ranged: d.Number("rangedtype") != 0,
			NoRatio: d.Number("noRatio") != 0, Boss: d.Number("boss") != 0,
			SetBoss: d.Number("SetBoss") != 0, BossXfer: d.Number("BossXfer") != 0,
			Minion1: -1, Minion2: -1,
		}

		t.byClass[c.Class] = c
		t.byID[c.Key] = c.Class
		fix = append(fix, pending{c.Class, d.String("minion1"), d.String("minion2")})
	}

	if d.Err != nil {
		return nil, d.Err
	}

	for _, f := range fix {
		c := t.byClass[f.class]

		if id, ok := t.byID[f.m1]; ok {
			c.Minion1 = id
		}

		if id, ok := t.byID[f.m2]; ok {
			c.Minion2 = id
		}

		t.byClass[f.class] = c
	}

	return t, nil
}

// Class implements ClassSource.
func (t *TxtClasses) Class(id int) (ClassInfo, bool) {
	c, ok := t.byClass[id]

	return c, ok
}

// ByID resolves a monstats Id ("fallen1").
func (t *TxtClasses) ByID(id string) (ClassInfo, bool) {
	c, ok := t.byID[id]
	if !ok {
		return ClassInfo{}, false
	}

	return t.byClass[c], true
}

// SuperUnique is a row of superuniques.txt (the columns the group planner uses).
type SuperUnique struct {
	Key, Name, Class string
	HcIdx            int
	MinGrp, MaxGrp   int
}

// LoadTxtSuperUniques parses superuniques.txt contents.
func LoadTxtSuperUniques(buf []byte) ([]SuperUnique, error) {
	d := d2txt.LoadDataDictionary(buf)

	var out []SuperUnique

	for d.Next() {
		out = append(out, SuperUnique{
			Key: d.String("Superunique"), Name: d.String("Name"), Class: d.String("Class"),
			HcIdx:  d.Number("hcIdx"),
			MinGrp: d.Number("MinGrp"), MaxGrp: d.Number("MaxGrp"),
		})
	}

	return out, d.Err
}
