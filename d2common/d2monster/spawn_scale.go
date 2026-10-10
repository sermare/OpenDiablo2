package d2monster

// ScaleTable is what ScaleSpawnClass reads of monstats and of the level.
// Class ids are monstats row indices; -1 / out of range means "no class".
type ScaleTable interface {
	// Known reports whether the class id is a valid monstats row.
	Known(class int) bool
	// BaseID is the BaseId column of a class (its class chain head).
	BaseID(class int) int
	// NextInClass is the NextInClass column of a class (-1: none).
	NextInClass(class int) int
	// Level is the monstats Level column (normal difficulty) of a class.
	Level(class int) int
	// ChainSteps is the step bound the exe reads from monstats record bytes
	// +0x4a (MONSTATS_GetRecordBytes4A4B 0x652550). UNVERIFIED: the column
	// is computed at load time and its meaning is taken to be the number of
	// NextInClass links behind the class.
	ChainSteps(class int) int
}

// ScaleSpawnClass is MONSTER_ComputeSpawnRangeForLevel 0x63fcb0 (VERIFIED
// reading of the decompile): it scales a class the InvisoSpawner lays to the
// area. levelMons are the level's monster-type classes (mon1.. of the level
// record, normal-difficulty columns: the function reads only those) and
// monLvl is the level's MonLvl1Ex.
//
//  1. No monster types on the level: the class is returned unchanged.
//  2. A level monster type whose BaseId equals the BaseId of the class is
//     returned (the first one in list order).
//  3. Otherwise the NextInClass chain is walked, starting with the NextInClass
//     of the base record (not of the class), at most ChainSteps(class) steps,
//     and stops before the first class whose Level exceeds monLvl+1 (or whose
//     id is invalid); the last class reached is returned (the original class
//     when the first step already fails).
func ScaleSpawnClass(t ScaleTable, class int, levelMons []int, monLvl int) int {
	if len(levelMons) == 0 {
		return class
	}

	base := class
	if t.Known(class) {
		base = t.BaseID(class)
	}

	if !t.Known(base) {
		return class
	}

	for _, m := range levelMons {
		b := m
		if t.Known(m) {
			b = t.BaseID(m)
		}

		if b == base {
			return m
		}
	}

	next := t.NextInClass(base)
	cur := class

	for i := 0; i < t.ChainSteps(class); i++ {
		if !t.Known(next) || monLvl+1 < t.Level(next) {
			return cur
		}

		cur = next
		next = t.NextInClass(next)
	}

	return cur
}
