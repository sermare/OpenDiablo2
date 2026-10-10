package d2monreg

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// The monster modifier table (monumod.txt, the 0x20-byte records at
// DataTables+0xc50, count at +0xc54) and the exact modifier picks of the
// original.
//
// VERIFIED against the decompilation (Game.exe 1.14b):
//
//	0x59e330 MONSTER_RollUniqueModifiers   champion decision and the modifier count
//	0x59e0e0 MONSTER_PickRandomBossUMod    weighted pick among the champion rows
//	0x59e1e0 MONSTER_PickRandomNonBossUMod weighted pick among the other rows
//	0x59dfc0 MONSTER_IsUModEligibleForMonster  the per-class eligibility
//	0x59e4c0 MONSTER_CopyLeaderUModsToMinion   only rows with the xfer column set
//
// Record layout (read from the exe): +4 version (ushort), +6 enabled, +7 xfer,
// +8 champion, +9 fPick, +0xa two excluded monster types (ushort), +0xe the
// three cpick weights (short, per difficulty), +0x14 the three upick weights,
// +0x1c the constant (row 0 holds the champion chance).
//
// UNVERIFIED: the type test of exclude1/exclude2 (0x59dc50) reads a bitmask
// table built from montype.txt; it is modelled as "the class' MonType is the
// excluded type or lists it (transitively) in equiv1..3".

// UMod is one monumod.txt row.
type UMod struct {
	Name     string
	Enabled  bool
	Version  int // 100 = expansion only
	Xfer     bool
	Champion bool
	FPick    int
	Exclude  [2]int // montype row indices, 0 = none
	CPick    [3]int
	UPick    [3]int
	Constant int
}

// UModTable is the monumod.txt table.
type UModTable struct {
	Rows []UMod
	// equiv[t] lists the montype rows t is an equivalent of (equiv1..3).
	equiv [][]int
}

// MaxUMods is the number of modifier ids a unit stores (9 bytes at +0x1c).
const MaxUMods = 9

// maxPickRows is the cap of the pick lists (the exe stops adding at 256).
const maxPickRows = 256

// LoadUMods parses monumod.txt and montype.txt and resolves the MonType of
// every class. The table is then used by the unique / champion roll.
func (t *Tables) LoadUMods(monumod, montype []byte) error {
	mt, err := readTSV(montype)
	if err != nil {
		return fmt.Errorf("montype: %w", err)
	}

	names := map[string]int{}

	for i, r := range mt.rows {
		n := strings.ToLower(mt.str(r, "type"))
		if _, dup := names[n]; !dup && n != "" {
			names[n] = i
		}
	}

	u := &UModTable{equiv: make([][]int, len(mt.rows))}

	for i, r := range mt.rows {
		for _, c := range []string{"equiv1", "equiv2", "equiv3"} {
			if e, ok := names[strings.ToLower(mt.str(r, c))]; ok {
				u.equiv[i] = append(u.equiv[i], e)
			}
		}
	}

	mm, err := readTSV(monumod)
	if err != nil {
		return fmt.Errorf("monumod: %w", err)
	}

	for _, r := range mm.rows {
		m := UMod{Name: mm.str(r, "uniquemod"), Enabled: mm.num(r, "enabled") != 0, Version: mm.num(r, "version"),
			Xfer: mm.num(r, "xfer") != 0, Champion: mm.num(r, "champion") != 0, FPick: mm.num(r, "fPick"),
			Constant: mm.num(r, "constants")}
		m.CPick = [3]int{mm.num(r, "cpick"), mm.num(r, "cpick (N)"), mm.num(r, "cpick (H)")}
		m.UPick = [3]int{mm.num(r, "upick"), mm.num(r, "upick (N)"), mm.num(r, "upick (H)")}

		for i, c := range []string{"exclude1", "exclude2"} {
			m.Exclude[i] = names[strings.ToLower(mm.str(r, c))] // unknown / empty = 0 (ends the list)
		}

		u.Rows = append(u.Rows, m)
	}

	for i := range t.Mons {
		t.Mons[i].Type = -1

		if n := strings.ToLower(t.Mons[i].TypeName); n != "" {
			if id, ok := names[n]; ok {
				t.Mons[i].Type = id
			}
		}
	}

	t.UMods = u

	return nil
}

// ChampionChance is the champion percentage (constant of row 0, 20 in the
// stock table); 0 without a table.
func (u *UModTable) ChampionChance() int {
	if u == nil || len(u.Rows) == 0 {
		return 0
	}

	return u.Rows[0].Constant
}

// typeIs reports whether montype row t is e or (transitively) lists e as an equivalent.
func (u *UModTable) typeIs(t, e int) bool {
	seen := map[int]bool{}

	var walk func(x int) bool

	walk = func(x int) bool {
		if x == e {
			return true
		}

		if x < 0 || x >= len(u.equiv) || seen[x] {
			return false
		}

		seen[x] = true

		for _, p := range u.equiv[x] {
			if walk(p) {
				return true
			}
		}

		return false
	}

	return walk(t)
}

