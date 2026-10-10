package d2mp

import (
	"math"
	"math/rand"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
)

// Skill kinds.
const (
	SkillMelee   = iota // swing at an adjacent monster
	SkillMissile        // a missile flies from the caster
	SkillPortal         // opens a town portal pair
)

// Skill ids used by DefaultRules (the real Skills.txt ids of the skills; the
// numbers in SkillDef are placeholders).
const (
	SkillAttack     uint16 = 0
	SkillFireBolt   uint16 = 36
	SkillChargedBlt uint16 = 38
	SkillFireBall   uint16 = 47
	SkillTownPortal uint16 = 220
)

// SkillDef describes what a skill does in the simulation.
type SkillDef struct {
	Kind     int
	Speed    float64 // missile tiles per second
	Range    float64 // missile range in tiles
	Min, Max int     // damage (melee: scaled by the hero instead)
	Splash   float64 // radius of the area damage around the impact (0 = single target)
	Radius   float64 // missile collision radius
	Cooldown uint32  // ms between casts
}

// MonsterDef describes a monster type.
type MonsterDef struct {
	Name     string
	HP       int32
	Min, Max int
	Speed    float64 // tiles per second
	Aggro    float64 // tiles
	XP       int
	Level    int
	Cooldown uint32 // ms between attacks
}

// MonsterSpawn places one monster in a level.
type MonsterSpawn struct {
	Type uint16
	X, Y float64
}

// ObjectSpawn places one object in a level.
type ObjectSpawn struct {
	Type uint16
	X, Y float64
	Dest uint16 // portals: destination level
}

// LevelDef is the content of one level, derived from the level seed alone, so
// that every party that evaluates Rules.Level(seed, level) gets the same level.
type LevelDef struct {
	W, H     int
	Blocked  []bool // W*H, row major
	SpawnX   float64
	SpawnY   float64
	Monsters []MonsterSpawn
	Objects  []ObjectSpawn
}

// Walkable reports whether the tile position is inside the level and free.
func (l *LevelDef) Walkable(x, y float64) bool {
	ix, iy := int(math.Floor(x)), int(math.Floor(y))

	return ix >= 0 && iy >= 0 && ix < l.W && iy < l.H && !l.Blocked[iy*l.W+ix]
}

// Rules supplies the game data of the simulation.
type Rules interface {
	// Level returns the content of a level for the level seed.
	Level(levelSeed uint32, level uint16) *LevelDef
	// Skill returns the definition of a skill (ok false: unknown skill).
	Skill(id uint16) (SkillDef, bool)
	// Monster returns the definition of a monster type.
	Monster(typ uint16) MonsterDef
	// Hero returns the life and the melee damage range of a hero.
	Hero(class uint8, level uint8) (hp int32, min, max int)
	// Drops returns the item codes a dying monster drops.
	Drops(rng *rand.Rand, m MonsterDef) []string
	// ChestLoot returns the item codes a chest holds.
	ChestLoot(rng *rand.Rand) []string
}

// DefaultRules is a self-contained rule set for headless games and tests. All
// of its numbers are placeholders (UNVERIFIED): it exists so that the netcode
// can be exercised without game files.
type DefaultRules struct{}

var defaultMonsters = []MonsterDef{
	{}, // 0 unused
	{"Fallen", 30, 3, 6, 5.0, 9, 40, 2, 1200},
	{"Zombie", 60, 4, 9, 3.0, 8, 60, 3, 1500},
	{"Skeleton", 45, 5, 8, 4.5, 9, 55, 3, 1100},
	{"Goatman", 80, 6, 11, 4.0, 10, 90, 5, 1300},
	{"Quill Rat", 25, 2, 5, 6.0, 8, 30, 2, 900},
	{"Brute", 140, 8, 14, 4.0, 10, 150, 6, 1400},
}

// Monster implements Rules.
func (DefaultRules) Monster(typ uint16) MonsterDef {
	if int(typ) >= len(defaultMonsters) || typ == 0 {
		return defaultMonsters[1]
	}

	return defaultMonsters[typ]
}

