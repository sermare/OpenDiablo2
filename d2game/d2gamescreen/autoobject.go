package d2gamescreen

import (
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2object"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// World object autotest, no clicking:
//
//	OD2_AUTOOBJECT=<id|name>[,count][;...]  spawns `count` (default 1) objects.txt objects
//	                                  next to the hero (by row id, or by Name: the first
//	                                  row with that name) and operates each one
//	OD2_AUTOOBJECT_SHRINE=<code|all>  forces the shrine type (shrines.txt Code) of
//	                                  shrine objects; "all" cycles through codes 1..22
//	OD2_AUTOOBJECT_LIFE=<n>           sets the hero's life to n first (to see wells and
//	                                  recharge shrines work)
//	OD2_AUTOOBJECT_NEAR=1             places each object 2 subtiles from the hero (explosions
//	                                  reach 3 subtiles)
//	OD2_AUTOOBJECT_MONSTER=<key>[,n]  spawns n monsters (default 2) next to the objects first
//	OD2_AUTOOBJECT_FORCE=lock,trap=<h> forces the container roll: locked, spawn handler h (1-9)
//	OD2_AUTOOBJECT_GIVE=<code>[,code] puts items (a "key", gems) in the inventory first
//	OD2_AUTOOBJECT_GIVE_AFTER=<code>  the same, right after the first object was operated
//
// After operating, the test jumps the effect clock forward so that buffs
// expire and used shrines/wells re-arm, and logs those as well. Every line has
// an "OBJECT" prefix ("OBJECT operate ...", "OBJECT effect expired ...",
// "OBJECT rearmed ...", "OBJECT autotest done ...").
const (
	autoObjectDelay  = 5.0
	autoObjectSettle = 1.5
)

type autoObjPhase int

const (
	aoWait autoObjPhase = iota
	aoSettle
	aoForward
	aoFinish
	aoDone
)

type autoObject struct {
	phase    autoObjPhase
	elapsed  float64
	timer    float64
	objects  []*d2mapentity.Object
	operated int
}

func autoObjectEnabled() bool { return os.Getenv("OD2_AUTOOBJECT") != "" }

func (v *Game) advanceAutoObject(elapsed float64) {
	if !autoObjectEnabled() || v.localPlayer == nil || v.gameControls == nil {
		return
	}

	a := &v.autoObject
	a.elapsed += elapsed
	a.timer += elapsed

	switch a.phase {
	case aoWait:
		if a.elapsed >= autoObjectDelay {
			v.autoObjectRun()
			a.phase, a.timer = aoSettle, 0
		}
	case aoSettle:
		if a.timer >= autoObjectSettle {
			v.autoObjectForward()
			a.phase, a.timer = aoForward, 0
		}
	case aoForward:
		if a.timer >= autoObjectSettle {
			a.phase = aoFinish
		}
	case aoFinish:
		v.Infof("OBJECT autotest done operated=%d active_effects=%d %s", a.operated,
			len(v.objects.overlay.Buffs.Active()), v.vitalsText())

		a.phase = aoDone
		v.autoTestExit()
	}
}

// autoObjectRows resolves the spec to an objects.txt row id.
func (v *Game) autoObjectRow(name string) int {
	if id, err := strconv.Atoi(name); err == nil {
		return id
	}

	best := -1

	for id, r := range v.asset.Records.Object.Details {
		if strings.EqualFold(r.Name, name) && (best < 0 || id < best) {
			best = id
		}
	}

	return best
}

func (v *Game) autoObjectRun() {
	type job struct {
		rec   *d2records.ObjectDetailRecord
		count int
	}

	var jobs []job

	total := 0

	for _, spec := range strings.Split(os.Getenv("OD2_AUTOOBJECT"), ";") {
		parts := strings.Split(spec, ",")
		id := v.autoObjectRow(strings.TrimSpace(parts[0]))
		count := 1

		if len(parts) > 1 {
			if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 {
				count = n
			}
		}

		rec := v.asset.Records.Object.Details[id]
		if rec == nil {
			v.Warningf("OBJECT autotest: no objects.txt row for %q", parts[0])
			continue
		}

		jobs = append(jobs, job{rec, count})
		total += count
	}

	if n, err := envInt("OD2_AUTOOBJECT_LIFE"); err == nil && n > 0 {
		v.localPlayer.Stats.Health = n
	}

	if n, err := envInt("OD2_AUTOOBJECT_MANA"); err == nil && n >= 0 {
		v.localPlayer.Stats.Mana = n
	}

	px, py := v.localPlayer.GetPositionF()
	cells := v.freeDropCells(int(math.Floor(px)), int(math.Floor(py)), total*2, true)
	next := 0

	v.autoObjectPrepare()

	v.Infof("OBJECT autotest start jobs=%d objects=%d hero=(%d,%d) level=%d %s", len(jobs), total, int(px), int(py),
		v.localPlayer.Stats.Level, v.vitalsText())

	for _, j := range jobs {
		rec := j.rec

		v.Infof("OBJECT autotest group id=%d name=%q fn=%d class=%s count=%d", rec.Index, rec.Name, rec.OperateFn,
			d2object.Lookup(rec.OperateFn).Class, j.count)

		for i := 0; i < j.count; i++ {
			if 2*next+1 >= len(cells) {
				v.Warningf("OBJECT autotest: no free cell for object %d", rec.Index)
				break
			}

			c := cells[2*next+1]
			next++

			sx, sy := c.X*subtilesInTile+2, c.Y*subtilesInTile+2
			if os.Getenv("OD2_AUTOOBJECT_NEAR") != "" {
				hx, hy := v.heroSubXY()
				sx, sy = hx+2, hy
			}

			ob, err := v.gameClient.MapEngine.NewObject(sx, sy, rec, d2resource.PaletteUnits)
			if err != nil {
				v.Warningf("OBJECT autotest: could not create %d: %v", rec.Index, err)
				continue
			}

			v.gameClient.MapEngine.AddEntity(ob)
			v.autoObject.objects = append(v.autoObject.objects, ob)
			v.autoObjectForce(ob)

			v.autoObjectOperate(ob, i)
			v.autoObject.operated++

			if v.autoObject.operated == 1 {
				v.autoObjectGive("OD2_AUTOOBJECT_GIVE_AFTER") // e.g. the key for the second, locked chest
			}
		}
	}
}

// autoObjectOperate operates one spawned object, forcing the shrine type when asked.
func (v *Game) autoObjectOperate(ob *d2mapentity.Object, index int) {
	spec := strings.TrimSpace(os.Getenv("OD2_AUTOOBJECT_SHRINE"))

	if spec != "" && d2object.Lookup(ob.Record().OperateFn).Class == d2object.ClassShrine {
		code := 0

		if spec == "all" {
			code = index%(len(d2object.DefaultShrines)-1) + 1
		} else if n, err := strconv.Atoi(spec); err == nil {
			code = n
		}

		if s, ok := d2object.ShrineByCode(v.shrineTable(), code); ok {
			v.useShrine(ob, v.objectInstance(ob), s)
			return
		}
	}

	v.operateWorldObject(ob)
}

// autoObjectForward moves the effect clock beyond the last expiry and re-arm time.
func (v *Game) autoObjectForward() {
	until := v.objects.clock

	for _, b := range v.objects.overlay.Buffs.Active() {
		until = math.Max(until, b.Expires)
	}

	for _, in := range v.objects.instances {
		until = math.Max(until, in.rearmAt)
	}

	if until > v.objects.clock {
		v.Infof("OBJECT autotest fast-forward effect clock %.1f -> %.1f", v.objects.clock, until+0.5)
		v.objects.clock = until + 0.5
	}
}

// autoObjectPrepare gives the items and spawns the monsters the scenario asked for (OD2_AUTOOBJECT_GIVE,
// OD2_AUTOOBJECT_MONSTER).
func (v *Game) autoObjectPrepare() {
	v.autoObjectGive("OD2_AUTOOBJECT_GIVE")

	spec := strings.TrimSpace(os.Getenv("OD2_AUTOOBJECT_MONSTER"))
	if spec == "" {
		return
	}

	parts := strings.Split(spec, ",")
	n := 2

	if len(parts) > 1 {
		if c, err := strconv.Atoi(parts[1]); err == nil && c > 0 {
			n = c
		}
	}

	md := v.monsterDirector()
	if md == nil {
		return
	}

	stat := md.FindStat(parts[0])
	if stat == nil {
		v.Warningf("OBJECT autotest: no monster %q", parts[0])
		return
	}

	hx, hy := v.heroSubXY()

	for i := 0; i < n; i++ {
		if m, err := md.SpawnNear(stat, hx+3, hy+i, 1); err != nil {
			v.Warningf("OBJECT autotest: monster %q: %v", parts[0], err)
		} else {
			mx, my := m.SubtilePos()
			v.Infof("OBJECT autotest monster %q hp=%d/%d level=%d at subtile (%d,%d) hero (%d,%d)", m.Label(), m.Vitals.HP,
				m.Vitals.MaxHP, m.Vitals.Level, mx, my, hx, hy)
		}
	}
}

// autoObjectGive puts the items named by an environment variable into the inventory.
func (v *Game) autoObjectGive(env string) {
	for _, code := range strings.Split(os.Getenv(env), ",") {
		if code = strings.TrimSpace(code); code == "" {
			continue
		}

		name, err := v.gameControls.GiveItem(code)
		v.Infof("OBJECT autotest gave %q -> %q err=%v", code, name, err)
	}
}

// autoObjectForce applies OD2_AUTOOBJECT_FORCE to a container: "lock" locks it, "trap=<h>" arms spawn handler h.
func (v *Game) autoObjectForce(ob *d2mapentity.Object) {
	spec := strings.TrimSpace(os.Getenv("OD2_AUTOOBJECT_FORCE"))
	if spec == "" || ob.Record().SubClass&d2object.SubChest == 0 {
		return
	}

	ci := d2object.ChestInit{}

	for _, f := range strings.Split(spec, ",") {
		switch {
		case f == "lock":
			ci.Locked = true
		case strings.HasPrefix(f, "trap="):
			ci.Handler, _ = strconv.Atoi(strings.TrimPrefix(f, "trap="))
		}
	}

	v.objectInstance(ob).chestInit = &ci
	v.Infof("OBJECT autotest forced id=%d locked=%v spawn_handler=%d", ob.Record().Index, ci.Locked, ci.Handler)
}
