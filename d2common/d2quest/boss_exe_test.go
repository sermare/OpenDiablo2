package d2quest

import "testing"

func bitOf(b int) uint16 { return 1 << b }

// TestExeBossBits pins the kill bits read from the binary (boss_exe.go).
func TestExeBossBits(t *testing.T) {
	for _, c := range []struct {
		name     string
		ev       Event
		level    int
		classic  bool
		id       int
		want     uint16
		wantNone bool
		unlockFx bool
	}{
		{name: "mephisto", ev: Event{Kind: EvMonsterKilled, Monster: NPCMephisto, Name: "Mephisto"}, id: QuestGuardian,
			want: bitOf(FlagPrimaryGoal) | bitOf(FlagRewardGranted) | bitOf(exeBitMephisto)},
		{name: "diablo expansion", ev: Event{Kind: EvMonsterKilled, Monster: NPCDiablo, Name: "Diablo"}, id: QuestTerrorsEnd,
			want: bitOf(FlagPrimaryGoal) | bitOf(FlagRewardGranted)},
		{name: "diablo classic", ev: Event{Kind: EvMonsterKilled, Monster: NPCDiablo, Name: "Diablo"}, classic: true, id: QuestTerrorsEnd,
			want: bitOf(FlagPrimaryGoal) | bitOf(FlagRewardGranted) | bitOf(6) | bitOf(7)},
		{name: "baal in the chamber", ev: Event{Kind: EvMonsterKilled, Monster: NPCBaalCrab, Name: "Baal"}, level: LevelWorldstoneChamber,
			id: QuestEveOfDestruction, want: bitOf(FlagPrimaryGoal) | bitOf(FlagRewardGranted), unlockFx: true},
		{name: "baal elsewhere", ev: Event{Kind: EvMonsterKilled, Monster: NPCBaalCrab, Name: "Baal"}, level: LevelThrone,
			id: QuestEveOfDestruction, wantNone: true},
	} {
		g, _ := newGame(t)
		g.ExeBossBits = true
		g.Expansion = !c.classic
		g.Level = c.level

		ev := c.ev
		ev.Level = c.level

		effects := g.Dispatch(ev)

		got := g.Rec.Slot(g.Quest(c.id).Slot)
		if c.wantNone {
			if got != 0 {
				t.Errorf("%s: slot = 0x%04x, want untouched", c.name, got)
			}

			continue
		}

		if c.name == "mephisto" {
			drop := false

			for _, f := range effects {
				drop = drop || (f.Kind == EffectGiveItem && f.Code == ItemMephistoSoulstone)
			}

			if !drop {
				t.Errorf("mephisto: no soulstone drop")
			}
		}

		// the exe never sets reward pending for these kills
		if got&bitOf(FlagRewardPending) != 0 {
			t.Errorf("%s: reward pending set (0x%04x)", c.name, got)
		}

		if got&c.want != c.want {
			t.Errorf("%s: slot = 0x%04x, want at least 0x%04x", c.name, got, c.want)
		}

		if c.unlockFx && !hasEffect(effects, EffectReward) {
			t.Errorf("%s: claim effects (unlock difficulty) not paid", c.name)
		}
	}
}

// TestExeBossBitsOffByDefault: the default flow still ends in reward pending.
func TestExeBossBitsOffByDefault(t *testing.T) {
	g, _ := newGame(t)
	g.Dispatch(Event{Kind: EvMonsterKilled, Monster: NPCMephisto, Name: "Mephisto"})

	got := g.Rec.Slot(g.Quest(QuestGuardian).Slot)
	if got&bitOf(FlagRewardPending) == 0 || got&bitOf(FlagRewardGranted) != 0 {
		t.Errorf("default flow changed: 0x%04x", got)
	}
}

// TestHephastoDropsTheHammer pins 0x5b4190 / 0x5af8c0: with ExeBossBits Hephasto's death (class 409) drops the Hellforge
// hammer and changes no quest bit; without the option nothing happens.
func TestHephastoDropsTheHammer(t *testing.T) {
	for _, on := range []bool{false, true} {
		g, _ := newGame(t)
		g.ExeBossBits = on
		before := g.Rec.Slot(g.Quest(QuestHellforge).Slot)

		effects := g.Dispatch(Event{Kind: EvMonsterKilled, Monster: NPCHephasto, Name: "hephasto"})

		drop := false

		for _, f := range effects {
			drop = drop || (f.Kind == EffectGiveItem && f.Code == ItemHellforgeHammer)
		}

		if drop != on {
			t.Errorf("ExeBossBits=%v: hammer drop = %v", on, drop)
		}

		if after := g.Rec.Slot(g.Quest(QuestHellforge).Slot); after != before {
			t.Errorf("ExeBossBits=%v: slot changed 0x%04x -> 0x%04x", on, before, after)
		}
	}
}
