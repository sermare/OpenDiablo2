package d2player

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2automap"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2config"
)

func testAutomap(dir string) *Automap {
	a := &Automap{store: d2automap.NewStore(), override: map[string]bool{}, Logger: d2util.NewLogger()}
	a.SetPersistence(d2automap.Sidecar{Dir: dir, Name: "Hero", GameID: 5})

	return a
}

func TestAutomapSaveReload(t *testing.T) {
	dir := t.TempDir()

	a := testAutomap(dir)
	a.store.Level(1).Add(d2automap.LayerFloor, 8, 4, 50)
	a.store.Level(1).Add(d2automap.LayerWall, 9, 4, 51)
	a.store.Level(29).Add(d2automap.LayerObject, 1, 2, 309)
	a.store.Level(3) // visited, nothing revealed: not written
	a.Flush()

	b := testAutomap(dir) // a new session
	b.loadLevel(1)
	b.loadLevel(29)
	b.loadLevel(3)

	if got := b.store.Level(1).Count(); got != 2 {
		t.Errorf("level 1 cells = %d, want 2", got)
	}

	if got := b.store.Level(29).CountLayer(d2automap.LayerObject); got != 1 {
		t.Errorf("level 29 objects = %d, want 1", got)
	}

	if b.saved[3] != 0 || b.store.Level(3).Count() != 0 {
		t.Error("level 3 should be empty")
	}

	// nothing changed: a second flush writes nothing (the saved counts match)
	if len(b.saved) != 2 {
		t.Errorf("saved = %v", b.saved)
	}
}

func TestAutomapNoSidecar(t *testing.T) {
	a := &Automap{store: d2automap.NewStore(), override: map[string]bool{}, Logger: d2util.NewLogger()}
	a.loadLevel(1)
	a.Flush() // must not panic or write anywhere
}

type fakeOptions struct{ cfg d2config.Configuration }

func (f *fakeOptions) Config() *d2config.Configuration { return &f.cfg }

func (f *fakeOptions) Change(key string, i int) error { return f.cfg.SetOption(key, i) }

func TestAutomapToggleOptionPersists(t *testing.T) {
	old := optionsBackend
	defer func() { optionsBackend = old }()

	fo := &fakeOptions{}
	SetOptionsBackend(fo)

	a := testAutomap(t.TempDir())

	for _, key := range []string{"fade", "center", "party", "names"} {
		if a.option(key) {
			t.Fatalf("%s starts off", key)
		}

		if !a.ToggleOption(key) || !a.option(key) {
			t.Errorf("%s did not turn on", key)
		}

		if got := fo.cfg.OptionValue(automapOptionKeys[key]); got != "YES" {
			t.Errorf("%s stored as %q", key, got)
		}

		if a.ToggleOption(key) || a.option(key) {
			t.Errorf("%s did not turn off", key)
		}
	}
}
