package d2monsters

import (
	"math"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// Shot is a ranged attack in flight. The Director builds it and hands it to a
// Launcher; the launcher only moves it. Whether a cell holds something the
// shot hits is the Director's business (Collide), what happens then too
// (Impact), so any projectile engine can be plugged in through Launcher
// without touching the combat rules.
type Shot struct {
	// Owner is the brain id of the monster that fired.
	Owner uint32
	// Missile is the missiles.txt name the attack uses ("" when unknown).
	Missile string
	// Mode is the animation mode the attack was made in.
	Mode string
	// From and To are subtile positions; the shot flies along the line from
	// From through To and keeps going Overshoot subtiles past To.
	From, To d2path.Point
	// Velocity is subtiles per 25 Hz frame (monstats/missiles Vel / 16,
	// VERIFIED conversion in the notes; the fallback when a missile is unknown
	// is DefaultShotVelocity, an engine choice).
	Velocity float64
	// Range is the maximum flight distance in subtiles (0 = to the aim point plus a few subtiles).
	Range int
	// Collide is called once for every new subtile the shot enters; returning
	// true stops the shot there ("hit a unit").
	Collide func(x, y int) bool
	// Impact is called exactly once when the shot ends: hit is true when
	// Collide stopped it, false when it hit a wall or ran out of range.
	Impact func(x, y int, hit bool)
}

// Launcher flies shots. The built-in implementation is a straight-line bolt;
// a missile engine (feat/skill-pipeline) can replace it with SetLauncher by
// implementing these two methods.
type Launcher interface {
	// Launch starts a shot; it returns false if the shot cannot be flown.
	Launch(s Shot) bool
	// Step advances every shot in flight by one game frame.
	Step()
	// InFlight is the number of shots currently flying (autotest summary).
	InFlight() int
}

// Defaults of the built-in launcher (engine choices, UNVERIFIED).
const (
	DefaultShotVelocity = 1.5 // subtiles per frame
	shotOvershoot       = 8
	shotSubstep         = 0.5 // subtiles; finer than a cell so no cell is skipped
)

// boltLauncher flies shots in straight lines over a collision grid; walls
// (FlagWall, VERIFIED to block missiles) stop them.
type boltLauncher struct {
	grid  d2path.Grid
	shots []*bolt
}

type bolt struct {
	s          Shot
	x, y       float64
	dx, dy     float64 // unit direction
	travelled  float64
	limit      float64
	cur        d2path.Point
	stepLength float64
}

// newBoltLauncher creates the built-in launcher.
func newBoltLauncher(g d2path.Grid) *boltLauncher { return &boltLauncher{grid: g} }

// Launch implements Launcher.
func (l *boltLauncher) Launch(s Shot) bool {
	dx, dy := float64(s.To.X-s.From.X), float64(s.To.Y-s.From.Y)
	dist := math.Hypot(dx, dy)

	if dist == 0 || s.Impact == nil {
		return false
	}

	vel := s.Velocity
	if vel <= 0 {
		vel = DefaultShotVelocity
	}

	// without a range the shot flies to the aim point and a little past it
	limit := float64(s.Range)
	if limit <= 0 {
		limit = dist + shotOvershoot
	}

	l.shots = append(l.shots, &bolt{
		s: s, x: float64(s.From.X) + 0.5, y: float64(s.From.Y) + 0.5,
		dx: dx / dist, dy: dy / dist, limit: limit, cur: s.From, stepLength: vel,
	})

	return true
}

// InFlight implements Launcher.
func (l *boltLauncher) InFlight() int { return len(l.shots) }

// Step implements Launcher.
func (l *boltLauncher) Step() {
	live := l.shots[:0]

	for _, b := range l.shots {
		if !l.advance(b) {
			live = append(live, b)
		}
	}

	for i := len(live); i < len(l.shots); i++ {
		l.shots[i] = nil
	}

	l.shots = live
}

// advance moves a bolt one frame and reports whether it ended.
func (l *boltLauncher) advance(b *bolt) bool {
	for moved := 0.0; moved < b.stepLength; moved += shotSubstep {
		step := math.Min(shotSubstep, b.stepLength-moved)
		b.x += b.dx * step
		b.y += b.dy * step
		b.travelled += step

		cell := d2path.Point{X: int(math.Floor(b.x)), Y: int(math.Floor(b.y))}

		if cell != b.cur {
			b.cur = cell

			if d2path.Blocked(l.grid, cell.X, cell.Y, d2path.FlagWall) {
				b.s.Impact(cell.X, cell.Y, false)

				return true
			}

			if b.s.Collide != nil && b.s.Collide(cell.X, cell.Y) {
				b.s.Impact(cell.X, cell.Y, true)

				return true
			}
		}

		if b.travelled >= b.limit {
			b.s.Impact(cell.X, cell.Y, false)

			return true
		}
	}

	return false
}
