package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

func TestWoundedAllyScan(t *testing.T) {
	d := &Director{units: map[uint32]*unit{}}
	mk := func(id uint32, x, pct int, allied bool) {
		d.units[id] = &unit{m: &d2mapentity.Monster{}, b: &d2monster.Brain{ID: id, X: x, Y: 100, Size: 1,
			HPPercent: pct, Allied: allied}}
	}

	me := &d2monster.Brain{ID: 1, X: 100, Y: 100}
	d.units[1] = &unit{m: &d2mapentity.Monster{}, b: me}
	mk(2, 110, 60, false) // in range, wounded
	mk(3, 120, 30, false) // in range, more wounded -> wins
	mk(4, 105, 80, false) // healthy
	mk(5, 105, 10, true)  // converted: other side
	mk(6, 160, 5, false)  // 60 subtiles away: 3600 > 2500

	q := d2monster.FBXScanQuery{Kind: d2monster.FBXScanWoundedAlly, Radius2: 2500, LifeBelow: 75}

	r := d.FBXScan(me, q)
	if !r.Found || r.T.ID != 3 {
		t.Fatalf("scan = %+v, want unit 3", r)
	}

	q.LifeBelow = 20
	if r := d.FBXScan(me, q); r.Found {
		t.Fatalf("nothing is below 20 percent in range: %+v", r)
	}
}
