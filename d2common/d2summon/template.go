package d2summon

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
)

// Difficulty indexes the per-difficulty monstats columns.
type Difficulty int

// Difficulties.
const (
	Normal Difficulty = iota
	Nightmare
	Hell
)

var diffSuffix = [3]string{"", "(N)", "(H)"}

// Resist indexes Template.Res.
const (
	ResPhysical = iota
	ResMagic
	ResFire
	ResLightning
	ResCold
	ResPoison
	numRes
)

var resCols = [numRes]string{"ResDm", "ResMa", "ResFi", "ResLi", "ResCo", "ResPo"}

// Base holds the monstats numbers of one difficulty.
type Base struct {
	MinHP, MaxHP int
	AC           int
	A1Min, A1Max int
	A1TH         int
	Res          [numRes]int
}

// Template is one monstats row, the data a minion starts from. Summoned
// monsters have no Level column: their numbers are flat per difficulty and
// the skill level only enters through the summon modifiers (VERIFIED: the
// necroskeleton row is 21 life, the clay golem 100).
type Template struct {
	Class  int // hcIdx
	ID     string
	AI     string
	Walk   int
	Run    int
	Diff   [3]Base
	Skill1 string // monstats Skill1 (the skill traps and casters use)
}

// Templates indexes monstats rows by id (case insensitive) and class.
type Templates struct {
	byID    map[string]*Template
	byClass map[int]*Template
	monLvl  []levelRow
}

// levelRow is the AC and TH part of a monlvl.txt row for the expansion
// (L-AC, L-TH) columns, per difficulty.
type levelRow struct {
	AC, TH [3]int
}

// LoadMonLvl parses monlvl.txt (rows indexed by level) for LevelBonus.
func (t *Templates) LoadMonLvl(monlvl []byte) error {
	d := d2txt.LoadDataDictionary(monlvl)
	t.monLvl = nil

	for d.Next() {
		t.monLvl = append(t.monLvl, levelRow{
			AC: [3]int{d.Number("L-AC"), d.Number("L-AC(N)"), d.Number("L-AC(H)")},
			TH: [3]int{d.Number("L-TH"), d.Number("L-TH(N)"), d.Number("L-TH(H)")},
		})
	}

	if len(t.monLvl) == 0 {
		return fmt.Errorf("d2summon: no rows in monlvl")
	}

	return nil
}

// LevelBonus is the armor class and attack rating a summon of the given level
// receives. VERIFIED (SKILL_ComputeSummonLevel 0x5c2850): the level is capped
// to the last MonLvl row, the columns are the expansion ones (L-AC, L-TH) of
// the difficulty, and both are added to the minion as stats 0x1f and 0x13.
// ok is false when no monlvl was loaded.
func (t *Templates) LevelBonus(level int, diff Difficulty) (ac, th int, ok bool) {
	if t == nil || len(t.monLvl) == 0 {
		return 0, 0, false
	}

	if level >= len(t.monLvl) {
		level = len(t.monLvl) - 1
	}

	if level < 0 {
		level = 0
	}

	if diff < Normal || diff > Hell {
		diff = Normal
	}

	r := t.monLvl[level]

	return r.AC[diff], r.TH[diff], true
}

// LoadTemplates parses the contents of monstats.txt.
func LoadTemplates(monstats []byte) (*Templates, error) {
	d := d2txt.LoadDataDictionary(monstats)
	t := &Templates{byID: map[string]*Template{}, byClass: map[int]*Template{}}

	for d.Next() {
		id := d.String("Id")
		if id == "" {
			continue
		}

		tp := &Template{
			Class: d.Number("hcIdx"), ID: id, AI: d.String("AI"),
			Walk: d.Number("Velocity"), Run: d.Number("Run"), Skill1: d.String("Skill1"),
		}

		for i, s := range diffSuffix {
			b := &tp.Diff[i]
			b.MinHP = d.Number(hpCol("minHP", i))
			b.MaxHP = d.Number(hpCol("maxHP", i))
			b.AC = d.Number("AC" + s)
			b.A1Min = d.Number("A1MinD" + s)
			b.A1Max = d.Number("A1MaxD" + s)
			b.A1TH = d.Number("A1TH" + s)

			for r, col := range resCols {
				b.Res[r] = d.Number(col + s)
			}
		}

		t.byID[strings.ToLower(id)] = tp
		t.byClass[tp.Class] = tp
	}

	if len(t.byID) == 0 {
		return nil, fmt.Errorf("d2summon: no monsters in monstats")
	}

	return t, nil
}

// hpCol spells the HP column names: the normal columns are minHP/maxHP, the
// others MinHP(N)/MaxHP(H) (VERIFIED against patch_d2 monstats.txt).
func hpCol(base string, diff int) string {
	if diff == 0 {
		return base
	}

	return strings.ToUpper(base[:1]) + base[1:] + diffSuffix[diff]
}

// ByID returns the template for a monstats id such as "necroskeleton".
func (t *Templates) ByID(id string) (*Template, bool) {
	tp, ok := t.byID[strings.ToLower(id)]

	return tp, ok
}

// ByClass returns the template for a monstats hcIdx.
func (t *Templates) ByClass(class int) (*Template, bool) {
	tp, ok := t.byClass[class]

	return tp, ok
}

// AvgLife is the mean of the minimum and maximum life of a monster at a
// difficulty (the pair the skill tooltip's kind-13 line averages, 0x4e8ed0).
func (t *Templates) AvgLife(id string, diff Difficulty) (int, bool) {
	tp, ok := t.ByID(id)
	if !ok || diff < 0 || int(diff) >= len(tp.Diff) {
		return 0, false
	}

	b := tp.Diff[diff]

	return (b.MinHP + b.MaxHP) / 2, true
}

// Len is the number of rows.
func (t *Templates) Len() int { return len(t.byID) }
