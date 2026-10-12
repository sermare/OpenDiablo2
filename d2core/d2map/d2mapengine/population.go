package d2mapengine

// PlannedMonster is one monster of a level's natural population, as decided
// by the real-map generator (d2mapgen, port of the original's room
// population). The game screen creates them through the monster director when
// it populates the level.
type PlannedMonster struct {
	// Key is the monstats key of the class to create.
	Key string
	// X, Y are the position in sub-tiles.
	X, Y int
	// Leader is the index (into the plan) of the pack leader this monster
	// follows, or -1.
	Leader int
	// Unique and Champion mark the pack leaders the original makes
	// unique (rare) / champion; the minions of a champion are champions too.
	Unique, Champion bool
	// Minion marks a follower of a unique, champion or super unique leader
	// (type bit 0x10); Mods are the monumod ids it carries (a unique's own
	// picks, a minion's inherited xfer ones).
	Minion bool
	Mods   []int
	// SuperKey is the superuniques.txt key of a super unique ("" otherwise) and
	// SuperIdx its hcIdx.
	SuperKey string
	SuperIdx int
	// Origin is 1 + the index (into the plan) of the unit this one was created around, 0 for a group leader (so the
	// zero value of a hand-made plan means "leader"). Unlike
	// Leader it is kept for the followers the original does not link (party packs without SetBoss, unlinked extras of
	// a super unique), so ranks and group counts can follow the whole tree of a pack.
	Origin int
	// Preset marks the units of a DS1 preset pack (place_unique_pack, place_champion, place_impgroup / miniongroup):
	// legal whatever the level's MonDen and monster rows say, so the natural-population checks leave them out.
	Preset bool
}

// SetPopulation stores the natural population of the loaded level. It is
// cleared by ResetMap.
func (m *MapEngine) SetPopulation(p []PlannedMonster) { m.population = p }

// Population returns the planned natural monsters of the loaded level, nil
// when the generator did not plan any (the old generators).
func (m *MapEngine) Population() []PlannedMonster { return m.population }

// PlannedObject is one random room object of a level (objgroup scatter of
// d2common/d2object.Scatter, the port of OBJRGN_SpawnRoomObjects).
type PlannedObject struct {
	// ID is the objects.txt row, X and Y the position in sub-tiles.
	ID, X, Y int
}

// SetRoomObjects stores the random room objects of the loaded level; the game
// screen creates them when it populates the level. Cleared by ResetMap.
func (m *MapEngine) SetRoomObjects(o []PlannedObject) { m.roomObjects = o }

// RoomObjects returns the planned random room objects, nil when none were planned.
func (m *MapEngine) RoomObjects() []PlannedObject { return m.roomObjects }