// Skill implements Rules.
func (DefaultRules) Skill(id uint16) (SkillDef, bool) {
	switch id {
	case SkillAttack:
		return SkillDef{Kind: SkillMelee, Cooldown: 600}, true
	case SkillFireBolt:
		return SkillDef{Kind: SkillMissile, Speed: 18, Range: 20, Min: 6, Max: 12, Radius: 0.7, Cooldown: 400}, true
	case SkillChargedBlt:
		return SkillDef{Kind: SkillMissile, Speed: 14, Range: 12, Min: 4, Max: 8, Radius: 0.7, Cooldown: 400}, true
	case SkillFireBall:
		return SkillDef{Kind: SkillMissile, Speed: 14, Range: 22, Min: 15, Max: 25, Radius: 0.7, Splash: 2.0, Cooldown: 800}, true
	case SkillTownPortal:
		return SkillDef{Kind: SkillPortal, Cooldown: 1000}, true
	}

	return SkillDef{}, false
}

// Hero implements Rules.
func (DefaultRules) Hero(class uint8, level uint8) (int32, int, int) {
	l := int(level)
	if l < 1 {
		l = 1
	}

	return int32(60 + 12*l), 4 + l*4/5, 8 + l*7/5
}

var dropCodes = []string{"hp1", "mp1", "gld", "cap", "ssd", "buc", "jav", "lrg", "amu"}

// Drops implements Rules.
func (DefaultRules) Drops(rng *rand.Rand, m MonsterDef) []string {
	var out []string

	n := rng.Intn(3) // 0..2 items
	for i := 0; i < n; i++ {
		out = append(out, dropCodes[rng.Intn(len(dropCodes))])
	}

	return out
}

// ChestLoot implements Rules.
func (DefaultRules) ChestLoot(rng *rand.Rand) []string {
	return []string{dropCodes[rng.Intn(len(dropCodes))], dropCodes[rng.Intn(len(dropCodes))]}
}

const (
	defLevelSize = 96
	defSpawn     = 48.0
)

// Level implements Rules: a 96x96 area with scattered pillars, a ring of walls,
// monsters (not in towns), a chest, a waypoint where the level has one and
// exit portals (towns lead to the next level, other levels back).
func (DefaultRules) Level(seed uint32, level uint16) *LevelDef {
	rng := rand.New(rand.NewSource(int64(seed)))
	l := &LevelDef{W: defLevelSize, H: defLevelSize, SpawnX: defSpawn, SpawnY: defSpawn}
	l.Blocked = make([]bool, l.W*l.H)

	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			edge := x < 1 || y < 1 || x >= l.W-1 || y >= l.H-1
			near := math.Hypot(float64(x)-defSpawn, float64(y)-defSpawn) < 12
			l.Blocked[y*l.W+x] = edge || (!near && rng.Intn(100) < 6)
		}
	}

	free := func(minDist float64) (float64, float64) {
		for {
			x, y := float64(2+rng.Intn(l.W-4)), float64(2+rng.Intn(l.H-4))
			if l.Walkable(x, y) && math.Hypot(x-defSpawn, y-defSpawn) >= minDist {
				return x + 0.5, y + 0.5
			}
		}
	}

	town := d2level.IsTown(int(level))

	if !town {
		n := 10 + rng.Intn(5)
		for i := 0; i < n; i++ {
			x, y := free(14)
			l.Monsters = append(l.Monsters, MonsterSpawn{Type: uint16(1 + rng.Intn(len(defaultMonsters)-1)), X: x, Y: y})
		}
	}

	// objects near the spawn point have fixed places so scripts can find them
	l.Objects = append(l.Objects, ObjectSpawn{Type: ObjChest, X: defSpawn + 3.5, Y: defSpawn + 0.5})

	if _, ok := d2level.WaypointBit(int(level)); ok || town {
		l.Objects = append(l.Objects, ObjectSpawn{Type: ObjWaypoint, X: defSpawn + 0.5, Y: defSpawn + 4.5})
	}

	dest := uint16(level) - 1
	if town {
		dest = level + 1
	} else if act := d2level.ActOfLevel(int(level)); act > 0 && int(level) == d2level.ActStartLevel(act)+1 {
		dest = uint16(d2level.ActStartLevel(act))
	}

	l.Objects = append(l.Objects, ObjectSpawn{Type: ObjPortal, X: defSpawn - 4.5, Y: defSpawn + 0.5, Dest: dest})

	for i := 0; i < 2; i++ {
		x, y := free(16)
		l.Objects = append(l.Objects, ObjectSpawn{Type: ObjChest, X: x, Y: y})
	}

	return l
}
