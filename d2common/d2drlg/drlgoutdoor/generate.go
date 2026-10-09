package drlgoutdoor

import (
	"errors"
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// Generate runs DRLG_GenerateOutdoorLevel for an Act 1 outdoor level
// (Levels 2-7, 0x11 Burial Grounds and 0x27 Moo Moo Farm in the golden): the
// boundary polygon, then DRLG_GenerateAct1Outdoors, then the cell-to-room pass.
func Generate(env *Env, p Params) (lv *Level, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case GameError:
				err = e
			case error:
				err = e
			default:
				err = fmt.Errorf("drlgoutdoor: level %d: %v", p.ID, r)
			}

			lv = nil
		}
	}()

	rec, ok := env.Tables.Level(p.ID)
	if !ok {
		return nil, fmt.Errorf("drlgoutdoor: level %d unknown", p.ID)
	}

	switch rec.LevelType {
	case 2, 0x10, 0x15, 0x16, 0x1b, 0x1c, 0x1e, 0x1f: // Act 1 wilderness, Act 2 desert, Act 3 jungle and Kurast, Act 4 mesa/chaos, Act 5 siege/snow
	default:
		return nil, errors.New("drlgoutdoor: LevelType not ported")
	}

	l := &Level{Params: p, LType: rec.LevelType, env: env, ctr: map[int]*counter{}, town: p.Town}
	l.Seed = newLevelSeed(p)
	l.OdFlags = p.OdFlags
	l.W, l.H = p.Rect.W>>3, p.Rect.H>>3
	l.Def, l.GridB, l.Flag, l.GridD = NewGrid(l.W, l.H), NewGrid(l.W, l.H), NewGrid(l.W, l.H), NewGrid(l.W, l.H)
	l.Poly = buildPolygon(p.Rect, p.Neighbors)

	for _, v := range l.verts() {
		l.PolygonAtAct = append(l.PolygonAtAct, Vertex{X: v.X, Y: v.Y, B: v.B, F: v.F})
	}

	var gen func() error

	switch rec.LevelType {
	case 0x10:
		gen = l.generateAct2
	case 0x15, 0x16:
		gen = l.generateAct3
	case 0x1b, 0x1c:
		gen = func() error { l.generateAct4(); return nil }
	case 0x1e, 0x1f:
		gen = func() error { l.generateAct5(); return nil }
	default:
		gen = l.generateAct1
	}

	if err := gen(); err != nil {
		return nil, err
	}

	l.trace("end")
	l.createRooms()
	l.Counters = map[int][2]int{}

	for d, c := range l.ctr {
		l.Counters[d] = [2]int{c.n, c.ctr}
	}

	return l, nil
}

// generateAct1 is DRLG_GenerateAct1Outdoors (0x683800) minus the room pass.
func (l *Level) generateAct1() error {
	id := l.Params.ID

	if id != 2 && id != 3 && id != 0x11 {
		l.trace("concave")
		l.concave()
	}

	l.trace("markexit")
	l.markExits()
	l.trace("edges")
	l.drawBoundaryEdges()

	if id >= 2 && id <= 7 {
		l.trace("lvlsub")
		l.applyLvlSub(0, 4)
		l.trace("caveent")

		if err := l.caveEntrances(); err != nil {
			return err
		}

		l.trace("lvlsub")
		l.applyLvlSub(1, 4)
		l.trace("lvlsub")
		l.applyLvlSub(2, 4)
		l.trace("towntrans")

		if err := l.townTransitions(); err != nil {
			return err
		}

		l.trace("lvlsub")
		l.applyLvlSub(3, 4)
		l.trace("rivers")
		l.placeRoads()
	}

	if id == 0x27 {
		for t := 0; t < 4; t++ {
			l.trace("lvlsub")
			l.applyLvlSub(t, 4)
		}
	}

	if id >= 3 && id <= 6 {
		l.trace("wpmark")
		l.markWaypoint()
	}

	if id >= 2 && id <= 7 {
		l.trace("scatter")
		l.scatterShrines(5)
	}

	l.trace("specials")
	l.placeSpecials()

	return nil
}

func newLevelSeed(p Params) *d2rand.Seed { return d2rand.New(p.BaseSeed + uint32(p.ID)) }
