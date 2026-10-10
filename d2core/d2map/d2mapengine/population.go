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
	// unique / champion (the modifiers are not applied by the engine yet).
	Unique, Champion bool
}

// SetPopulation stores the natural population of the loaded level. It is
// cleared by ResetMap.
func (m *MapEngine) SetPopulation(p []PlannedMonster) { m.population = p }

// Population returns the planned natural monsters of the loaded level, nil
// when the generator did not plan any (the old generators).
func (m *MapEngine) Population() []PlannedMonster { return m.population }
