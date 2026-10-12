package d2uber

import (
	"errors"
	"testing"
)

func TestCheckKeepsInputsOnRefusal(t *testing.T) {
	m, e, _ := newEvent()
	level, diff := LevelHarrogath, 2
	e.Where = func() (int, int) { return level, diff }

	if err := e.Check(PortalPandemonium); err != nil {
		t.Fatalf("Harrogath on Hell: %v", err)
	}

	if err := e.Check("Cow Portal"); err != nil {
		t.Fatalf("other recipes are not the event's: %v", err)
	}

	diff = 1
	if e.Check(PortalPandemonium) == nil || e.Check(PortalFinale) == nil {
		t.Fatal("Nightmare must refuse")
	}

	diff = 2

	e.OpenFinalePortal(m)

	if e.Check(PortalFinale) == nil {
		t.Fatal("the Tristram portal opens once per game")
	}

	e.ExeStub = true
	if !errors.Is(e.Check(PortalPandemonium), ErrExeStub) {
		t.Fatal("exe stub mode must fail the key recipe")
	}
}
