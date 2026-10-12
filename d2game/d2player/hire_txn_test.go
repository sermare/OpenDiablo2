package d2player

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

func TestHireDecide(t *testing.T) {
	ok := HireRequest{Kind: TxnHire, GateOpen: true, OnOffer: true, CanSpawn: true, Cost: 100, Gold: 100}

	with := func(f func(r *HireRequest)) HireRequest {
		r := ok
		f(&r)

		return r
	}

	for _, tc := range []struct {
		name string
		req  HireRequest
		want int
	}{
		{"hired", ok, HireReplyOK},
		{"stash covers the rest", with(func(r *HireRequest) { r.Gold, r.Stash = 10, 90 }), HireReplyOK},
		{"short", with(func(r *HireRequest) { r.Gold, r.Stash = 10, 89 }), HireReplyNoGold},
		{"gate", with(func(r *HireRequest) { r.GateOpen = false }), HireReplyGateClosed},
		{"gate beats gold", with(func(r *HireRequest) { r.GateOpen, r.Gold = false, 0 }), HireReplyGateClosed},
		{"not on offer", with(func(r *HireRequest) { r.OnOffer = false }), HireReplyBadRequest},
		{"cannot come", with(func(r *HireRequest) { r.CanSpawn = false }), HireReplyCannotNow},
		{"gold beats spawn", with(func(r *HireRequest) { r.CanSpawn, r.Gold = false, 0 }), HireReplyNoGold},
		{"max without replace", with(func(r *HireRequest) { r.Mercs = 1 }), HireReplyMaxMercs},
		{"replace confirmed", with(func(r *HireRequest) { r.Mercs, r.Replacing = 1, true }), HireReplyOK},
		{"revive", HireRequest{Kind: TxnRevive, OnOffer: true, Cost: 50, Stash: 50}, HireReplyOK},
		{"revive nobody", HireRequest{Kind: TxnRevive, Cost: 50, Gold: 50}, HireReplyBadRequest},
		{"revive broke", HireRequest{Kind: TxnRevive, OnOffer: true, Cost: 50, Gold: 49}, HireReplyNoGold},
	} {
		if got := HireDecide(tc.req); got != tc.want {
			t.Errorf("%s: code %d, want %d", tc.name, got, tc.want)
		}
	}

	// every code the server sends has a message
	for _, c := range []int{HireReplyBadRequest, HireReplyGateClosed, HireReplyNoGold, HireReplyCannotNow, HireReplyMaxMercs} {
		if HireReplyKey(c) == "" {
			t.Errorf("code %d has no message", c)
		}
	}
}

func TestHirePay(t *testing.T) {
	for _, tc := range []struct {
		gold, stash, cost, g, s int
		ok                      bool
	}{
		{500, 300, 200, 300, 300, true},
		{100, 300, 250, 0, 150, true},
		{100, 100, 201, 100, 100, false},
	} {
		g, s, ok := HirePay(tc.gold, tc.stash, tc.cost)
		if g != tc.g || s != tc.s || ok != tc.ok {
			t.Errorf("%+v: got %d %d %v", tc, g, s, ok)
		}
	}
}

func TestHireTxnTimer(t *testing.T) {
	var x HireTxn

	if r, to := x.Tick(1); r || to {
		t.Fatal("idle txn replied")
	}

	x.Start(150, 3, HireRequest{})

	if r, _ := x.Tick(HireTxnLatency / 2); r {
		t.Fatal("replied too early")
	}

	if r, _ := x.Tick(HireTxnLatency); !r {
		t.Fatal("no reply after the latency")
	}

	x.Start(150, 3, HireRequest{})
	x.Hold = true

	if r, to := x.Tick(HireTxnTimeout - 1); r || to {
		t.Fatal("held txn ended early")
	}

	if _, to := x.Tick(1); !to {
		t.Fatal("no timeout after 60 s")
	}
}

func TestHireListWheel(t *testing.T) {
	h := &HireList{hover: -1, open: true, entries: make([]HireListEntry, 10)}

	if !h.OnWheel(d2enum.KeyMouseWheelDown) || h.First() != 1 {
		t.Fatalf("wheel down: first=%d", h.First())
	}

	for i := 0; i < 10; i++ {
		h.OnWheel(d2enum.KeyMouseWheelDown)
	}

	if h.First() != 10-HireListRows {
		t.Fatalf("clamped first=%d", h.First())
	}

	h.OnWheel(d2enum.KeyMouseWheelUp)

	if h.First() != 10-HireListRows-1 {
		t.Fatalf("wheel up: first=%d", h.First())
	}

	if h.OnWheel(d2enum.KeyEnter) {
		t.Fatal("Enter is not a wheel notch")
	}

	h.open = false

	if h.OnWheel(d2enum.KeyMouseWheelUp) {
		t.Fatal("a closed list swallowed the wheel")
	}
}
