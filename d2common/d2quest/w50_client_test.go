package d2quest

import "testing"

// TestW50BetrayalPages: goal done shows page 4 (talk to Anya) and 5 once she spoke (bit 4); granted shows nothing (0x588d90).
func TestW50BetrayalPages(t *testing.T) {
	g, _ := newGame(t)
	g.Hero.Level = 30
	g.ForceDone(QuestRescue)
	g.ForceDone(QuestPrison)

	moveTo(g, 1, LevelHarrogath)
	talk(g, NPCDrehya)
	pageIs(t, g, QuestBetrayal, 1, "Anya opened the portal")

	moveTo(g, LevelHarrogath, LevelNihlathakTemple)
	g.Tick(20)

	kill(g, NPCNihlathakBoss, LevelNihlathakTemple)
	g.Tick(20)
	pageIs(t, g, QuestBetrayal, 4, "Nihlathak dead: talk to Anya")

	q := g.Quest(QuestBetrayal)
	g.set(q, FlagEnterArea, "Anya spoke (msg 20148)")
	pageIs(t, g, QuestBetrayal, 5, "Anya personalises an item")

	g.set(q, FlagRewardGranted, "personalised")
	pageIs(t, g, QuestBetrayal, 0, "granted")
}

type flagSet map[[2]int]bool

func (f flagSet) get(slot, bit int) bool { return f[[2]int{slot, bit}] }

func (f flagSet) on(slot int, bits ...int) flagSet {
	for _, b := range bits {
		f[[2]int{slot, b}] = true
	}

	return f
}

// TestW50ClientView walks the rules of UI_BuildQuestLogEntries (0x49e060).
func TestW50ClientView(t *testing.T) {
	for _, c := range []struct {
		name       string
		slot, page int
		flags      flagSet
		global     flagSet
		multi      bool
		wantKey    string
		wantCat    LogCategory
		wantPage   int
	}{
		{"den running", 1, 2, flagSet{}, nil, false, "qstsa1q12", LogCatActive, 2},
		{"den done in this game, unseen", 1, 13, flagSet{}.on(1, 0, 13), nil, false, "qstsComplete", LogCatAnimating, 13},
		{"den done in this game, seen", 1, 13, flagSet{}.on(1, 0, 13, 12), nil, false, "qstsComplete", LogCatDone, 13},
		{"den done in an earlier game", 1, 0, flagSet{}.on(1, 0, 12), nil, false, "qstsprevious", LogCatDone, 11},
		{"den reward pending shows page 5", 1, 4, flagSet{}.on(1, 1, 13), nil, false, "qstsa1q15", LogCatActive, 5},
		{"quest not available", 3, 0, flagSet{}, nil, false, "", LogCatLocked, 0},
		{"tools reward pending", 3, 10, flagSet{}.on(3, 1, 13, 15), nil, false, "qstsa1q32b", LogCatActive, 10},
		{"tools reward pending without the CB bit", 3, 10, flagSet{}.on(3, 1, 13), nil, false, "qstsa1q32b", LogCatActive, 10},
		{"cain: someone else rescued", 4, 0, flagSet{}, flagSet{}.on(4, 13), false, "qstsThankYouComeAgain", LogCatActive, 0},
		{"cain: someone else rescued, multi", 4, 0, flagSet{}, flagSet{}.on(4, 13), true, "qstsThankYouComeAgainMulti", LogCatActive, 0},
		{"cannot complete bit", 6, 2, flagSet{}.on(6, 14), nil, false, "qstsThankYouComeAgain", LogCatActive, 2},
		{"staff: assembled", 10, 6, flagSet{}, nil, false, "qstsa2q24", LogCatActive, 6},
		{"siege: reward pending, page byte 0 reads as 4", 35, 0, flagSet{}.on(35, 1), nil, false, "qsta5q14", LogCatActive, 4},
		{"siege: sockets added", 35, 4, flagSet{}.on(35, 0, 13), nil, false, "qstsComplete", LogCatAnimating, 13},
		{"tombs: arcane done, page 1 shows the symbol", 14, 1, flagSet{}.on(12, 0), nil, false, "qstsa2q61a", LogCatActive, 7},
		{"tombs: Tyrael's message", 14, 2, flagSet{}.on(14, 3), nil, false, "qstsa2q64", LogCatActive, 5},
		{"tombs: Jerhyn's thanks", 14, 2, flagSet{}.on(14, 4), nil, false, "qstsa2q65", LogCatActive, 6},
		{"prison: reward pending, no scroll", 37, 5, flagSet{}.on(37, 1, 13, 15), nil, false, "qstsa5q34", LogCatActive, 5},
		{"prison: scroll given, Anya not", 37, 6, flagSet{}.on(37, 1, 13, 15, 8), nil, false, "qstsa5q35", LogCatActive, 6},
		{"betrayal: Anya spoke", 38, 5, flagSet{}.on(38, 1, 13, 15, 4), nil, false, "qstsa5q43a", LogCatActive, 5},
		{"hellforge: reward pending shows the done icon", 27, 4, flagSet{}.on(27, 1, 13, 15), nil, false, "qstsprevious", LogCatDone, 10},
		{"fallen angel: the page byte decides", 25, 3, flagSet{}.on(25, 1, 13), nil, false, "qstsa4q13a", LogCatActive, 3},
		{"temple: council dead elsewhere", 21, 0, flagSet{}.on(21, 4), flagSet{}.on(21, 13), false, "qstsa3q53", LogCatActive, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			ctx := LogCtx{Flag: c.flags.get, Multi: c.multi}
			if c.global != nil {
				ctx.Global = c.global.get
			}

			v := ClientLogView(c.slot, c.page, ctx)
			if v.Key != c.wantKey || v.Cat != c.wantCat || v.Page != c.wantPage {
				t.Errorf("got key=%q cat=%d page=%d, want key=%q cat=%d page=%d", v.Key, v.Cat, v.Page, c.wantKey, c.wantCat, c.wantPage)
			}
		})
	}
}

// TestW50ClientViewCounts: the Den counts monsters, the Rescue quest soldiers.
func TestW50ClientViewCounts(t *testing.T) {
	flags := flagSet{}

	v := ClientLogView(1, 4, LogCtx{Flag: flags.get, DenLeft: 7})
	if v.Key != "qstsa1q14" || v.Count != 7 || !v.Append {
		t.Errorf("den, 7 left: %+v", v)
	}

	if v = ClientLogView(1, 4, LogCtx{Flag: flags.get, DenLeft: 1}); v.Key != "qstsa1q140" {
		t.Errorf("den, 1 left: %+v", v)
	}

	if v = ClientLogView(36, 2, LogCtx{Flag: flags.get, Soldiers: 10}); v.Key != "qstsa5q22" || v.Count != 10 {
		t.Errorf("rescue, 10 left: %+v", v)
	}

	if v = ClientLogView(36, 2, LogCtx{Flag: flags.get}); v.Key != "qstsThankYouComeAgain" {
		t.Errorf("rescue, none counted: %+v", v)
	}
}
