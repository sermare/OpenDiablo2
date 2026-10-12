package d2gamescreen

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2groundexp"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// Ground item expiry (d2groundexp, VERIFIED 0x556a80 / 0x556c00 / 0x5407e0 / 0x52ad80). Items and gold piles
// on the floor disappear after a lifetime that depends on their kind; the purge only runs when the game frame
// is a multiple of 1500, for the live level and for the saved entries of the levels left behind.
//
// UNVERIFIED engine approximation: the exe stamps an item the frame it lands; here the stamp is taken by a scan
// that runs every 25 frames, so the stamp can be up to one second late. OD2_NOPERSIST does not affect expiry.

const groundScanFrames = 25

// groundExpiry is the stamp book of the items seen on the ground, by entity.
type groundExpiry struct {
	stamp     map[*d2mapentity.Item]int
	lastScan  int
	lastPurge int // game frame / 1500 of the last purge
}

func (v *Game) groundKind(it *d2mapentity.Item) d2groundexp.Kind {
	if it.IsGold() {
		return d2groundexp.Kind{Gold: true, GoldAmount: it.Gold}
	}

	if it.Item == nil {
		return d2groundexp.Kind{}
	}

	k := d2groundexp.Kind{Quest: it.Item.IsQuestItem(), Quality: int(it.Item.Quality())}

	if t := v.itemFactory().DescriptionTables(); t != nil {
		k.SocketFiller = t.IsA(it.Item.TypeCode, "sock")
	}

	return k
}

// stampGround gives every ground item that has no stamp yet its expiry frame.
func (v *Game) stampGround(frame int) {
	g := &v.groundExp
	if g.stamp == nil {
		g.stamp = map[*d2mapentity.Item]int{}
	}

	for _, e := range v.gameClient.MapEngine.Entities() {
		if it, ok := e.(*d2mapentity.Item); ok {
			if _, seen := g.stamp[it]; !seen {
				g.stamp[it] = d2groundexp.ExpiryFrame(frame, v.groundKind(it))
			}
		}
	}
}

// advanceGroundExpiry runs the scan and the periodic purge. Called once per game tick.
func (v *Game) advanceGroundExpiry() {
	if v.gameClient == nil || v.gameClient.MapEngine == nil || !v.isRealLevel() {
		return
	}

	frame := v.gameFrame()
	g := &v.groundExp

	if frame-g.lastScan >= groundScanFrames {
		g.lastScan = frame
		v.stampGround(frame)
	}

	bucket := frame / d2groundexp.PurgePeriodFrames
	if bucket == g.lastPurge {
		return
	}

	// the exe runs the purge on the frame that is a multiple of 1500; a tick may skip that exact frame
	g.lastPurge = bucket
	v.stampGround(frame)

	engine := v.gameClient.MapEngine
	removed := 0

	for _, e := range engine.Entities() {
		it, ok := e.(*d2mapentity.Item)
		if !ok {
			continue
		}

		if d2groundexp.LiveExpired(g.stamp[it], frame) {
			v.expireGround(it)
			engine.RemoveEntity(it)
			delete(g.stamp, it)

			removed++
		}
	}

	saved := 0

	for _, sl := range v.levelStore.levels {
		keep := sl.items[:0]

		for _, it := range sl.items {
			if d2groundexp.SavedExpired(g.stamp[it], frame) {
				delete(g.stamp, it)

				saved++

				continue
			}

			keep = append(keep, it)
		}

		sl.items = keep
	}

	if removed+saved > 0 {
		v.Infof("GROUND expiry at frame %d: %d items gone from the level, %d from saved levels", frame, removed, saved)
	}
}

// groundExpired says whether a saved ground item is already past its stamp (restore of a level).
func (v *Game) groundExpired(it *d2mapentity.Item) bool {
	return d2groundexp.SavedExpired(v.groundExp.stamp[it], v.gameFrame())
}
