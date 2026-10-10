package d2skills

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
)

func TestLootedCorpsesAndAreaReset(t *testing.T) {
	e := auraEngine(d2state.Defs{"whirlwind": {}})

	if e.isLooted("c1") {
		t.Fatal("a fresh corpse is looted")
	}

	e.markLooted("c1")

	if !e.isLooted("c1") || e.isLooted("c2") {
		t.Error("looted bookkeeping")
	}

	e.abc.whirling = map[string]bool{"hero": true}
	e.setOf("hero").Apply(0, d2state.Instance{Name: "whirlwind"})
	e.abcReset()

	if e.isLooted("c1") || len(e.abc.whirling) != 0 || e.HasState("hero", "whirlwind") {
		t.Error("the area change kept corpses, runs or the whirlwind state")
	}
}

func TestActNow(t *testing.T) {
	e := &Engine{}
	if e.actNow() != 1 {
		t.Error("no host: act 1")
	}

	e.opt.Act = func() int { return 4 }
	if e.actNow() != 4 {
		t.Error("host act ignored")
	}

	e.opt.Act = func() int { return 0 }
	if e.actNow() != 1 {
		t.Error("act 0 (unknown level) must fall back to 1")
	}
}

func TestSign(t *testing.T) {
	for in, want := range map[int]int{-9: -1, 0: 0, 4: 1} {
		if got := sign(in); got != want {
			t.Errorf("sign(%d) = %d", in, got)
		}
	}
}
