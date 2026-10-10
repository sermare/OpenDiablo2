package d2reward

import "testing"

// Table tests of the item reward rules read in Game.exe 1.14b (handler
// 0x577b70 classes 0x9a, 0x1ff, 0x200).

func TestLarzukRange(t *testing.T) {
	// a 2x3 body armour (6 cells) with the real Armor type brackets 3/4/6.
	armor := SocketItem{Code: "plt", ItemLevel: 30, Quality: 2, BaseSockets: 4, MaxSock1: 3, MaxSock25: 4, MaxSock40: 6, Width: 2, Height: 3}

	tests := []struct {
		name   string
		mod    func(i *SocketItem)
		lo, hi int
		err    bool
	}{
		{"normal ilvl 26-40 bracket", func(i *SocketItem) {}, 4, 4, false},
		{"ilvl 25 uses MaxSock1", func(i *SocketItem) { i.ItemLevel = 25 }, 3, 3, false},
		{"ilvl 26 uses MaxSock25", func(i *SocketItem) { i.ItemLevel = 26 }, 4, 4, false},
		{"ilvl 40 still MaxSock25", func(i *SocketItem) { i.ItemLevel = 40 }, 4, 4, false},
		{"ilvl 41 uses MaxSock40 but the base limits it", func(i *SocketItem) { i.ItemLevel = 41 }, 4, 4, false},
		{"base with 6 slots at ilvl 41", func(i *SocketItem) { i.ItemLevel = 41; i.BaseSockets = 6 }, 6, 6, false},
		{"superior like normal", func(i *SocketItem) { i.Quality = 3 }, 4, 4, false},
		{"low like normal", func(i *SocketItem) { i.Quality = 1 }, 4, 4, false},
		{"magic is 1..2", func(i *SocketItem) { i.Quality = 4 }, 1, 2, false},
		{"magic with max 1", func(i *SocketItem) { i.Quality = 4; i.ItemLevel = 25; i.MaxSock1 = 1 }, 1, 1, false},
		{"set gets 1", func(i *SocketItem) { i.Quality = 5 }, 1, 1, false},
		{"rare gets 1", func(i *SocketItem) { i.Quality = 6 }, 1, 1, false},
		{"unique gets 1", func(i *SocketItem) { i.Quality = 7 }, 1, 1, false},
		{"crafted gets 1", func(i *SocketItem) { i.Quality = 8 }, 1, 1, false},
		{"small item is limited by its cells", func(i *SocketItem) { i.Width, i.Height = 1, 2 }, 2, 2, false},
		{"1x1 item gets one", func(i *SocketItem) { i.Width, i.Height = 1, 1 }, 1, 1, false},
		{"cells are capped at 6", func(i *SocketItem) { i.Width, i.Height = 4, 4; i.BaseSockets = 9; i.MaxSock25 = 9 }, 6, 6, false},
		{"already socketed", func(i *SocketItem) { i.Sockets = 2 }, 0, 0, true},
		{"has gems", func(i *SocketItem) { i.Gems = 1 }, 0, 0, true},
		{"no socket slots", func(i *SocketItem) { i.BaseSockets = 0 }, 0, 0, true},
		{"type allows none", func(i *SocketItem) { i.MaxSock25 = 0 }, 0, 0, true},
		{"quest item", func(i *SocketItem) { i.Quest = true }, 0, 0, true},
		{"non-sellable item", func(i *SocketItem) { i.NonSellable = true }, 0, 0, true},
		{"gold", func(i *SocketItem) { i.Gold = true }, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := armor
			tt.mod(&it)

			lo, hi, err := LarzukRange(it)
			if (err != nil) != tt.err || lo != tt.lo || hi != tt.hi {
				t.Fatalf("got %d..%d, %v; want %d..%d err=%v", lo, hi, err, tt.lo, tt.hi, tt.err)
			}
		})
	}
}

