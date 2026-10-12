package d2skills

import (
	"fmt"
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2pl2"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// State visuals (w49): what a unit looks like while states last. Two parts, both from states.txt:
//   - the colour shift (colorpri / colorshift: frozen blue, poisoned green, ...; the exe picks the state with the
//     highest colorpri in FUN_004d65a0 and remaps the unit's palette through PL2 hue row colorshift), drawn here as a
//     per-channel multiplier (d2pl2.HueTint);
//   - the persistent overlays (overlay1..overlay4: aura circles, curses, Frozen Armor, stun stars), one looping
//     d2mapentity.StateOverlay per state slot that follows the unit and goes when the state ends.
//
// UNVERIFIED: the draw order of the slots (see d2mapentity.StateOverlayOrder), overlay lights, and the exact tint.

// stateOverlaySlots is the number of overlay columns of states.txt.
const stateOverlaySlots = 4

type overlayKey struct {
	unit  string
	state string
	slot  int
}

// stateOverlayNames returns the overlay rows a state shows, slot by slot ("" and "null" are empty).
func stateOverlayNames(r *d2records.StateRecord) [stateOverlaySlots]string {
	var out [stateOverlaySlots]string

	if r == nil || r.NoOverlays {
		return out
	}

	for i, n := range []string{r.Overlay1, r.Overlay2, r.Overlay3, r.Overlay4} {
		if n != "" && n != "null" {
			out[i] = n
		}
	}

	return out
}

// hueTint returns the tint of a colour shift for the act's palette (cached).
func (e *Engine) hueTint(shift int) (r, g, b float64, ok bool) {
	act := e.actNow()

	if e.pl2 == nil || e.pl2Act != act {
		e.pl2, e.pl2Act = nil, act

		if data, err := e.asset.LoadFile(fmt.Sprintf("/data/global/palette/act%d/pal.pl2", act)); err == nil {
			if p, perr := d2pl2.Load(data); perr == nil {
				e.pl2 = p
			}
		}

		e.hueCache = map[int][4]float64{}
	}

	if c, has := e.hueCache[shift]; has {
		return c[0], c[1], c[2], c[3] > 0
	}

	r, g, b, ok = e.pl2.HueTint(shift)

	v := 0.0
	if ok {
		v = 1
	}

	e.hueCache[shift] = [4]float64{r, g, b, v}

	return r, g, b, ok
}

// tinted is what the visuals need of a unit.
type tintedUnit interface {
	SetStateTint(r, g, b float64, on bool)
	StateTint() (r, g, b float64, on bool)
	Label() string
}

// syncStateVisuals applies the colour shift and overlays of every unit holding states. It runs once per 25 Hz
// frame; units that lost all their states, died or left are cleaned up.
func (e *Engine) syncStateVisuals() {
	if e.asset == nil || e.mapEngine == nil || e.monsters == nil || e.visFrame == e.frame {
		return
	}

	e.visFrame = e.frame

	live := map[overlayKey]bool{}

	if len(e.sets) > 0 {
		byID := map[string]*d2mapentity.Monster{}

		for _, m := range e.monsters.Monsters() {
			byID[m.ID()] = m
		}

		ids := make([]string, 0, len(e.sets))
		for id := range e.sets {
			ids = append(ids, id)
		}

		sort.Strings(ids)

		for _, id := range ids {
			set := e.sets[id]

			var (
				t     tintedUnit
				x, y  float64
				alive = true
			)

			if m := byID[id]; m != nil {
				t, alive = m, m.Alive()
				x, y = m.Position.X(), m.Position.Y()
			} else if h := e.heroes[id]; h != nil {
				t = h.p
				x, y = h.p.Position.X(), h.p.Position.Y()
			} else {
				continue
			}

			if !alive {
				t.SetStateTint(1, 1, 1, false)

				continue
			}

			e.syncUnitTint(set, t)

			for _, name := range set.Names(e.frame) {
				rec := e.asset.Records.States[name]

				for slot, ov := range stateOverlayNames(rec) {
					if ov == "" {
						continue
					}

					k := overlayKey{id, name, slot}
					live[k] = true

					e.keepStateOverlay(k, ov, x, y)
				}
			}
		}
	}

	for k, ent := range e.stateFx {
		if !live[k] {
			if ent != nil {
				e.mapEngine.RemoveEntity(ent)
			}

			delete(e.stateFx, k)
		}
	}
}

func (e *Engine) syncUnitTint(set *d2state.Set, t tintedUnit) {
	shift, ok := set.ColorShift(e.frame)
	if ok {
		if r, g, b, got := e.hueTint(shift); got {
			if _, _, _, was := t.StateTint(); !was {
				e.emit("state", "STATE tint unit=%s shift=%d mult=%.2f,%.2f,%.2f", t.Label(), shift, r, g, b)
			}

			t.SetStateTint(r, g, b, true)

			return
		}
	}

	t.SetStateTint(1, 1, 1, false)
}

// keepStateOverlay creates the overlay entity of a state slot when missing and moves it onto the unit.
func (e *Engine) keepStateOverlay(k overlayKey, overlay string, x, y float64) {
	ent := e.stateFx[k]

	if ent == nil {
		rec := e.asset.Records.Layout.Overlays[overlay]
		if rec == nil || rec.Filename == "" || rec.Filename == "null" {
			e.stateFx[k] = nil // unknown or empty: remembered so the lookup is not repeated

			return
		}

		var err error

		if ent, err = e.mapEngine.NewStateOverlay(x, y, rec, d2mapentity.StateOverlayOrder(k.slot, rec.PreDraw)); err != nil {
			e.Debugf("no graphic for state overlay %s: %v", overlay, err)
			e.stateFx[k] = nil

			return
		}

		e.stateFx[k] = ent
		e.mapEngine.AddEntity(ent)
		e.emit("state", "STATE overlay on unit=%s state=%s overlay=%s slot=%d", k.unit, k.state, overlay, k.slot)
	}

	if ent != nil {
		ent.Follow(x, y)
	}
}
