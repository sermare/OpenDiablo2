package d2monster

// Difficulty indexes the per-difficulty columns (game byte +0x6d).
type Difficulty int

// Difficulties.
const (
	Normal Difficulty = iota
	Nightmare
	Hell
)

// DefaultAggroDistance is used when monstats aidist is 0 (VERIFIED, 0x35 = 35).
const DefaultAggroDistance = 35

// MaxAggroDistance is the hard cap on the aggro scan (VERIFIED, 0x37 = 55).
const MaxAggroDistance = 55

// NumSkills is the number of Skill slots in monstats.
const NumSkills = 8

// SkillSlot is one of the monstats Skill1..8 / Sk1mode..Sk8mode entries.
type SkillSlot struct {
	Name  string // skills.txt name, empty when the slot is unused
	Mode  Mode   // animation mode the skill is cast in (0 when unknown)
	Level int
}

// Used reports whether the slot holds a skill (Skill id >= 0 in the exe).
func (s SkillSlot) Used() bool { return s.Name != "" }

// Profile is the difficulty-resolved AI data of one monster class: what the
// think functions read from monstats.txt.
type Profile struct {
	Class  int    // monstats hcIdx
	ID     string // monstats Id
	AI     string // monai name ("Skeleton", "Fallen", ...)
	AIDel  int    // aidel
	AIDist int    // aidist (aggro radius, 0 = default)
	Threat int
	// AIP holds aip1..aip8 at indices 1..8 (index 0 unused).
	AIP    [9]int
	Skills [NumSkills]SkillSlot
	// Melee is monstats flag bit 1 (isMelee, class flag test 0x452b20 bit 1).
	// UNVERIFIED mapping of the bit's name; the bit index is VERIFIED.
	Melee bool
	// NoWalk is true when the class lacks the WL mode (monstats2 mode byte
	// +0xf0 bit 2, test 0x467af0). The zero value (can walk) is the usual case.
	NoWalk bool
	Walk   int // Velocity
	// ChaseReach is the stop distance the CorruptLancer passes to "run to target
	// within" (monstats2 byte +0xe, UNVERIFIED meaning); 0 when unknown.
	ChaseReach int
	Run        int // Run velocity
	// Level and combat numbers are read by the engine, not by the AI.
}

// Aggro returns the effective aggro radius: aidist (default 35 when 0)
// limited to the hard cap of 55.
func (p *Profile) Aggro() int {
	d := p.AIDist
	if d == 0 {
		d = DefaultAggroDistance
	}

	if d > MaxAggroDistance {
		d = MaxAggroDistance
	}

	return d
}

// ProfileSource resolves a monster class to its profile. The engine adapts
// the loaded monstats records to it.
type ProfileSource interface {
	// Profile returns the profile for a class id and difficulty.
	Profile(class int, diff Difficulty) (*Profile, bool)
}
