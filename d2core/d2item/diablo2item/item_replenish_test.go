package diablo2item

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2stats"
)

// A thrown weapon the hero loaded from a save carries its unique's "replenishes quantity" only in the legacy
// property pools (no rolled data): StatTotal must find stat 253 there.
func TestStatTotalFindsLegacyPoolStat(t *testing.T) {
	stats := realStatRows(t)
	name := statNameByID(stats, ReplenishQuantityID)

	if name != "item_replenish_quantity" {
		t.Fatalf("stat %d is %q", ReplenishQuantityID, name)
	}

	rec := &d2records.ItemCommonRecord{Code: "9ja", Stackable: true, MaxStack: 60}
	it, sf := fakeStackItem(stats, rec)

	if got := it.StatTotal(ReplenishQuantityID); got != 0 {
		t.Fatalf("no properties: %d, want 0", got)
	}

	it.properties = map[PropertyPool][]*Property{
		PropertyPoolUnique: {{stats: []d2stats.Stat{sf.NewStat(name, 30)}}},
	}

	if got := it.StatTotal(ReplenishQuantityID); got != 30 {
		t.Errorf("legacy pool: %d, want 30", got)
	}
}
