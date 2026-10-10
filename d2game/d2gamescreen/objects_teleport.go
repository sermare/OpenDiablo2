package d2gamescreen

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2object"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// padLandingRadius is how many subtiles around the partner pad the hero may land (the original asks for
// a free cell within 3 tiles of the partner).
const padLandingRadius = 15

// Level ids of the Act 2 teleport objects.
const (
	levelPalaceCellar3 = 54
	levelDurielLair    = 73
	levelArcaneSanct   = 74
)

// operateTeleport runs the teleport objects of Act 2: the Arcane Sanctuary pads (OperateFn 27, D2MOO
// OBJECTS_OperateFunction27_TeleportPad), the portal between Palace Cellar 3 and the sanctuary (34) and the
// portal to Duriel's lair (43).
func (v *Game) operateTeleport(ob *d2mapentity.Object, info d2object.Info) {
	rec := ob.Record()

	switch info.Fn {
	case d2object.FnTeleportPad:
		v.operateTeleportPad(ob)
	case d2object.FnArcanePortal:
		dest := levelArcaneSanct
		if v.currentLevel() == levelArcaneSanct {
			dest = levelPalaceCellar3
		}

		v.Infof("OBJECT arcane portal id=%d: %d -> %d", rec.Index, v.currentLevel(), dest)
		v.startLevelChange(dest, d2level.StartPortal, "portal")
	case d2object.FnDurielPortal:
		v.Infof("OBJECT Duriel's lair portal id=%d: %d -> %d", rec.Index, v.currentLevel(), levelDurielLair)
		v.startLevelChange(levelDurielLair, d2level.StartPortal, "portal")
	default:
		v.Infof("OBJECT stub %q (id %d) fn=%d class=%s: %s", ob.Label(), rec.Index, rec.OperateFn, info.Class, info.Name)
	}
}

// operateTeleportPad moves the hero next to the nearest other pad of the same kind.
func (v *Game) operateTeleportPad(ob *d2mapentity.Object) {
	rec := ob.Record()
	sx, sy := ob.GetPositionF()

	var pads []d2object.Pos

	for _, e := range v.gameClient.MapEngine.Entities() {
		if o, ok := e.(*d2mapentity.Object); ok && o.Record().Index == rec.Index {
			x, y := o.GetPositionF()
			pads = append(pads, d2object.Pos{X: x, Y: y})
		}
	}

	to, ok := d2object.PadPartner(d2object.Pos{X: sx, Y: sy}, pads)
	if !ok {
		v.Infof("OBJECT teleport pad id=%d at (%.1f,%.1f): no partner pad within %.0f tiles (%d pads of this kind)", rec.Index, sx, sy,
			d2object.PadSearchRadius, len(pads))

		return
	}

	lx, ly, found := v.gameClient.MapEngine.NearestWalkable(int(to.X*subtilesInTile), int(to.Y*subtilesInTile), padLandingRadius)
	if !found {
		v.Warningf("OBJECT teleport pad id=%d: no free cell next to the partner pad at (%.1f,%.1f)", rec.Index, to.X, to.Y)

		return
	}

	v.localPlayer.SetPositionSubtile(lx, ly)
	v.Infof("OBJECT teleport pad id=%d (%.1f,%.1f) -> partner (%.1f,%.1f), hero lands at subtile (%d,%d)", rec.Index, sx, sy, to.X, to.Y, lx, ly)
}