// Eligible is MONSTER_IsUModEligibleForMonster for row i and a class. expansion
// is the flag at game+0x70.
func (u *UModTable) Eligible(t *Tables, i, class int, expansion bool) bool {
	r := &u.Rows[i]
	if !r.Enabled || (!expansion && r.Version >= 100) {
		return false
	}

	var mon *Mon

	if class >= 0 && class < len(t.Mons) {
		mon = &t.Mons[class]
	}

	for _, e := range r.Exclude {
		if e < 1 {
			break
		}

		if mon != nil && mon.Type >= 0 && u.typeIs(mon.Type, e) {
			return false
		}
	}

	hasMode := func(mode uint) bool {
		if mon == nil || mon.Ex < 0 || mon.Ex >= len(t.Mon2s) {
			return false
		}

		return t.Mon2s[mon.Ex].Modes&(1<<mode) != 0
	}

	switch r.FPick {
	case 1:
		return hasMode(4)
	case 2:
		// melee-only classes and classes with nomultishot cannot take it
		return mon == nil || !(mon.Has(FlagIsMelee) || mon.Has(FlagNoMultiShot))
	case 3:
		return hasMode(2)
	}

	return true
}

type weighted struct{ row, w int }

// pick draws one candidate by weight with the unit's own seed.
func pick(seed *d2rand.Seed, cands []weighted, total int) int {
	r := int(seed.Roll(int32(total)))

	for _, c := range cands {
		if r < c.w {
			return c.row
		}

		r -= c.w
	}

	return 0
}

// PickBoss is MONSTER_PickRandomBossUMod: a row among the champion rows
// (champion column set) with a positive cpick weight of the difficulty, 0 when
// none. One seed step is taken when there is any weight.
func (u *UModTable) PickBoss(t *Tables, seed *d2rand.Seed, diff, class int, expansion bool) int {
	var cands []weighted

	total := 0

	for i := range u.Rows {
		r := &u.Rows[i]
		if w := r.CPick[diff]; w > 0 && r.Champion && u.Eligible(t, i, class, expansion) {
			total += w

			cands = append(cands, weighted{i, w})

			if len(cands) > maxPickRows-1 {
				break
			}
		}
	}

	return pick(seed, cands, total)
}

// PickNonBoss is MONSTER_PickRandomNonBossUMod: a row among the non-champion
// rows with a positive upick weight that is not in used, 0 when none. One seed
// step is taken when there is any weight.
func (u *UModTable) PickNonBoss(t *Tables, seed *d2rand.Seed, diff, class int, expansion bool, used map[int]bool) int {
	var cands []weighted

	total := 0

	for i := range u.Rows {
		r := &u.Rows[i]
		if w := r.UPick[diff]; w > 0 && !r.Champion && u.Eligible(t, i, class, expansion) && !used[i] {
			total += w

			cands = append(cands, weighted{i, w})

			if len(cands) > maxPickRows-1 {
				break
			}
		}
	}

	return pick(seed, cands, total)
}

// Roll is MONSTER_RollUniqueModifiers on a unit seed: with the champion flag a
// roll(100) below the champion chance makes a champion carrying one champion
// row; otherwise 1 + difficulty (a Roll(1) step first, capped to 9 mods in all)
// non-champion rows are picked. have are the ids the unit already carries;
// the result is the full list.
func (u *UModTable) Roll(t *Tables, seed *d2rand.Seed, diff, class int, expansion, allowChampion bool, have []int) (mods []int, champion bool) {
	mods = append([]int(nil), have...)
	if len(mods) >= MaxUMods-1 {
		return mods, false
	}

	if allowChampion && int(seed.Roll(100)) < u.ChampionChance() {
		if id := u.PickBoss(t, seed, diff, class, expansion); id != 0 {
			mods = append(mods, id)
		}

		return mods, true
	}

	n := int(seed.Roll(1)) + 1 + diff
	if MaxUMods-1 < n+len(mods) {
		n = MaxUMods - len(mods)
	}

	used := map[int]bool{}
	for _, m := range mods {
		used[m] = true
	}

	for ; n > 0; n-- {
		id := u.PickNonBoss(t, seed, diff, class, expansion, used)
		if id == 0 {
			break
		}

		mods = append(mods, id)
		used[id] = true
	}

	return mods, false
}

// XferMods is MONSTER_CopyLeaderUModsToMinion: the leader's modifiers whose
// row has the xfer column set.
func (u *UModTable) XferMods(leader []int) []int {
	var out []int

	for _, id := range leader {
		if id > 0 && id < len(u.Rows) && u.Rows[id].Xfer {
			out = append(out, id)
		}
	}

	return out
}
