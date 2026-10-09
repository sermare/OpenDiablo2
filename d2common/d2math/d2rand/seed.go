package d2rand

const (
	multiplier  uint64 = 0x6AC690C5
	initialHigh uint32 = 0x29A // 666: the high word right after seeding
)

// Seed is the generator state: a low and a high 32-bit word.
type Seed struct {
	Lo, Hi uint32
}

// New returns a generator seeded like the game's InitSeed.
func New(seed uint32) *Seed {
	s := &Seed{}
	s.Init(seed)

	return s
}

// Init resets the generator: the low word is the seed and the high word is
// always 0x29A.
func (s *Seed) Init(seed uint32) {
	s.Lo, s.Hi = seed, initialHigh
}

// Step advances the generator and returns the new low word.
func (s *Seed) Step() uint32 {
	t := uint64(s.Lo)*multiplier + uint64(s.Hi)
	s.Lo, s.Hi = uint32(t), uint32(t>>32)

	return s.Lo
}

// Roll returns a number in [0, n). For n < 1 it returns 0 without advancing
// the generator; for any n >= 1 (including 1) it advances it exactly once.
// Powers of two use a mask, anything else an unsigned modulo, as in the game.
func (s *Seed) Roll(n int32) uint32 {
	if n < 1 {
		return 0
	}

	v := s.Step()

	if n&(n-1) == 0 {
		return v & uint32(n-1)
	}

	return v % uint32(n)
}

// Chance reports a fair coin flip: one step, the lowest bit.
func (s *Seed) Chance() bool {
	return s.Step()&1 == 1
}

// DrlgBaseSeed derives the base seed of a game's dungeon: seed the generator
// with the game seed, take one step, and keep the resulting low word. The
// returned state is the generator after that step, from which act-specific
// extras are drawn.
func DrlgBaseSeed(gameSeed uint32) (base uint32, drlg *Seed) {
	drlg = New(gameSeed)
	base = drlg.Step()

	return base, drlg
}

// LevelSeed returns the seed a level is generated from. Levels are always
// seeded from the base seed plus their id, so generation order does not matter.
func LevelSeed(base uint32, levelID uint32) *Seed {
	return New(base + levelID)
}

// NewRoomSeed derives the seed of a new room from its level's seed, the way
// the game allocates rooms: step the level seed once, seed the room from the
// new low word, then step the room once; the low word after that step is the
// room's identifying seed value.
func NewRoomSeed(level *Seed) (room *Seed, roomValue uint32) {
	room = New(level.Step())
	roomValue = room.Step()

	return room, roomValue
}
