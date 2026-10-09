package d2player

import "testing"

func TestNPCMenuFor(t *testing.T) {
	cases := []struct {
		id   int
		want []NPCMenuAction
	}{
		{148, []NPCMenuAction{NPCActionTalk, NPCActionTrade}},                  // Akara
		{154, []NPCMenuAction{NPCActionTalk, NPCActionTradeRepair}},            // Charsi
		{147, []NPCMenuAction{NPCActionTalk, NPCActionTrade, NPCActionGamble}}, // Gheed
		{252, []NPCMenuAction{NPCActionTalk, NPCActionHire, NPCActionTrade}},   // Asheara
		{257, []NPCMenuAction{NPCActionTradeRepair}},                           // Halbu: no Talk
		{405, []NPCMenuAction{NPCActionTrade, NPCActionGamble}},                // Jamella
		{244, []NPCMenuAction{NPCActionTalk, NPCActionIdentify}},               // Cain
		{175, []NPCMenuAction{NPCActionTalk, NPCActionTravelWest}},             // Warriv2
		{150, []NPCMenuAction{NPCActionTalk, NPCActionHire}},                   // Kashya
	}

	for _, c := range cases {
		rows, ok := NPCMenuFor(c.id)
		if !ok {
			t.Fatalf("class %d (%s) missing", c.id, NPCClassName(c.id))
		}

		if len(rows) != len(c.want) {
			t.Fatalf("class %d: got %v want %v", c.id, rows, c.want)
		}

		for i := range rows {
			if rows[i].Action != c.want[i] {
				t.Fatalf("class %d row %d: got %v want %v", c.id, i, rows[i].Action, c.want[i])
			}
		}
	}
}

func TestNPCMenuStringIDs(t *testing.T) {
	// ids documented in ui-npc.md
	if rowTalk.StringID != 0xd35 || rowTrade.StringID != 0xd44 || rowTradeRepair.StringID != 0xd06 ||
		rowGamble.StringID != 0xd46 || rowHire.StringID != 0xd45 || rowIdentify.StringID != 0xfb4 {
		t.Fatal("string ids differ from the notes")
	}
}

func TestNPCMenuUnknownDefaultsToTalk(t *testing.T) {
	rows, ok := NPCMenuFor(99999)
	if ok || len(rows) != 1 || rows[0].Action != NPCActionTalk {
		t.Fatalf("unexpected default %v %v", rows, ok)
	}
}
