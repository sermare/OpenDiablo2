package d2player

import "math"

// Test-only motion watches for OD2_AUTOSCRIPT scenarios. A scenario that waits a fixed time after a click is
// load sensitive: the first walk of a run loads animation files lazily, and on a busy machine that stalls the
// frame loop for the whole wait. The commands heromoved / herostill arm a watch that logs one line
// ("HERO moved ..." / "HERO still ...") when the condition holds, and the scenario waits for that line with
// until:, however slow the machine is.
const (
	// heroWatchMovedTiles is how far the hero must get from where heromoved was armed.
	heroWatchMovedTiles = 0.5
	// heroWatchStillSeconds is how long (game seconds) the hero must not move for herostill.
	heroWatchStillSeconds = 0.5
	// heroWatchStillEpsilon is the largest step still counted as standing.
	heroWatchStillEpsilon = 0.01
)

type heroWatchKind int

const (
	heroWatchNone heroWatchKind = iota
	heroWatchMoved
	heroWatchStill
)

// heroWatch decides, from the hero position of each update, whether its condition holds.
type heroWatch struct {
	kind   heroWatchKind
	x0, y0 float64 // moved: where it was armed; still: the last position
	still  float64 // still: seconds without moving
}

func (w *heroWatch) arm(kind heroWatchKind, x, y float64) {
	*w = heroWatch{kind: kind, x0: x, y0: y}
}

// update feeds one update (the position and the game seconds it took) and reports the condition, once:
// the watch disarms itself when it fires.
func (w *heroWatch) update(x, y, elapsed float64) (fired bool, dist float64) {
	switch w.kind {
	case heroWatchMoved:
		dist = math.Hypot(x-w.x0, y-w.y0)
		if dist >= heroWatchMovedTiles {
			w.kind = heroWatchNone
			return true, dist
		}
	case heroWatchStill:
		if math.Hypot(x-w.x0, y-w.y0) > heroWatchStillEpsilon {
			w.x0, w.y0, w.still = x, y, 0
			return false, 0
		}

		w.still += elapsed
		if w.still >= heroWatchStillSeconds {
			w.kind = heroWatchNone
			return true, 0
		}
	}

	return false, dist
}

// advanceHeroWatch logs the line of an armed watch (see the constants above).
func (g *GameControls) advanceHeroWatch(elapsed float64) {
	kind := g.heroWatch.kind
	if kind == heroWatchNone || g.hero == nil {
		return
	}

	p := g.hero.Position.World()
	if fired, dist := g.heroWatch.update(p.X(), p.Y(), elapsed); fired {
		if kind == heroWatchMoved {
			g.watchLogf("HERO moved dist=%.2f pos=(%.2f,%.2f)", dist, p.X(), p.Y())
		} else {
			g.watchLogf("HERO still pos=(%.2f,%.2f)", p.X(), p.Y())
		}
	}
}

// SetWatchLog sets where the watch lines go. The scenario runner only sees the log of the game screen, not the
// one of this package, so the game screen passes its own.
func (g *GameControls) SetWatchLog(f func(format string, args ...interface{})) { g.watchLog = f }

func (g *GameControls) watchLogf(format string, args ...interface{}) {
	if g.watchLog != nil {
		g.watchLog(format, args...)
		return
	}

	g.Infof(format, args...)
}
