package d2npc

import "testing"

// w50: the gossip gates. The record arrays of Game.exe (0x7245b0..0x72595f) were read from the file and compared with
// gossip_data.go line by line (464 lines, no difference). 36 records carry a quest gate; a gated line is rolled only
// while bit 0 (reward granted) of the quest slot has the wanted value, which is what ties the gossip to the progress
// of the game. UI_InitNpcMenuEntry 0x4ade00 (verified by decompile): ten rolls, index > 1, hero class fits, gate holds.
var gossipGates = []struct {
	class, rec, msg, quest int
	want                   bool // the wanted value of bit 0 of the quest slot
}{
	{148, 12, 23, 6, false},                                               // Akara: before Andariel
	{154, 3, 39, 4, true}, {147, 3, 48, 4, false}, {147, 7, 52, 6, false}, // Charsi, Gheed
	{150, 3, 27, 4, false}, {150, 4, 28, 7, false}, {150, 6, 30, 3, false}, // Kashya
	{155, 3, 3, 4, true}, {155, 9, 9, 6, false}, {155, 10, 10, 4, false}, // Warriv
	{175, 4, 218, 11, false}, {176, 4, 224, 11, false}, // Warriv 2, Atma
	{177, 5, 290, 13, false}, {177, 9, 294, 14, false}, {177, 10, 295, 11, false}, // Drognan
	{199, 5, 208, 11, false}, {199, 6, 209, 9, false}, {199, 7, 210, 11, false}, // Elzix
	{200, 3, 233, 11, false}, {200, 8, 238, 11, false}, // Geglash
	{198, 3, 192, 11, false}, {198, 10, 199, 14, false}, // Greiz
	{201, 2, 255, 9, false}, {201, 3, 256, 9, false}, {201, 5, 258, 11, false}, {201, 8, 261, 14, false}, // Jerhyn
	{202, 6, 279, 11, false}, {202, 10, 283, 11, false}, // Lysander
	{178, 4, 267, 11, false}, {210, 3, 244, 11, false}, {210, 5, 246, 14, false}, // Fara, Meshif
	{253, 3, 468, 22, false}, {253, 11, 476, 32, true}, {255, 6, 521, 22, false}, // Hratli, Ormus
	{406, 4, 267, 11, false}, {257, 4, 267, 11, false},
}

func TestW50GossipGates(t *testing.T) {
	gated := 0

	for _, e := range gossipEntries {
		for _, r := range e.Records {
			if r.Gated {
				gated++
			}
		}
	}

	if gated != len(gossipGates) {
		t.Errorf("%d gated records in the table, the pinned list has %d", gated, len(gossipGates))
	}

	for _, g := range gossipGates {
		e, ok := EntryFor(g.class)
		if !ok {
			t.Errorf("class %d has no entry", g.class)

			continue
		}

		r := e.Records[g.rec]
		if r.Msg != g.msg || !r.Gated || r.Want != g.want || r.Quest != g.quest {
			t.Errorf("class %d record %d: %+v, want msg %d quest %d want %v", g.class, g.rec, r, g.msg, g.quest, g.want)

			continue
		}

		// the line is a candidate only on the right side of the quest
		open := func(q, bit int) bool { return q == g.quest && bit == 0 && g.want }
		shut := func(q, bit int) bool { return q == g.quest && bit == 0 && !g.want }

		if !r.Eligible(AnyClass, open) || r.Eligible(AnyClass, shut) {
			t.Errorf("class %d record %d: gate does not follow quest %d", g.class, g.rec, g.quest)
		}
	}

	// the Act 5 NPCs have no gated gossip (their pools do not depend on the progress)
	for _, c := range []int{511, 512, 513, 514, 515, 520, 521} {
		if e, ok := EntryFor(c); ok {
			for i, r := range e.Records {
				if r.Gated {
					t.Errorf("class %d record %d is gated", c, i)
				}
			}
		}
	}
}

// TestW50GossipFollowsProgress: Akara's last pool line (msg 23) is only rolled while Andariel is alive (slot 6 bit 0
// clear), Charsi's msg 39 only after the Search for Cain... of slot 4 (bit 0 set).
func TestW50GossipFollowsProgress(t *testing.T) {
	akara, _ := EntryFor(148)

	for _, done := range []bool{false, true} {
		q := func(quest, bit int) bool { return quest == 6 && bit == 0 && done }

		got := map[int]bool{}

		for roll := 0; roll < 13; roll++ {
			idx := PickGossip(akara, 1, q, func(int) int { return roll })
			got[akara.Records[idx].Msg] = true
		}

		if got[23] == done {
			t.Errorf("Andariel done=%v: msg 23 rolled = %v", done, got[23])
		}
	}
}
