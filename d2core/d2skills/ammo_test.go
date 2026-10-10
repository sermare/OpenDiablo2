package d2skills

import "testing"

// TestHeroAmmoFollowsTheStack proves the hero's ranged skills find ammunition only while the worn stack
// has some, and that using it empties the stack (Options.AmmoLeft / UseAmmo).
func TestHeroAmmoFollowsTheStack(t *testing.T) {
	stack := 2

	e := &Engine{opt: Options{
		AmmoLeft: func() int { return stack },
		UseAmmo: func() bool {
			if stack < 1 {
				return false
			}

			stack--

			return true
		},
	}}
	h := &heroUnit{e: e}

	if !h.HasAmmo() {
		t.Fatal("no ammunition with a stack of 2")
	}

	for i := 0; i < 2; i++ {
		if !h.ConsumeAmmo() {
			t.Fatalf("throw %d refused", i+1)
		}
	}

	if h.HasAmmo() || h.ConsumeAmmo() {
		t.Error("ammunition left after the stack ran out")
	}

	// without the hooks and without the scenario switch there is none (the old behaviour)
	if bare := (&heroUnit{e: &Engine{}}); bare.HasAmmo() || bare.ConsumeAmmo() {
		t.Error("ammunition without a source")
	}

	// the scenarios shoot without limit
	if inf := (&heroUnit{e: &Engine{opt: Options{InfiniteAmmo: true}}}); !inf.HasAmmo() || !inf.ConsumeAmmo() {
		t.Error("no ammunition in a scenario with infinite ammunition")
	}

	// a hireling shoots without limit
	if merc := (&heroUnit{e: &Engine{}, merc: &mercCtx{}}); !merc.HasAmmo() || !merc.ConsumeAmmo() {
		t.Error("a hireling is out of ammunition")
	}
}
