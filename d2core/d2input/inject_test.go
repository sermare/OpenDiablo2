package d2input

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
)

// recorder is a handler that logs the events it gets and can stop the propagation.
type recorder struct {
	name   string
	log    *[]string
	handle bool
}

func (r *recorder) note(s string) bool { *r.log = append(*r.log, r.name+":"+s); return r.handle }

func (r *recorder) OnMouseButtonDown(_ d2interface.MouseEvent) bool   { return r.note("down") }
func (r *recorder) OnMouseButtonUp(_ d2interface.MouseEvent) bool     { return r.note("up") }
func (r *recorder) OnMouseButtonRepeat(_ d2interface.MouseEvent) bool { return r.note("repeat") }
func (r *recorder) OnMouseMove(_ d2interface.MouseMoveEvent) bool     { return r.note("move") }
func (r *recorder) OnKeyDown(_ d2interface.KeyEvent) bool             { return r.note("keydown") }
func (r *recorder) OnKeyUp(_ d2interface.KeyEvent) bool               { return r.note("keyup") }

// mouseOnly keeps the last button down event it got.
type mouseOnly struct{ got *d2interface.MouseEvent }

func (m *mouseOnly) OnMouseButtonDown(e d2interface.MouseEvent) bool { m.got = &e; return false }

func TestInjectDispatchesLikePolledEvents(t *testing.T) {
	var log []string

	im := &inputManager{}
	hi := &recorder{name: "hi", log: &log, handle: true}
	lo := &recorder{name: "lo", log: &log}

	if err := im.BindHandlerWithPriority(lo, d2enum.PriorityDefault); err != nil {
		t.Fatal(err)
	}

	if err := im.BindHandlerWithPriority(hi, d2enum.PriorityHigh); err != nil {
		t.Fatal(err)
	}

	im.InjectMouseMove(10, 20, 0)
	im.InjectMouseButton(true, d2enum.MouseButtonLeft, 0, 10, 20)
	im.InjectMouseButton(false, d2enum.MouseButtonLeft, 0, 10, 20)
	im.InjectMouseRepeat(d2enum.MouseButtonLeft, 0, 10, 20)
	im.InjectKey(true, d2enum.KeyI, 0)
	im.InjectKey(false, d2enum.KeyI, 0)

	// the high priority handler consumed every event, so the default one saw none (as with polled events)
	want := []string{"hi:move", "hi:down", "hi:up", "hi:repeat", "hi:keydown", "hi:keyup"}
	if len(log) != len(want) {
		t.Fatalf("log %v, want %v", log, want)
	}

	for i := range want {
		if log[i] != want[i] {
			t.Fatalf("log %v, want %v", log, want)
		}
	}

	log = nil
	hi.handle = false
	im.InjectMouseButton(true, d2enum.MouseButtonLeft, 0, 1, 2)

	if len(log) != 2 || log[0] != "hi:down" || log[1] != "lo:down" {
		t.Fatalf("unhandled events must reach the lower priority: %v", log)
	}
}

func TestInjectedMouseEventFields(t *testing.T) {
	im := &inputManager{}
	m := &mouseOnly{}

	if err := im.BindHandler(m); err != nil {
		t.Fatal(err)
	}

	im.InjectKey(true, d2enum.KeyI, 0) // not a key handler: ignored
	im.InjectMouseButton(true, d2enum.MouseButtonRight, d2enum.KeyModShift, 33, 44)

	if m.got == nil {
		t.Fatal("no event")
	}

	e := *m.got
	if e.Button() != d2enum.MouseButtonRight || e.KeyMod() != d2enum.KeyModShift || e.X() != 33 || e.Y() != 44 {
		t.Fatalf("event %v %v %d,%d", e.Button(), e.KeyMod(), e.X(), e.Y())
	}
}

func TestInjectDoesNotMoveThePolledCursor(t *testing.T) {
	im := &inputManager{cursorX: 5, cursorY: 6}
	im.InjectMouseMove(100, 200, 0)

	if im.cursorX != 5 || im.cursorY != 6 {
		t.Fatalf("polled cursor moved to %d,%d", im.cursorX, im.cursorY)
	}
}
