package d2monstats

import "testing"

func TestPlayerBonus(t *testing.T) {
	for _, c := range []struct{ n, align, hp, xp, players int }{
		{0, 0, 0, 0, 1}, {1, 0, 0, 0, 1}, {2, 0, 50, 50, 2}, {3, 0, 100, 100, 3},
		{8, 0, 350, 350, 8}, {9, 0, 350, 350, 9}, {10, 0, 400, 360, 10},
		{5, 1, 0, 0, 1}, {5, 2, 0, 0, 1},
	} {
		hp, xp, p := PlayerBonus(c.n, c.align)
		if hp != c.hp || xp != c.xp || p != c.players {
			t.Errorf("PlayerBonus(%d,%d)=%d,%d,%d want %d,%d,%d", c.n, c.align, hp, xp, p, c.hp, c.xp, c.players)
		}
	}
}

func TestEffectivePlayers(t *testing.T) {
	if EffectivePlayers(1, 5, 2) != 5 || EffectivePlayers(1, 5, 0) != 1 || EffectivePlayers(6, 5, 1) != 6 {
		t.Error("EffectivePlayers")
	}
}

func TestPlayerBonusAppliesBeforeCap(t *testing.T) {
	ml := synthLvl()
	c := &Class{MinHP: [3]int{100, 0, 0}, MaxHP: [3]int{100, 0, 0}, Exp: [3]int{100, 0, 0}}
	base := ml.Scale(c, Normal, 0, false, nil)
	s := ml.ScaleOpts(c, Normal, 0, false, nil, Options{Players: 3})

	if s.HP != base.HP*2 || s.XP != base.XP*2 || s.Players != 3 {
		t.Errorf("3 players: %+v vs %+v", s, base)
	}

	big := &Class{MinHP: [3]int{1 << 29, 0, 0}, MaxHP: [3]int{1 << 29, 0, 0}}
	if s := ml.ScaleOpts(big, Normal, 0, false, nil, Options{Players: 8}); s.HP != maxHP {
		t.Errorf("cap after bonus: %d", s.HP)
	}

	friend := &Class{Align: 1, MinHP: [3]int{100, 0, 0}, MaxHP: [3]int{100, 0, 0}}
	if s := ml.ScaleOpts(friend, Normal, 0, false, nil, Options{Players: 8}); s.HP != ml.Scale(friend, Normal, 0, false, nil).HP {
		t.Error("friendly class must not scale")
	}
}

func TestClassicAdjustment(t *testing.T) {
	ml := synthLvl()
	c := &Class{
		MinHP: [3]int{100, 100, 100}, MaxHP: [3]int{100, 100, 100},
		AC: [3]int{100, 100, 100}, Exp: [3]int{100, 100, 100}, Level: [3]int{3, 4, 5},
	}
	plain := ml.Scale(c, Hell, 0, false, nil)
	cl := ml.ScaleOpts(c, Hell, 0, false, nil, Options{Classic: true})

	if cl.HP256 != plain.HP256/2 || cl.AC != plain.AC*10/12 || cl.XP != plain.XP*10/26 || cl.Level != 3+50 {
		t.Errorf("hell classic: %+v vs %+v", cl, plain)
	}

	n := ml.ScaleOpts(c, Nightmare, 0, false, nil, Options{Classic: true})
	if n.Level != 3+25 || n.XP != ml.Scale(c, Nightmare, 0, false, nil).XP*10/17 {
		t.Errorf("nightmare classic: %+v", n)
	}

	if s := ml.ScaleOpts(c, Normal, 0, false, nil, Options{Classic: true}); s.Level != 3 || s.HP != ml.Scale(c, Normal, 0, false, nil).HP {
		t.Error("normal must be untouched")
	}

	c.Align = 1
	if s := ml.ScaleOpts(c, Hell, 0, false, nil, Options{Classic: true}); s.HP != ml.Scale(c, Hell, 0, false, nil).HP {
		t.Error("friendly class must be untouched")
	}
}

func TestLevelExclusionSwitch(t *testing.T) {
	c := &Class{Boss: true, Level: [3]int{4, 5, 6}}
	if c.ResolveLevel(Hell, 9) != 9 {
		t.Error("boss-only class takes area level under the primeevil rule")
	}

	LevelExclusionFlag = ExcludeBoss
	defer func() { LevelExclusionFlag = ExcludePrimeEvil }()

	if c.ResolveLevel(Hell, 9) != 6 {
		t.Error("boss rule")
	}
}
