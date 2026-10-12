package diablo2item

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2trade"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// Refill raises every charged skill to its maximum (ITEM_UpdateSkillChargesToMax 0x55dc40).
func TestRefillRestoresCharges(t *testing.T) {
	f := priceFactory()
	it := &Item{factory: f, CommonCode: "cap", quality: d2drop.QualityMagic,
		rolled: &d2drop.Rolled{Writes: []d2drop.StatWrite{{Kind: 'L', Stat: d2trade.StatChargedSkill, Param: 3, Value: 2 | 10<<8}}}}

	if !d2trade.HasMissingCharges(entriesOf(it.priceStats(), d2trade.StatChargedSkill)) {
		t.Fatal("should start with missing charges")
	}

	it.chargesFull = true

	got := entriesOf(it.priceStats(), d2trade.StatChargedSkill)
	if len(got) != 1 || got[0].Value != 10|10<<8 || d2trade.HasMissingCharges(got) {
		t.Fatalf("got %+v", got)
	}
}