func TestLarzukSocketsRoll(t *testing.T) {
	magic := SocketItem{ItemLevel: 30, Quality: 4, BaseSockets: 6, MaxSock25: 4, Width: 2, Height: 3}

	for want, roll := range map[int]func(int) int{
		1: func(int) int { return 0 },
		2: func(n int) int { return n - 1 },
	} {
		got, err := LarzukSockets(magic, roll)
		if err != nil || got != want {
			t.Errorf("roll -> %d, %v; want %d", got, err, want)
		}
	}

	if got, _ := LarzukSockets(magic, nil); got != 2 {
		t.Errorf("nil roll takes the maximum, got %d", got)
	}

	normal := magic
	normal.Quality = 2

	called := false

	if got, _ := LarzukSockets(normal, func(int) int { called = true; return 0 }); got != 4 || called {
		t.Errorf("normal items roll nothing: %d called=%v", got, called)
	}
}

func TestCanPersonalize(t *testing.T) {
	tests := []struct {
		name string
		it   PersonalizeItem
		err  bool
	}{
		{"nameable base, any quality", PersonalizeItem{Nameable: true}, false},
		{"not nameable (jewel, potion)", PersonalizeItem{}, true},
		{"already personalised", PersonalizeItem{Nameable: true, Personalized: true}, true},
		{"non-sellable", PersonalizeItem{Nameable: true, NonSellable: true}, true},
		{"gold", PersonalizeItem{Nameable: true, Gold: true}, true},
		{"quiver or body part", PersonalizeItem{Nameable: true, Excluded: true}, true},
	}

	for _, tt := range tests {
		if err := CanPersonalize(tt.it); (err != nil) != tt.err {
			t.Errorf("%s: %v", tt.name, err)
		}
	}
}

func TestCanImbue(t *testing.T) {
	ok := ImbueItem{WeaponOrArmor: true, Quality: 2}

	tests := []struct {
		name string
		mod  func(i *ImbueItem)
		err  bool
	}{
		{"normal", func(i *ImbueItem) {}, false},
		{"low quality", func(i *ImbueItem) { i.Quality = 1 }, false},
		{"superior", func(i *ImbueItem) { i.Quality = 3 }, false},
		{"magic is refused", func(i *ImbueItem) { i.Quality = 4 }, true},
		{"set", func(i *ImbueItem) { i.Quality = 5 }, true},
		{"rare", func(i *ImbueItem) { i.Quality = 6 }, true},
		{"unique", func(i *ImbueItem) { i.Quality = 7 }, true},
		{"crafted", func(i *ImbueItem) { i.Quality = 8 }, true},
		{"tempered", func(i *ImbueItem) { i.Quality = 9 }, true},
		{"jewelry", func(i *ImbueItem) { i.WeaponOrArmor = false }, true},
		{"quest item", func(i *ImbueItem) { i.Quest = true }, true},
		{"with gems", func(i *ImbueItem) { i.Gems = 1 }, true},
		{"socketed", func(i *ImbueItem) { i.Socketed = true }, true},
		{"throwing weapon", func(i *ImbueItem) { i.Throwable = true }, true},
		{"non-sellable", func(i *ImbueItem) { i.NonSellable = true }, true},
		{"gold", func(i *ImbueItem) { i.Gold = true }, true},
	}

	for _, tt := range tests {
		it := ok
		tt.mod(&it)

		if err := CanImbue(it); (err != nil) != tt.err {
			t.Errorf("%s: %v", tt.name, err)
		}
	}
}

func TestImbueLevel(t *testing.T) {
	// ITEMGEN_GetDropBaseLevel is the character level, at least 1; +4 above 5.
	for lvl, want := range map[int]int{-3: 1, 0: 1, 1: 1, 5: 5, 6: 10, 30: 34, 94: 98} {
		if got := ImbueLevel(lvl); got != want {
			t.Errorf("clvl %d: ilvl %d, want %d", lvl, got, want)
		}
	}
}
