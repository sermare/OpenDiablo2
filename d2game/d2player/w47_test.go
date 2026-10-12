package d2player

import "testing"

// Tyrael's expansion row: string 0x58d2, key "Travel To Harrogath" of expansionstring.tbl (VERIFIED w47, 0x4b2f20).
func TestRowTravelHarrogath(t *testing.T) {
	if RowTravelHarrogath.StringID != 0x58d2 || RowTravelHarrogath.Key != "Travel To Harrogath" ||
		RowTravelHarrogath.Action != NPCActionTravelHarrogath {
		t.Fatalf("row = %+v", RowTravelHarrogath)
	}

	if NPCActionTravelHarrogath.String() != "TravelHarrogath" || NPCActionCancel.String() != "Cancel" {
		t.Error("action names are out of step with the constants")
	}

	// the static table still gives Tyrael (Act 4) Talk only: the travel row is added by the client at run time
	rows, _ := NPCMenuFor(367)
	if len(rows) != 1 || rows[0].Action != NPCActionTalk {
		t.Errorf("Tyrael static rows = %+v", rows)
	}
}
