package d2mp

import (
	"math/rand"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
)

// MonDummy is the monster type of the passive training dummies EngineRules
// places beside the town spawn (OUR numbering, above the DefaultRules monsters).
const MonDummy uint16 = 7

// EngineArena is the side of the square area EngineRules gives a level, in
// tiles: larger than the biggest town the engine generates, so the engine's
// own tile coordinates are valid simulation coordinates.
const EngineArena = 400

// EngineRules is the rule set used when the game screen of the engine plays
// through the realm. The engine keeps drawing and walking on its own map; the
// simulation only has to agree with it about where the heroes are and which
// monsters exist, so a level is an open arena in the ENGINE's tile
// coordinates, with the hero spawn where the engine puts its heroes.
//
// Everything else (damage, speeds, drops) is DefaultRules: placeholders
// (UNVERIFIED), see DefaultRules.
type EngineRules struct {
	DefaultRules
	// SpawnX and SpawnY are the engine's start tile (MapEngine.GetStartPosition).
	SpawnX, SpawnY float64
	// Dummies is the number of passive training dummies placed east of the
	// spawn in a town level (0 = none).
	Dummies int
}

// NewEngineRules returns the rules for an engine whose heroes start at the
// given tile. Heroes are sturdy: the engine keeps the real life totals.
func NewEngineRules(spawnX, spawnY float64, dummies int) EngineRules {
	return EngineRules{DefaultRules: DefaultRules{HeroLife: 5000}, SpawnX: spawnX, SpawnY: spawnY, Dummies: dummies}
}

// Level implements Rules: an open arena with a wall ring. Towns hold the
// dummies, other levels are the DefaultRules level moved to the engine spawn.
func (r EngineRules) Level(seed uint32, level uint16) *LevelDef {
	l := &LevelDef{W: EngineArena, H: EngineArena, SpawnX: Snap(r.SpawnX), SpawnY: Snap(r.SpawnY)}
	l.Blocked = make([]bool, l.W*l.H)

	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			l.Blocked[y*l.W+x] = x < 1 || y < 1 || x >= l.W-1 || y >= l.H-1
		}
	}

	if d2level.IsTown(int(level)) {
		for i := 0; i < r.Dummies; i++ {
			l.Monsters = append(l.Monsters, MonsterSpawn{Type: MonDummy, X: Snap(r.SpawnX) + 4 + 2*float64(i), Y: Snap(r.SpawnY) + float64(i%2)*2 - 1})
		}

		return l
	}

	// outside the towns: the placeholder monsters, placed around the spawn
	rng := rand.New(rand.NewSource(int64(seed)))
	for i := 0; i < 10; i++ {
		l.Monsters = append(l.Monsters, MonsterSpawn{
			Type: uint16(1 + rng.Intn(len(defaultMonsters)-1)),
			X:    Snap(r.SpawnX) + float64(14+rng.Intn(30)), Y: Snap(r.SpawnY) + float64(rng.Intn(40)-20) + 0.5})
	}

	return l
}

// Monster implements Rules: the dummies stand still and never attack.
func (r EngineRules) Monster(typ uint16) MonsterDef {
	if typ == MonDummy {
		return MonsterDef{Name: "Training Dummy", HP: 40, Min: 0, Max: 0, Speed: 0, Aggro: 0, XP: 5, Level: 1, Cooldown: 1000}
	}

	return r.DefaultRules.Monster(typ)
}
