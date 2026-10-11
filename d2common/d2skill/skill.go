package d2skill

import (
	"sort"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2calc"
)

// DamageSpec holds the damage columns skills.txt and missiles.txt share:
// HitShift, SrcDam, MinDam.., EType, EMin.., ELen.. and the synergy calcs.
type DamageSpec struct {
	HitShift   int
	SrcDam     int
	MinDam     int
	MaxDam     int
	MinLevDam  [5]int
	MaxLevDam  [5]int
	DmgSymPer  *d2calc.Program
	EType      string
	EMin, EMax int
	EMinLev    [5]int
	EMaxLev    [5]int
	EDmgSymPer *d2calc.Program
	ELen       int
	ELevLen    [3]int
	ELenSymPer *d2calc.Program
}

// Skill is a skills.txt row, reduced to what the pipeline reads. Params and
// Calc are 1-based (index 0 unused).
type Skill struct {
	ID        int
	Name      string
	CharClass string

	// flags
	InTown       bool
	UseManaOnDo  bool
	DecQuant     bool
	Lob          bool
	Passive      bool
	Aura         bool
	Progressive  bool
	Kick         bool
	NoAmmo       bool
	AttackNoMana bool
	// AllowTownRoom is skills.txt dwBits4 & 0x100: the skill may target a town room (Bone Wall otherwise refused).
	AllowTownRoom bool
	// Bits4 is the raw dwBits4 word when known (interrupt.go reads bit 31).
	Bits4 uint32

	SrvStFunc, SrvDoFunc int
	SrvMissile           string
	SrvMissileA          string
	SrvMissileB          string
	SrvMissileC          string
	LineOfSight          int
	MaxLvl               int

	StartMana, MinMana, ManaShift, Mana, LvlMana int

	Delay *d2calc.Program

	ToHit, LevToHit int
	ToHitCalc       *d2calc.Program
	ResultFlags     int
	HitFlags        int
	HitClass        int

	Params [9]int
	Calc   [5]*d2calc.Program

	DamageSpec

	AuraFilter      int
	AuraState       string
	AuraTargetState string
	AuraLenCalc     *d2calc.Program
	AuraRangeCalc   *d2calc.Program
	AuraStat        [7]string
	AuraStatCalc    [7]*d2calc.Program

	PassiveState string
	// PassiveIType is passiveitype: the weapon type a mastery's stats are keyed to.
	PassiveIType string
	// IType1 is itypea1, the weapon type the skill needs (the throw family test of 0x646a90).
	IType1      string
	PassiveStat [6]string
	PassiveCalc [6]*d2calc.Program

	PetMax   *d2calc.Program
	Skpoints *d2calc.Program

	// Summons (class skills): the monstats key, pet type group, mode and the
	// monster skills the summon uses.
	Summon       string
	PetType      string
	SumMode      string
	SumSkill     [6]string // 1-based
	TargetCorpse bool
	Periodic     bool
	PerDelay     *d2calc.Program
	// Range is the skills.txt range column (h2h, rng, none...).
	Range string
}

// Registry is a skill lookup table.
type Registry struct {
	byID   map[int]*Skill
	byName map[string]*Skill
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{byID: map[int]*Skill{}, byName: map[string]*Skill{}}
}

// Add registers a skill.
func (r *Registry) Add(s *Skill) {
	r.byID[s.ID] = s
	r.byName[strings.ToLower(s.Name)] = s
}

// ByID returns a skill by id or nil.
func (r *Registry) ByID(id int) *Skill { return r.byID[id] }

// ByName returns a skill by (case-insensitive) name or nil.
func (r *Registry) ByName(name string) *Skill { return r.byName[strings.ToLower(name)] }

// Len is the number of skills.
func (r *Registry) Len() int { return len(r.byID) }

// IDs lists the skill ids in ascending order.
func (r *Registry) IDs() []int {
	ids := make([]int, 0, len(r.byID))
	for id := range r.byID {
		ids = append(ids, id)
	}

	sort.Ints(ids)

	return ids
}
