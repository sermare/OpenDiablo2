package d2mapgen

import (
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
)

type fakeProvider struct {
	name  string
	ids   map[int]bool
	panic bool
}

func (f fakeProvider) Name() string        { return f.name }
func (f fakeProvider) CanLoad(id int) bool { return f.ids[id] }
func (f fakeProvider) Load(*MapGenerator, int, LoadRequest) error {
	if f.panic {
		panic("bad data")
	}

	return nil
}

func TestProviderOrder(t *testing.T) {
	g := &MapGenerator{Logger: d2util.NewLogger(), providers: []LevelProvider{fakeProvider{name: "town", ids: map[int]bool{1: true}}}}

	if !g.CanLoadLevel(1) || g.CanLoadLevel(35) {
		t.Fatal("only the town should load")
	}

	g.RegisterProvider(fakeProvider{name: "maze", ids: map[int]bool{35: true, 1: true}})

	if p := g.ProviderFor(1); p == nil || p.Name() != "maze" {
		t.Error("a provider registered later must win")
	}

	if p := g.ProviderFor(35); p == nil || p.Name() != "maze" {
		t.Error("maze must load 35")
	}

	if g.ProviderFor(99) != nil {
		t.Error("nothing loads level 99")
	}
}

func TestLoadLevelErrors(t *testing.T) {
	g := &MapGenerator{Logger: d2util.NewLogger()}

	if _, err := g.LoadLevel(5, LoadRequest{}); err == nil {
		t.Error("no provider must be an error")
	}

	g.RegisterProvider(fakeProvider{name: "bad", ids: map[int]bool{5: true}, panic: true})

	_, err := g.LoadLevel(5, LoadRequest{})
	if err == nil || !strings.Contains(err.Error(), "panic") {
		t.Errorf("a panicking provider must become an error, got %v", err)
	}
}
