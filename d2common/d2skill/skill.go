// Package d2skill is the server side skill cast pipeline: it turns a skills.txt
// row plus a caster into mana payments, cooldowns, missiles, melee hits and
// state effects, the way SKILL_ServerRunStartFunc (0x56d4e0) and
// SKILL_ServerRunSkillFunc (0x56d6a0) do (skills-2.md, sections 1 to 5).
//
// The package is pure: the caster, the map and the engine are interfaces, the
// skill data are plain structs a loader fills in from skills.txt (see
// d2records.RecordManager.SkillTable), and every calc column is a compiled
// d2calc.Program. Missiles are created in a d2missile.Sim.
//
// Implemented skills (by srvdofunc / srvstfunc): the generic missile path
// (Fire Bolt, Ice Bolt, Magic Arrow, Fire Arrow, Cold Arrow, Fire Ball, Glacial
// Spike, Bone Spear, Holy Bolt, javelins...), Charged Bolt (do 17), Attack/Kick/
// Bash/Stun/Jab and the other plain melee skills (do 1, 2, 7), Throw (do 3),
// Frozen Armor and the other timed self states (do 18, 25, 47, 54, 116), Static
// Field (do 20), Inner Sight (do 6), the Nova rings (do 22) and Warmth
// (passive). Telekinesis and Find Item are not implemented.
//
// Class skills (class.go): a table maps each srvdofunc to a handler that reads
// the skills.txt columns, so skills sharing a function share code. Barbarian:
// Double Swing, Frenzy, Leap, Leap Attack, War Cry, Shout / Battle Orders, Taunt,
// Double Throw. Amazon: Multiple Shot, Guided Arrow, Charged Strike, Strafe,
// Fend, Lightning Strike. Necromancer: the curses (do 30, 59, 61), Teeth, Bone
// Armor, Poison Dagger, Corpse Explosion, Poison Explosion, Bone Wall / Prison,
// Raise Skeleton and the golems (summon orders). Paladin: Sacrifice, Smite, Zeal,
// Charge, Vengeance, the auras (do 65, 66, 81, 82). Sorceress: Chain Lightning,
// Fire Wall, Blaze, Energy Shield, Teleport, Meteor, Blizzard, Thunder Storm,
// Inferno. Druid: Firestorm, Twister, Tornado, Hurricane, Armageddon, Volcano,
// Raven, spirit wolves and totems, the forms, Maul / Feral Rage, Rabies.
// Assassin: the charge builders and finishers, Fire / Shock traps, sentries,
// Shadow Warrior, Psychic Hammer, Mind Blast.
//
// Outcomes that are not a missile or a melee strike come back as Effects
// (states, curses, auras, area hits, strikes, storms, summons, movement) for the
// engine to apply; d2state holds what a state means. Handlers cite the notes'
// function name and mark every behaviour that was inferred from the skills.txt
// columns rather than read from the binary with U. See Pipeline for what each
// does and what is unverified.
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
	PassiveStat  [6]string
	PassiveCalc  [6]*d2calc.Program

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
