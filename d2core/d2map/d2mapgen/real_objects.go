package d2mapgen

import (
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monreg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2object"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2objspawn"
)

// The random room objects of a real-map level (chests, barrels, urns, wells,
// shrines, racks ...): d2object.Scatter, the port of OBJRGN_SpawnRoomObjects,
// run for every room right before its natural monsters, on the room seed and
// the game seed the population already uses. OD2_ROOM_OBJECTS=0 switches it off.
//
// Engine stand-ins: the collision grid is the engine's walk-blocked sub-tiles
// plus the footprints of the objects made so far (the exe also sees the DS1
// preset objects, which are not part of this overlay); the room link test of
// ShouldSpawn is not modelled (every room of a non-town level is eligible).

func roomObjectsEnabled() bool { return os.Getenv("OD2_ROOM_OBJECTS") != "0" }

// objectField is the Collider over the engine: a box is blocked when any of its sub-tiles is
// walk-blocked or carries an object of the scatter.
type objectField struct {
	g        *MapGenerator
	occupied map[[2]int]bool
}

func (f *objectField) Blocked(x, y, w, h, mask int) bool {
	x0, y0 := x-w/2, y-h/2

	for sy := y0; sy < y0+h; sy++ {
		for sx := x0; sx < x0+w; sx++ {
			if f.g.engine.WalkBlocked(sx, sy) || (mask == d2object.MaskPlace && f.occupied[[2]int{sx, sy}]) {
				return true
			}
		}
	}

	return false
}

func (f *objectField) mark(p d2object.Placed, t d2objspawn.Tables) {
	d, _ := t.Def(p.ID)
	w, h := d.SizeX, d.SizeY

	if w < 1 {
		w = 1
	}

	if h < 1 {
		h = 1
	}

	for sy := p.Y - h/2; sy < p.Y-h/2+h; sy++ {
		for sx := p.X - w/2; sx < p.X-w/2+w; sx++ {
			f.occupied[[2]int{sx, sy}] = true
		}
	}
}

// roomObjects is the per-level state of the scatter.
type roomObjects struct {
	sc     *d2object.Scatter
	tables d2objspawn.Tables
	lv     d2object.LevelRow
	field  *objectField
	// byID counts the objects made per objects.txt row (for the log)
	byID map[int]int
	plan []d2mapengine.PlannedObject
}

// newRoomObjects prepares the scatter of a level; nil when it is off, the level is a town or has no groups.
func (p *popLevel) newRoomObjects(game *d2monreg.Game, total int) *roomObjects {
	g := p.g

	if !roomObjectsEnabled() || d2level.IsTown(p.level) || g.asset == nil || g.asset.Records == nil {
		return nil
	}

	rec := p.env.tb.Level(p.level)
	if rec == nil {
		return nil
	}

	lv := d2object.LevelRow{ID: p.level, Themes: rec.Themes}
	any := false

	for i := 0; i < 8; i++ {
		lv.Groups[i], lv.Probs[i] = rec.ObjGrp[i], rec.ObjPrb[i]
		any = any || lv.Groups[i] != 0
	}

	if !any && lv.Themes == 0 {
		return nil
	}

	ro := &roomObjects{
		tables: d2objspawn.FromRecords(g.asset.Records.Object.Details, g.asset.Records.Object.Groups),
		lv:     lv, byID: map[int]int{},
		field: &objectField{g: g, occupied: map[[2]int]bool{}},
	}

	ro.sc = &d2object.Scatter{
		Defs: ro.tables.Defs, Groups: ro.tables.Groups, Game: &game.Seed, Region: &d2object.Region{Total: total}, Col: ro.field,
		Create: func(pl d2object.Placed) {
			ro.field.mark(pl, ro.tables)
			ro.byID[pl.ID]++
			ro.plan = append(ro.plan, d2mapengine.PlannedObject{ID: pl.ID, X: pl.X, Y: pl.Y})
		},
	}

	return ro
}

// room runs the scatter for one room (subtile rectangle and its seed, shared with the monsters).
func (ro *roomObjects) room(p *popLevel, r *popRoom, rc *d2monreg.Room) {
	if ro == nil {
		return
	}

	ctx := &d2object.RoomCtx{
		Rect: d2object.Rect{X: rc.X, Y: rc.Y, W: rc.W, H: rc.H}, Seed: &rc.Seed,
	}

	if _, err := ro.sc.Room(ro.lv, ctx); err != nil {
		p.g.Warningf("room objects level %d: %v", p.level, err)
	}
}

// finish hands the plan to the engine and logs the summary.
func (ro *roomObjects) finish(p *popLevel) {
	if ro == nil {
		return
	}

	p.g.engine.SetRoomObjects(ro.plan)

	ids := make([]int, 0, len(ro.byID))
	for id := range ro.byID {
		ids = append(ids, id)
	}

	sort.Ints(ids)

	var sb strings.Builder

	for _, id := range ids {
		d, _ := ro.tables.Def(id)
		sb.WriteString(" " + strings.ReplaceAll(d.Name, " ", "_") + "x" + strconv.Itoa(ro.byID[id]))
	}

	p.g.Infof("ROOMOBJ level %d: %d objects in %d rooms (rooms seen %d)%s unmodelled=%v", p.level, len(ro.plan), len(p.rooms),
		ro.sc.Region.Seen, sb.String(), ro.sc.Unmodelled)
}
