package d2quest

// What the NPCs say after the three end-boss kills (Mephisto, Diablo, Baal).
//
// VERIFIED (Game.exe 1.14b, read-only Ghidra; details in the notes boss-kill-speech.md): these lines are NOT ambient sounds fired by the
// kill. They are ordinary quest speech lists that the server builds when the hero clicks Talk on the NPC (the OnNpcInteract callbacks
// 0x5b9e20, 0x5b23b0 and 0x58b7d0) and that depend on per-hero record bits. Hearing a line (message acknowledged, OnNpcMessageHeard
// 0x5ba140, 0x5b2290, 0x58b720) flips a bit so the line is spoken once and the NPC then offers it as a Talk topic.
//
//   - A3Q6 (slot 22): the Mephisto kill sets bit 11. While it is set every Kurast NPC with a row in speech table 5 speaks its SUCCESS
//     line (Alkor 657, Ormus 658, Meshif 659, Asheara 660, Hratli 661, Cain 662, Natalya 663). Hearing one of them clears bit 11 and
//     remembers the hero in the node's "recently rewarded" list (0x543070); afterwards the same NPCs offer the line as a topic (table 6)
//     while the quest is done. A timer 0x5b9dc0 / event 0x1e7 run on hearing when the primary goal bit is set: not modelled.
//   - A4Q2 (slot 26), classic: the Diablo kill sets bits 6 and 7; Tyrael speaks 684 while bit 7 is set and Cain 685 while bit 6 is set
//     (table 2); hearing 684 clears bit 7, 685 clears bit 6; the other NPC offers its line as a topic (table 3).
//     Expansion: with the quest done, Tyrael speaks 20000 (table 4, plus the topic copy table 5) until bit 9 is set by hearing 20000
//     (the same handler opens the portal to Harrogath through 0x5b2200: not modelled here), Cain 20001 until bit 8 is set by hearing 20001.
//   - A5Q6 (slot 40): once done (bit 0), Tyrael speaks 20175 (table 2) whenever the primary goal bit (13) is set, every time; Larzuk
//     20178 / Anya 20176 / Malah 20179 / Qual-Kehk 20180 speak their line (table 2) until the per-NPC "heard" bit 4 / 9 / 6 / 8 is set
//     by hearing it, then offer it as a topic (table 3) while bit 13 is set. Cain (class 520, bit 5) goes through tables 4/5 and a bit 10
//     of the node's slot at +0xe0: not decoded, left out.
//
// UNVERIFIED: that the node's slot at +0xe0 (read by 0x5b9e20, 0x5b23b0, 0x58b7d0) is the quest's own slot; the "!" marker (CanNpcTalk)
// of these quests while the lines are pending; the engine keeps the "recently rewarded" list as a flag of the quest data.

// FlagTownCheers is bit 0xb of A3Q6: set by Mephisto's death, cleared when a Kurast NPC's success line has been heard.
const FlagTownCheers = exeBitGuardianCheer

const (
	exeBitGuardianCheer = 11 // A3Q6 bit 0xb
	exeBitTyraelPending = 7  // A4Q2 classic: Tyrael's 684 not heard yet
	exeBitCainPending   = 6  // A4Q2 classic: Cain's 685 not heard yet
	exeBitExpCainHeard  = 8  // A4Q2 expansion: Cain's 20001 heard
	exeBitExpTyrHeard   = 9  // A4Q2 expansion: Tyrael's 20000 heard
)

// killSpeech is the post-kill speech of one end boss quest, active unless Game.LegacyBossBits is set.
type killSpeech struct {
	activate func(g *Game, q *Quest, d *genData, npc int) []Speech
	heard    func(g *Game, q *Quest, d *genData, e *Event)
}

func guardianSpeech() *killSpeech {
	return &killSpeech{
		activate: func(g *Game, q *Quest, d *genData, npc int) []Speech {
			switch {
			case g.get(q, exeBitGuardianCheer):
				return q.pick(npc, 5)
			case g.get(q, FlagRewardGranted) && d.cheered:
				return q.pick(npc, 6)
			}

			return nil
		},
		heard: func(g *Game, q *Quest, d *genData, e *Event) {
			if e.Msg < 657 || e.Msg > 663 || !g.get(q, exeBitGuardianCheer) {
				return
			}

			g.clear(q, exeBitGuardianCheer, "town cheers heard")

			d.cheered = true
		},
	}
}

func terrorSpeech() *killSpeech {
	return &killSpeech{
		activate: func(g *Game, q *Quest, d *genData, npc int) []Speech {
			if !g.Expansion {
				tyr, cain := g.get(q, exeBitTyraelPending), g.get(q, exeBitCainPending)

				switch {
				case tyr && npc == NPCTyrael2, cain && npc == NPCCain4:
					return q.pick(npc, 2)
				case tyr && npc == NPCCain4, !tyr && cain && npc == NPCTyrael2:
					return q.pick(npc, 3)
				}

				return nil
			}

			if !g.get(q, FlagRewardGranted) {
				return nil
			}

			var out []Speech

			if !g.get(q, exeBitExpTyrHeard) {
				switch npc {
				case NPCTyrael2:
					out = append(out, q.pick(npc, 4)...)
				case NPCCain4:
					out = append(out, q.pick(npc, 5)...)
				}
			}

			if !g.get(q, exeBitExpCainHeard) {
				switch npc {
				case NPCCain4:
					out = append(out, q.pick(npc, 4)...)
				case NPCTyrael2:
					out = append(out, q.pick(npc, 5)...)
				}
			}

			return out
		},
		heard: func(g *Game, q *Quest, d *genData, e *Event) {
			switch {
			case !g.Expansion && e.Msg == 684 && e.NPC == NPCTyrael2:
				g.clear(q, exeBitTyraelPending, "Tyrael's 684 heard")
			case !g.Expansion && e.Msg == 685 && e.NPC == NPCCain4:
				g.clear(q, exeBitCainPending, "Cain's 685 heard")
			case g.Expansion && e.Msg == 20000 && e.NPC == NPCTyrael2:
				g.set(q, exeBitExpTyrHeard, "Tyrael's 20000 heard")
			case g.Expansion && e.Msg == 20001 && e.NPC == NPCCain4:
				g.set(q, exeBitExpCainHeard, "Cain's 20001 heard")
			}
		},
	}
}

// Per-NPC "heard" bits of A5Q6 (slot 40), by NPC class and by message id.
//
//nolint:gochecknoglobals // static lookup data
var (
	eveHeardBit = map[int]int{NPCLarzuk: 4, NPCDrehya: 9, NPCMalah: 6, NPCQualKehk: 8}
	eveMsgBit   = map[int]int{20178: 4, 20176: 9, 20179: 6, 20180: 8, 20177: 5, 20175: 7}
)

func eveSpeech() *killSpeech {
	return &killSpeech{
		activate: func(g *Game, q *Quest, d *genData, npc int) []Speech {
			if !g.get(q, FlagRewardGranted) {
				return nil
			}

			goal := g.get(q, FlagPrimaryGoal)

			if npc == NPCTyrael3 {
				if goal {
					return q.pick(npc, 2)
				}

				return nil
			}

			bit, ok := eveHeardBit[npc]

			switch {
			case !ok:
				return nil
			case !g.get(q, bit):
				return q.pick(npc, 2)
			case goal:
				return q.pick(npc, 3)
			}

			return nil
		},
		heard: func(g *Game, q *Quest, d *genData, e *Event) {
			if bit, ok := eveMsgBit[e.Msg]; ok && g.get(q, FlagRewardGranted) {
				g.set(q, bit, "Eve success line heard")
			}
		},
	}
}
