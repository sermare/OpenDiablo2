package d2player

import "testing"

// The rows of the reward NPCs, from the exe's menu table (ui-npc.md, 0x725d60).
// The reward itself is not a row of the table: Larzuk, Anya and Charsi take an
// item dropped on them, Akara's reset row is added by the game screen.
func TestRewardNPCMenus(t *testing.T) {
	cases := []struct {
		name string
		id   int
		want []NPCMenuAction
	}{
		{"Larzuk", 511, []NPCMenuAction{NPCActionTalk, NPCActionTradeRepair}},
		{"Anya", 512, []NPCMenuAction{NPCActionTalk, NPCActionTrade, NPCActionGamble}},
		{"Malah", 513, []NPCMenuAction{NPCActionTalk, NPCActionTrade}},
		{"Charsi", 154, []NPCMenuAction{NPCActionTalk, NPCActionTradeRepair}},
		{"Akara", 148, []NPCMenuAction{NPCActionTalk, NPCActionTrade}},
		{"Kashya", 150, []NPCMenuAction{NPCActionTalk, NPCActionHire}},
		{"Qual-Kehk", 515, []NPCMenuAction{NPCActionTalk, NPCActionHire}},
		{"Asheara", 252, []NPCMenuAction{NPCActionTalk, NPCActionHire, NPCActionTrade}},
		{"Hratli", 253, []NPCMenuAction{NPCActionTalk, NPCActionTradeRepair}},
		{"Jerhyn", 201, []NPCMenuAction{NPCActionTalk}},
		{"Warriv1", 155, []NPCMenuAction{NPCActionTalk}},
		{"Warriv2", 175, []NPCMenuAction{NPCActionTalk, NPCActionTravelWest}},
		{"Tyrael2", 367, []NPCMenuAction{NPCActionTalk}},
		{"Nihlathak", 514, []NPCMenuAction{NPCActionTalk, NPCActionGamble}},
		{"Cain5", 265, []NPCMenuAction{NPCActionTalk, NPCActionIdentify}},
		{"Cain6", 520, []NPCMenuAction{NPCActionTalk, NPCActionIdentify}},
	}

	for _, c := range cases {
		rows, ok := NPCMenuFor(c.id)
		if !ok || len(rows) != len(c.want) {
			t.Errorf("%s (%d): rows %v known=%v, want %v", c.name, c.id, rows, ok, c.want)

			continue
		}

		for i, r := range rows {
			if r.Action != c.want[i] {
				t.Errorf("%s row %d: %v, want %v", c.name, i, r.Action, c.want[i])
			}

			if r.Fallback == "" {
				t.Errorf("%s row %d has no label", c.name, i)
			}
		}
	}
}

func TestRewardRowActions(t *testing.T) {
	if NPCActionReward.String() != "Reward" || NPCActionRespec.String() != "Respec" {
		t.Errorf("names %q %q", NPCActionReward, NPCActionRespec)
	}

	if RowRespec.StringID != 0x2ba0 || RowRespec.Action != NPCActionRespec || RowRespec.Fallback == "" {
		t.Errorf("respec row %+v", RowRespec)
	}

	// the new actions come after the hire rows: existing numbers do not move
	if NPCActionReviveMerc >= NPCActionReward || NPCActionReward >= NPCActionRespec {
		t.Error("action order changed")
	}
}
