package d2gamescreen

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2boss"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2cube"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2uber"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
)

// The Pandemonium event in the running game: the cube opens the portals
// (keys -> an uber area, organs -> Tristram), entering those levels spawns the
// uber bosses (real monstats rows with their own AIs), their kills drop the
// organs and the final reward. The decisions are made by d2common/d2uber; this
// file carries out its actions.

const uberSpawnOffset = 12 // sub-tiles from the hero

type uberRuntime struct {
	mgr     *d2boss.Manager
	ev      *d2uber.Event
	frames  float64
	spawned []*d2mapentity.Monster
}

// uberRT returns the event runtime, creating it on first use.
func (v *Game) uberRT() *uberRuntime {
	if v.uber != nil {
		return v.uber
	}

	u := &uberRuntime{}
	u.mgr = d2boss.New(func(s string) { v.Infof("%s", s) })
	u.ev = d2uber.New(u.mgr)

	if os.Getenv("OD2_AUTOUBER") == "" {
		u.ev.Pick = rand.Intn // nolint:gosec // a game choice, not security
	}

	v.uber = u

	return u
}

// advanceUber runs the event's timers (the arrival of Diablo and Baal).
func (v *Game) advanceUber(elapsed float64) {
	u := v.uber
	if u == nil {
		return
	}

	u.frames += elapsed * questFrameRate

	if n := int(u.frames); n > 0 {
		u.frames -= float64(n)
		v.uberApply(u.mgr.Tick(n))
	}
}

// isUberLevel is a level of the event (133..136).
func isUberLevel(level int) bool {
	_, ok := d2uber.AreaOf(level)

	return ok || level == d2uber.LevelTristram
}

// uberEnter tells the event the hero arrived in a level.
func (v *Game) uberEnter(level int) {
	if v.uber == nil && !isUberLevel(level) {
		return
	}

	u := v.uberRT()
	v.localHeroSub(u)
	v.uberApply(u.mgr.Enter(level))
}

func (v *Game) localHeroSub(u *uberRuntime) {
	if v.localPlayer != nil {
		u.mgr.HeroX, u.mgr.HeroY = v.heroSub()
	}
}

// uberKilled feeds a monster death to the event.
func (v *Game) uberKilled(ev d2monsters.KillEvent) {
	if v.uber == nil {
		return
	}

	v.uberApply(v.uber.mgr.Killed(d2boss.Kill{Class: ev.Class, Super: -1, Name: ev.Label}))
}

// uberKeyDrop drops the Pandemonium key of a Hell boss (UNVERIFIED source).
func (v *Game) uberKeyDrop(class int) {
	if code, ok := d2uber.KeyDrop(class, v.questDifficulty()); ok {
		v.Infof("UBER key %s dropped by class %d", code, class)
		v.spawnQuestItem(code)
	}
}

// uberCube handles the portal recipes of the cube; it returns the number of
// actions carried out.
func (v *Game) uberCube(rec d2cube.Recipe) (int, bool) {
	u := v.uberRT()
	v.localHeroSub(u)

	as, mine := u.ev.UseCube(u.mgr, rec)
	v.uberApply(as)

	return len(as), mine
}

// uberApply carries out the manager's actions.
func (v *Game) uberApply(actions []d2boss.Action) {
	for _, a := range actions {
		switch a.Kind {
		case d2boss.ActSpawnMonster:
			v.uberSpawn(a)
		case d2boss.ActPortal:
			if err := v.commandSpawnPortal([]string{fmt.Sprint(a.Level)}); err != nil {
				v.Infof("UBER portal %q to level %d failed: %v", a.Name, a.Level, err)
			} else {
				v.Infof("UBER portal %q opened to level %d", a.Name, a.Level)
			}
		case d2boss.ActDropItem:
			v.Infof("UBER drop %s (%s)", a.Key, a.Name)
			v.spawnQuestItem(a.Key)
		default:
			v.Infof("UBER action %s", a)
		}
	}
}

// uberSpawn creates an uber boss with the Director.
func (v *Game) uberSpawn(a d2boss.Action) {
	d := v.monsters
	if d == nil {
		v.Infof("UBER spawn %s skipped: no monster director", a.Name)

		return
	}

	st := d.FindStat(a.Key)
	if st == nil {
		v.Infof("UBER spawn %s failed: no monstats row %q", a.Name, a.Key)

		return
	}

	hx, hy := v.heroSub()

	m, err := d.SpawnNear(st, hx+uberSpawnOffset, hy-3, 3)
	if err != nil {
		v.Infof("UBER spawn %s failed: %v", a.Name, err)

		return
	}

	v.uber.spawned = append(v.uber.spawned, m)

	ai := ""
	if b := d.BrainOf(m); b != nil {
		ai = b.Profile.AI
	}

	v.Infof("UBER spawn name=%q boss=%s class=%d key=%s ai=%s level=%d", m.Label(), a.Name, st.ID, a.Key, ai, a.Level)
}
