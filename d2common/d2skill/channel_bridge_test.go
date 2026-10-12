package d2skill

import "testing"

// TestChannelBridge: a list that ended inside the render frame of the request lives on (the held button of the exe
// asks every game frame); one that ended before that frame, or is still alive, is left alone.
func TestChannelBridge(t *testing.T) {
	reg := loadRealRegistry(t)
	id := reg.ByName("Inferno").ID
	frame := 0
	p := &Pipeline{Skills: reg, Frame: func() int { return frame }}
	u := newUnit(reg, map[string]int{"Inferno": 5})
	tg := Target{X: 10, Y: 10}

	if p.ChannelBridge(u, 0) {
		t.Fatal("no channel yet: nothing to bridge")
	}

	p.Start(u, id, tg)
	p.channelDo(u) // until = frame + ChannelStartFrames (the first run's list)

	frame = 3
	if p.ChannelBridge(u, 0) {
		t.Fatal("the list is alive: nothing to bridge")
	}

	frame = ChannelStartFrames + 6 // one render frame covered 12 ticks: the list (ended at frame 20) ended inside it
	if !p.ChannelBridge(u, frame-12) {
		t.Fatal("the list ended inside the last render frame: bridge")
	}

	if st := p.Start(u, id, tg); !st.OK || st.ChannelFirst {
		t.Fatalf("bridged run must be a continuation: %+v", st)
	}

	frame += 40
	if p.ChannelBridge(u, frame-12) {
		t.Fatal("the list ended long before the last render frame: a real pause, start over")
	}
}
