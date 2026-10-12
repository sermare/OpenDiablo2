package d2quest

// Quest log getters (node+0xE8, "GetLogState") of Game.exe 1.14b that the quests did not have yet. Each returns the page
// byte the server pushes to the client; ClientLogView turns it into a text. VERIFIED by reading the getters:
// Siege 0x584bd0, Betrayal 0x588d90, Seven Tombs 0x59a660, Golden Bird 0x5b7d90, Horadric Staff 0x59c1f0.

// siegeLogPage is QUEST_A5_SiegeOnHarrogath_GetLogState: reward pending or goal done shows page 3 until Larzuk's
// "I will add sockets" line was heard (bit 5), then 4; otherwise the stored page while the quest runs (state <= 4).
func siegeLogPage(g *Game, q *Quest) int {
	if g.get(q, FlagRewardPending) || g.get(q, FlagPrimaryGoal) {
		if g.get(q, FlagCustom1) {
			return 4
		}

		return 3
	}

	switch {
	case !q.NotIntro, g.get(q, FlagRewardGranted):
		return 0
	case g.get(q, FlagCompletedNow):
		return 12
	case q.State > 4:
		return 0
	}

	return q.LastState
}

// betrayalLogPage is QUEST_A5_BetrayalOfHarrogath_GetLogState: reward granted shows nothing; goal done or reward
// pending shows page 4 (talk to Anya) and, once Anya spoke (bit 4), 5 (she personalises an item); otherwise the stored
// page while the state is at most 3.
func betrayalLogPage(g *Game, q *Quest) int {
	switch {
	case g.get(q, FlagRewardGranted):
		return 0
	case g.get(q, FlagRewardPending) || g.get(q, FlagPrimaryGoal):
		if g.get(q, FlagEnterArea) {
			return 5
		}

		return 4
	case !q.NotIntro:
		return 0
	case g.get(q, FlagCompletedNow):
		return 12
	case q.State > 3:
		return 0
	}

	return q.LastState
}

// tombsLogPage is QUEST_A2_TheSevenTombs_GetLogState. UNVERIFIED part not modelled: the "staff items lost" page
// (8 or 9) the Horadric Staff quest stores when the hero dropped the parts and cannot rebuild the staff.
func tombsLogPage(g *Game, q *Quest) int {
	if g.get(q, FlagRewardGranted) {
		return 0
	}

	switch {
	case g.get(q, FlagLeaveTown):
		return 5
	case g.get(q, FlagEnterArea):
		return 6
	case !q.NotIntro:
		return 0
	case g.get(q, FlagCustom1), q.State < 3:
		return q.LastState
	}

	return 12
}

// goldenBirdLogPage is QUEST_A3_TheGoldenBird_GetLogState: reward pending 5, granted 13/11; the bird in the pack shows
// 3, and 4 once Cain was asked about it (bit 4); the figurine shows 1, and 2 once Cain heard of it (bit 2); otherwise
// the stored page, where a stored page above 5 is shown as 7 in a game that is not the network type 3 (6 otherwise).
func goldenBirdLogPage(g *Game, q *Quest) int {
	switch {
	case g.get(q, FlagRewardPending):
		return 5
	case g.get(q, FlagRewardGranted):
		if g.get(q, FlagPrimaryGoal) {
			return 13
		}

		return 11
	case g.hasItem(ItemGoldenBird):
		if g.get(q, FlagEnterArea) {
			return 4
		}

		return 3
	case g.hasItem(ItemJadeFigurine):
		if g.get(q, FlagStarted) {
			return 2
		}

		return 1
	case !q.NotIntro:
		return 0
	case q.LastState > 5:
		return 7
	}

	return q.LastState
}

// staffLogPage is QUEST_A2_HoradricStaff_GetLogState. Pages (strings in the table of logpages.csv): 1 show the scroll to
// Cain, 2 search for the parts, 3 restore the staff in the cube, 4 take the artifacts to Cain, 5 and 6 take the staff
// to the tomb, 9 the hero lost parts for good (not modelled).
func staffLogPage(g *Game, q *Quest, assembled bool) int {
	if g.get(q, FlagRewardGranted) || g.get(q, FlagRewardPending) {
		if t := g.byID[QuestSevenTombs]; t != nil && g.get(t, FlagRewardGranted) {
			if g.get(q, FlagPrimaryGoal) {
				return 13
			}

			return 11
		}

		return custom6Page(g, q)
	}

	scroll, shaft, box := g.hasItem(ItemHoradricScroll), g.hasItem(ItemStaffOfKingsShaft), g.hasItem(ItemHoradricCube)
	amulet, staff := g.hasItem(ItemViperAmulet), g.hasItem(ItemHoradricStaff) || assembled
	leave := g.get(q, FlagLeaveTown)

	if shaft && box && amulet {
		switch {
		case g.get(q, FlagCustom2):
			return 3
		case leave:
			return 2
		}

		return 4
	}

	switch {
	case staff:
		return custom6Page(g, q)
	case scroll && !leave:
		return 1
	case leave:
		return 2
	case amulet || shaft || box:
		return 4
	}

	return 0
}

func custom6Page(g *Game, q *Quest) int {
	if g.get(q, FlagCustom6) {
		return 5
	}

	return 6
}
