package d2gamescreen

import (
	"fmt"
	"strconv"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2boss"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
)

// The Chaos Sanctuary (level 108) in the running game: the five seals are real
// objects of the level (objects.txt 392..396, placed by the DS1 pieces of the
// sanctuary), operating one reports it to the Diablo encounter of d2common/d2boss
// (the seal bosses Infector of Souls, Lord De Seis and Grand Vizier of Chaos
// appear at the seal dummy positions, Diablo arrives at the "diablo start"
// dummy 255 once all five seals are open and the three seal bosses are dead).
// The decisions are made by d2boss.Seals; this file carries out its actions
// with real monsters of the Director. The Baal / Duriel / Mephisto encounters
// of the same package are not wired here.

type chaosRuntime struct {
	mgr    *d2boss.Manager
	frames float64
	// super maps the leaders of the seal-boss packs to their super unique row.
	super map[*d2mapentity.Monster]int
	// diabloAt is the subtile of the dummy object 255 of the level.
	diabloAt    d2path.Point
	hasDiabloAt bool
	diablo      *d2mapentity.Monster
}

// commandRestoreVitals is the console command "restorevitals": the hero's life
// and mana are full again (debug; the scripted fights of the boss scenarios
// have no belt of mana potions to drink).
func (v *Game) commandRestoreVitals(_ []string) error {
	if v.localPlayer == nil || v.localPlayer.Stats == nil {
		return fmt.Errorf("restorevitals needs a game")
	}

	s := v.localPlayer.Stats
	s.Health, s.Mana = s.MaxHealth, s.MaxMana
	v.Infof("VITALS restored life=%d mana=%d", s.Health, s.Mana)

	return nil
}

// chaosSpawnSearch is how far (subtiles) a seal boss may be moved to reachable ground.
const chaosSpawnSearch = 60

// nearestReachable searches rings of growing size around p for a reachable
// sub-tile.
func nearestReachable(r *d2mapengine.Reachable, p d2path.Point, radius int) (d2path.Point, bool) {
	for ring := 0; ring <= radius; ring++ {
		for dy := -ring; dy <= ring; dy++ {
			for dx := -ring; dx <= ring; dx++ {
				if (dx == ring || dx == -ring || dy == ring || dy == -ring) && r.At(p.X+dx, p.Y+dy) {
					return d2path.Point{X: p.X + dx, Y: p.Y + dy}, true
				}
			}
		}
	}

	return p, false
}

// isSealObject is one of the five seals of the Chaos Sanctuary.
func isSealObject(id int) bool { return id >= d2boss.ObjSealVizier && id <= d2boss.ObjSealInfector }

func (v *Game) chaosRT() *chaosRuntime {
	if v.chaos == nil {
		v.chaos = &chaosRuntime{
			mgr:   d2boss.New(func(s string) { v.Infof("%s", s) }),
			super: map[*d2mapentity.Monster]int{},
		}
	}

	return v.chaos
}

// chaosEnter tells the encounter the hero arrived in a level; in the Chaos
// Sanctuary it also reads the position of the Diablo start dummy.
func (v *Game) chaosEnter(level int) {
	if level != d2boss.LevelChaos {
		return
	}

	rt := v.chaosRT()
	rt.hasDiabloAt = false

	seals := 0

	for _, e := range v.gameClient.MapEngine.Entities() {
		ob, ok := e.(*d2mapentity.Object)
		if !ok {
			continue
		}

		switch id := ob.Record().Index; {
		case id == d2boss.ObjDiabloStart:
			rt.diabloAt = d2path.Point{X: int(ob.Position.X()), Y: int(ob.Position.Y())}
			rt.hasDiabloAt = true
		case isSealObject(id):
			seals++
		}
	}

	v.Infof("CHAOS entered: %d seals, diablo start dummy %v at %v", seals, rt.hasDiabloAt, rt.diabloAt)
	v.chaosApply(rt.mgr.Enter(level))
}

// advanceChaos runs the encounter's timers (the arrival of Diablo).
func (v *Game) advanceChaos(elapsed float64) {
	rt := v.chaos
	if rt == nil {
		return
	}

	rt.frames += elapsed * questFrameRate

	if n := int(rt.frames); n > 0 {
		rt.frames -= float64(n)
		rt.mgr.HeroX, rt.mgr.HeroY = v.heroSub()
		v.chaosApply(rt.mgr.Tick(n))
	}
}

// chaosOperate is a seal used by the hero.
func (v *Game) chaosOperate(ob *d2mapentity.Object) {
	if v.questRT != nil && v.questRT.area != d2boss.LevelChaos {
		return
	}

	rt := v.chaosRT()
	id := ob.Record().Index

	if _, err := ob.Open(); err != nil {
		v.Infof("CHAOS seal %d animation: %v", id, err)
	}

	x, y := int(ob.Position.X()), int(ob.Position.Y())
	v.Infof("CHAOS seal operated id=%d at subtile (%d,%d): %s", id, x, y, rt.mgr.States())
	v.chaosApply(rt.mgr.Operate(d2boss.Operate{Object: id, X: x, Y: y}))
	v.Infof("CHAOS state after seal %d: %s", id, rt.mgr.States())
}

