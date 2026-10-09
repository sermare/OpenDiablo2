package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func arMonster(mod func(*d2records.MonStatRecord)) *d2mapentity.Monster {
	r := &d2records.MonStatRecord{}
	if mod != nil {
		mod(r)
	}

	return &d2mapentity.Monster{Stat: r}
}

func TestHeroAROperands(t *testing.T) {
	list := func(id int, v int64) *d2statlist.Totals {
		l := d2statlist.NewList()
		l.Add(id, 0, v)

		return &d2statlist.Totals{Stats: l}
	}
	undead := arMonster(func(r *d2records.MonStatRecord) { r.IsUndeadHigh = true })
	demon := arMonster(func(r *d2records.MonStatRecord) { r.IsDemon = true })
	boss := arMonster(func(r *d2records.MonStatRecord) { r.IsSpecialBoss = true })
	plain := arMonster(nil)

	tests := []struct {
		name            string
		t               *d2statlist.Totals
		m               *d2mapentity.Monster
		wantAR, wantDef int
	}{
		{"no totals", nil, undead, 100, 200},
		{"empty list", &d2statlist.Totals{Stats: d2statlist.NewList()}, undead, 100, 200},
		{"undead bonus on undead", list(d2statlist.StatUndeadAR, 50), undead, 150, 200},
		{"undead bonus on plain", list(d2statlist.StatUndeadAR, 50), plain, 100, 200},
		{"demon bonus on demon", list(d2statlist.StatDemonAR, 30), demon, 130, 200},
		{"demon bonus on undead", list(d2statlist.StatDemonAR, 30), undead, 100, 200},
		{"ignore defense plain", list(d2statlist.StatIgnoreDef, 1), plain, 100, 0},
		{"ignore defense boss", list(d2statlist.StatIgnoreDef, 1), boss, 100, 200},
		{"target ac 40 plain", list(d2statlist.StatTargetACPct, 40), plain, 100, 120},
		{"target ac 40 boss halved", list(d2statlist.StatTargetACPct, 40), boss, 100, 160},
	}

	for _, tt := range tests {
		ar, def := HeroAROperands(tt.t, tt.m, 100, 200)
		if ar != tt.wantAR || def != tt.wantDef {
			t.Errorf("%s: got %d,%d want %d,%d", tt.name, ar, def, tt.wantAR, tt.wantDef)
		}
	}
}

// A hero without any of the stats must roll exactly as before the wiring:
// same AR and defense in, same hits and same seed state out.
func TestHeroAROperandsRegressionFixedSeed(t *testing.T) {
	l := d2statlist.NewList()
	l.Add(d2statlist.StatToHit, 0, 120) // unrelated stat

	a, b := d2rand.New(0xC0FFEE), d2rand.New(0xC0FFEE)
	tot := &d2statlist.Totals{Stats: l}
	m := arMonster(func(r *d2records.MonStatRecord) { r.IsUndeadLow = true })
	lv := d2combat.ToHitInput{AttackerLevel: 20, DefenderLevel: 15}

	for i := 0; i < 500; i++ {
		ar, def := 80+i%300, 40+(i*7)%400

		in1 := lv
		in1.AttackRating, in1.Defense = ar, def
		h1, c1, r1 := d2combat.RollToHit(a, in1)

		ar2, def2 := HeroAROperands(tot, m, ar, def)
		in2 := lv
		in2.AttackRating, in2.Defense = ar2, def2
		h2, c2, r2 := d2combat.RollToHit(b, in2)

		if h1 != h2 || c1 != c2 || r1 != r2 {
			t.Fatalf("roll %d differs: %v/%d/%d vs %v/%d/%d", i, h1, c1, r1, h2, c2, r2)
		}
	}

	if a.Step() != b.Step() {
		t.Fatal("seed state diverged")
	}
}
