package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func TestAttachElements(t *testing.T) {
	r := &d2records.MonStatRecord{}
	r.ElementAttackMode1, r.ElementType1 = "A1", "fire"
	r.ElementChance1Hell, r.ElementDamageMin1Hell, r.ElementDamageMax1Hell = 100, 50, 100
	r.ElementAttackMode2, r.ElementType2 = "A2", "ltng"
	r.ElementChance2Hell, r.ElementDamageMin2Hell, r.ElementDamageMax2Hell = 60, 25, 50
	r.ElementAttackMode3, r.ElementType3 = "S1", "stun"
	r.ElementChance3Hell = 100

	var v d2mapentity.MonsterVitals

	attachElements(&v, r, 2, 200)

	if v.A1.ElemType != "fire" || v.A1.ElemMin != 100 || v.A1.ElemMax != 200 || v.A1.ElemPct != 0 {
		t.Errorf("A1 %+v", v.A1)
	}

	if v.A2.ElemType != "ltng" || v.A2.ElemMin != 50 || v.A2.ElemMax != 100 || v.A2.ElemPct != 60 {
		t.Errorf("A2 %+v", v.A2)
	}

	if v.S1.ElemType != "" {
		t.Errorf("stun is not damage: %+v", v.S1)
	}

	// the Normal and Nightmare columns are empty: nothing attaches
	var w d2mapentity.MonsterVitals

	attachElements(&w, r, 0, 200)

	if w.A1.ElemType != "" || w.A2.ElemType != "" {
		t.Errorf("zero chance attached: %+v %+v", w.A1, w.A2)
	}

	// noRatio takes the columns raw
	r.IgnoreMonLevelTxt = true

	var x d2mapentity.MonsterVitals

	attachElements(&x, r, 2, 200)

	if x.A1.ElemMin != 50 || x.A1.ElemMax != 100 {
		t.Errorf("noRatio %+v", x.A1)
	}
}
