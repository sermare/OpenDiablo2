package d2state

import (
	"errors"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
)

// Def is the part of a states.txt row that decides how a state behaves in a
// Set. The state id is the row number (none = 0, freeze = 1, poison = 2,
// cold = 11, stunned = 21, ...), which matches the ids the exe uses (0x2f
// sanctuary, 0x73 burning, 0x36 uninterruptable, 0x6d...).
type Def struct {
	ID   int
	Name string
	// Group: verified, the column is read by Set.ClearGroup's exe twin
	// 0x56a480 (buff casts) and the monster AI check 0x5ea850.
	Group int
	// Curse: a new curse replaces the old one (U, see Defs.exclusive).
	Curse bool
	// RemHit: the state ends when its unit is hit (cloak_of_shadows only).
	RemHit bool
	Aura   bool
	// PlrStayDeath, MonStayDeath and BossStayDeath: the state survives the
	// death of a player / monster / boss; every other state is cleared.
	// Meaning inferred from the shipped data (freeze keeps MonStayDeath so
	// frozen corpses stay blue; alignment, sync_warped and corpse_noselect
	// keep PlrStayDeath). U in the exe.
	PlrStayDeath, MonStayDeath, BossStayDeath bool
	// Shatter: a unit that dies with this state shatters (freeze).
	Shatter bool
	// Blue is the states.txt column. Verified (0x4d65a0): the client's colour
	// choice does not read it; only ColorPri and ColorShift decide (the
	// column is used elsewhere, not located).
	Blue bool
	// ColorPri and ColorShift pick the palette shift of the unit: the state
	// with the highest ColorPri wins, ties go to the lowest id (verified,
	// FUN_004d65a0 scans ids upward with a strict greater-than from priority
	// 0 and ignores id 0). ColorShift
	// indexes the PL2 HueVariations (111 entries). 0 ColorPri = no shift.
	ColorPri, ColorShift int
	Overlay1             string
	Stat                 string
}

// Defs maps a state name (lower case, as skills.txt writes it) to its row.
type Defs map[string]Def

var errBadTable = errors.New("d2state: unreadable states table")

// ParseDefs reads a states.txt. Rows get their row number as id.
func ParseDefs(data []byte) (defs Defs, err error) {
	defer func() {
		if r := recover(); r != nil { // d2txt panics on an empty file
			defs, err = nil, errBadTable
		}
	}()

	d := d2txt.LoadDataDictionary(data)
	defs = Defs{}
	id := 0

	for d.Next() {
		name := d.String("state")
		defs[name] = Def{
			ID: id, Name: name, Group: d.Number("group"),
			Curse: d.Number("curse") > 0, RemHit: d.Number("remhit") > 0, Aura: d.Number("aura") > 0,
			PlrStayDeath: d.Number("plrstaydeath") > 0, MonStayDeath: d.Number("monstaydeath") > 0,
			BossStayDeath: d.Number("bossstaydeath") > 0, Shatter: d.Number("shatter") > 0,
			Blue: d.Number("blue") > 0, ColorPri: d.Number("colorpri"), ColorShift: d.Number("colorshift"),
			Overlay1: d.String("overlay1"), Stat: d.String("stat"),
		}
		id++
	}

	return defs, d.Err
}

// exclusive reports whether applying state a must end the active state b
// (both are curses; the group column is handled by Set.ClearGroup, which the
// cast code calls, not by the timed-state function). Shrine states carry the curse
// flag in states.txt but are not skill curses; they are left alone (U).
func (d Defs) exclusive(a, b string) bool {
	da, oka := d[a]
	db, okb := d[b]

	if !oka || !okb || a == b {
		return false
	}

	return da.Curse && db.Curse && !isShrine(a) && !isShrine(b)
}

func isShrine(name string) bool { return len(name) > 7 && name[:7] == "shrine_" }

// stays reports whether a state's statlist survives the death of a unit of
// a kind ("player", "monster" or "boss"). States without a row are cleared.
// Verified (0x63b5b0): players use plrstaydeath, every monster including a
// boss uses monstaydeath.
func (d Defs) stays(name, kind string) bool {
	df, ok := d[name]
	if !ok {
		return false
	}

	if kind == "player" {
		return df.PlrStayDeath
	}

	return df.MonStayDeath
}

// BitStays reports whether the state bit (the visual, sent to clients)
// survives death: the death routines 0x57d310 and 0x5a3f20 clear the unit's
// bits with the plr mask for players, the mon mask for monsters and the boss
// mask for monsters whose record passes MONSTER_IsStatRecordFlag40 (kind
// "boss").
func (d Defs) BitStays(name, kind string) bool {
	df, ok := d[name]
	if !ok {
		return false
	}

	switch kind {
	case "player":
		return df.PlrStayDeath
	case "boss":
		return df.BossStayDeath
	}

	return df.MonStayDeath
}