// chaosKilled feeds a monster death to the encounter.
func (v *Game) chaosKilled(ev d2monsters.KillEvent) {
	rt := v.chaos
	if rt == nil {
		return
	}

	super := -1
	if s, ok := rt.super[ev.Monster]; ok {
		super = s
	}

	v.chaosApply(rt.mgr.Killed(d2boss.Kill{Class: ev.Class, Super: super, Name: ev.Label, Level: d2boss.LevelChaos}))
}

// chaosApply carries out the actions of the encounter.
func (v *Game) chaosApply(actions []d2boss.Action) {
	for _, a := range actions {
		switch a.Kind {
		case d2boss.ActSpawnMonster:
			v.chaosSpawn(a)
		case d2boss.ActPurge:
			v.chaosPurge()
		default:
			v.Infof("CHAOS action %s", a)
		}
	}
}

// superKeyOfRow returns the superuniques.txt key of a hardcoded index.
func (v *Game) superKeyOfRow(row int) (string, bool) {
	for key, rec := range v.asset.Records.Monster.Unique.Super {
		if rec.HcIdx == strconv.Itoa(row) {
			return key, true
		}
	}

	return "", false
}

func (v *Game) chaosSpawn(a d2boss.Action) {
	d := v.monsterDirector()
	if d == nil {
		v.Errorf("CHAOS spawn %s: no monster director", a.Name)
		return
	}

	rt := v.chaosRT()
	hx, hy := v.heroSub()

	if a.Super >= 0 { // a seal boss with its pack, at the dummy next to the seal
		key, ok := v.superKeyOfRow(a.Super)
		if !ok {
			v.Errorf("CHAOS spawn %s: no super unique with index %d", a.Name, a.Super)
			return
		}

		at := d2path.Point{X: a.X, Y: a.Y}
		if a.X == 0 && a.Y == 0 {
			at = d2path.Point{X: hx + bossHeroOff, Y: hy}
		}

		// the exe's dummy offset can fall behind a wall of the DS1 piece as built here (the Grand Vizier's
		// dummy lies beyond the south wall of the west arm): the boss then appears on the nearest ground the
		// hero can reach (UNVERIFIED which of the two is right in the original)
		if r := v.gameClient.MapEngine.ReachableFrom(hx, hy); r != nil && !r.At(at.X, at.Y) {
			if p, found := nearestReachable(r, at, chaosSpawnSearch); found {
				v.Infof("CHAOS spawn %s: the dummy (%d,%d) is not reachable, using (%d,%d)", a.Name, at.X, at.Y, p.X, p.Y)
				at = p
			}
		}

		res, err := d.SpawnSuperUnique(key, at)
		if err != nil {
			v.Errorf("CHAOS spawn %s: %v", a.Name, err)
			return
		}

		rt.super[res.Leader] = a.Super
		v.Infof("CHAOS spawn seal boss %q key=%s super=%d at (%d,%d) pack=%d", res.Leader.Label(), key, a.Super, at.X, at.Y,
			len(res.Monsters))

		return
	}

	st := d.FindStat(strconv.Itoa(a.Class))
	if st == nil {
		v.Errorf("CHAOS spawn %s: no monstats row %d", a.Name, a.Class)
		return
	}

	x, y := hx+bossHeroOff, hy

	if rt.hasDiabloAt {
		x, y = rt.diabloAt.X, rt.diabloAt.Y
	}

	if r := v.gameClient.MapEngine.ReachableFrom(hx, hy); r != nil && !r.At(x, y) {
		if p, found := nearestReachable(r, d2path.Point{X: x, Y: y}, chaosSpawnSearch); found {
			x, y = p.X, p.Y
		}
	}

	m, err := d.SpawnNear(st, x, y, 3)
	if err != nil {
		v.Errorf("CHAOS spawn %s: %v", a.Name, err)
		return
	}

	if a.Class == d2boss.ClassDiablo {
		rt.diablo = m
	}

	sx, sy := m.SubtilePos()
	v.Infof("CHAOS spawn %q class=%d at subtile %s dummy=%v", m.Label(), a.Class, fmt.Sprint(sx, ",", sy), rt.hasDiabloAt)
}

// chaosPurge sends the other monsters of the level away when Diablo is
// summoned (the exe puts every living non-pet monster except Diablo into its
// death mode; here they vanish without a kill, so they feed no counters).
func (v *Game) chaosPurge() {
	d := v.monsterDirector()
	if d == nil {
		return
	}

	n := 0

	for _, m := range d.Monsters() {
		if !m.Alive() || m.MonstatID() == d2boss.ClassDiablo {
			continue
		}

		if b := d.BrainOf(m); b != nil {
			d.Dismiss(b)

			n++
		}
	}

	v.Infof("CHAOS purge: %d monsters of the level are gone", n)
}

// commandSetMana is the console command "setmana <n>": the hero's mana is n
// (at most the maximum; debug, the natural regeneration scenario starts empty).
func (v *Game) commandSetMana(args []string) error {
	if len(args) != 1 || v.localPlayer == nil || v.localPlayer.Stats == nil {
		return fmt.Errorf("usage: setmana <n> (in a game)")
	}

	n, err := strconv.Atoi(args[0])
	if err != nil || n < 0 {
		return fmt.Errorf("setmana needs a non-negative number")
	}

	s := v.localPlayer.Stats
	if n > s.MaxMana {
		n = s.MaxMana
	}

	s.Mana = n
	v.Infof("VITALS mana set to %d/%d", s.Mana, s.MaxMana)

	return nil
}
