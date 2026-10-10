package d2combat

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// TestOracleLeech compares ApplyLeech with the real 0x57a3b0.
func TestOracleLeech(t *testing.T) {
	var g struct {
		Cases []json.RawMessage `json:"cases"`
	}

	loadJSON(t, "leech_golden.json", &g)

	bad := 0

	for _, raw := range g.Cases {
		var atype, sp, dead, st5c, dtype, dmana, dstam, pct, diff, d1, d2, total, ll, ml, sl, outL, outM int
		var pre, post, ev []int
		var lo, hi, lo2, hi2 int64

		if err := json.Unmarshal(raw, &[]interface{}{&atype, &sp, &dead, &st5c, &dtype, &dmana, &dstam, &pct, &diff, &d1, &d2, &pre, &total, &ll, &ml, &sl,
			&lo, &hi, &outL, &outM, &post, &ev, &lo2, &hi2}); err != nil {
			t.Fatal(err)
		}

		seed := &d2rand.Seed{Lo: uint32(lo), Hi: uint32(hi)}
		o := ApplyLeech(seed, LeechIn{
			Total: int32(total), LifeLeech: int32(ll), ManaLeech: int32(ml), StamLeech: int32(sl),
			HasAttacker: atype >= 0, AttackerKind: atype, Special: sp == 1,
			Life: int32(pre[0]), MaxLife: int32(pre[1]), Mana: int32(pre[2]), MaxMana: int32(pre[3]),
			AttackerDead: dead == 1, AttackerNoHeal: st5c == 1,
			DefenderIsMonster: dtype == 1, DefenderDrain: int32(pct & 255), DefenderMana: int32(dmana), DefenderStamina: int32(dstam),
			DiffLifeDiv: int32(d1), DiffManaDiv: int32(d2),
		})

		gotEv := make([]int, len(o.Events))
		copy(gotEv, o.Events)

		if len(ev) == 0 {
			ev = []int{}
		}

		if len(gotEv) == 0 {
			gotEv = []int{}
		}

		wantLife, wantMana := post[0], post[1]
		if atype < 0 {
			wantLife, wantMana = 0, 0
		}

		gotLife, gotMana := int(o.Life), int(o.Mana)
		if atype < 0 {
			gotLife, gotMana = 0, 0
		}

		if int(o.LifeLeech) != outL || int(o.ManaLeech) != outM || gotLife != wantLife || gotMana != wantMana || !reflect.DeepEqual(gotEv, ev) ||
			int64(seed.Lo) != lo2 || int64(seed.Hi) != hi2 {
			bad++

			if bad <= 5 {
				t.Errorf("leech atype=%d sp=%d dtype=%d pct=%d total=%d l/m/s=%d/%d/%d pre=%v: got L=%d M=%d life=%d mana=%d ev=%v seed=%x:%x\n want L=%d M=%d life=%d mana=%d ev=%v seed=%x:%x",
					atype, sp, dtype, pct, total, ll, ml, sl, pre, o.LifeLeech, o.ManaLeech, gotLife, gotMana, gotEv, seed.Lo, seed.Hi, outL, outM, wantLife, wantMana, ev, lo2, hi2)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d of %d leech cases differ", bad, len(g.Cases))
	}
}
