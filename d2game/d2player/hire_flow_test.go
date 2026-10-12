package d2player

import (
	"strings"
	"testing"
)

func TestHireClick(t *testing.T) {
	for _, tc := range []struct {
		hasMerc bool
		want    HireClickAction
	}{{false, HireSendRequest}, {true, HireConfirmReplace}} {
		if got := HireClick(tc.hasMerc); got != tc.want {
			t.Errorf("HireClick(%v) = %v, want %v", tc.hasMerc, got, tc.want)
		}
	}
}

func TestHireReplyKey(t *testing.T) {
	for code, want := range map[int]string{
		0: "", 1: "", 3: "", 5: "", 6: "",
		7: "TransactionResults6", 8: "TransactionResults6", 10: "TransactionResults4", 11: "TransactionResults8",
		12: "TransactionResults10", 15: "TransactionResults11", 16: "TransactionResults5",
		2: "TransactionResults9", 99: "TransactionResults9",
	} {
		if got := HireReplyKey(code); got != want {
			t.Errorf("HireReplyKey(%d) = %q, want %q", code, got, want)
		}
	}

	if HireReplyKey(HireReplyNoGold) == "" {
		t.Error("not enough gold has no message")
	}
}

func TestHireListGeometry(t *testing.T) {
	x, y, w, h := HireListRect(800, 600)
	if x != 155 || y != 300-0xa0 || w != 0x1ea || h != 0x118 {
		t.Errorf("list rect = %d,%d %dx%d", x, y, w, h)
	}

	dx, dy, dw, dh := HireListDialogRect(800, 600)
	if dx != x || dy != y-0x1e || dw != 0x1ea || dh != 0x15e {
		t.Errorf("dialog rect = %d,%d %dx%d", dx, dy, dw, dh)
	}

	// six rows of 46 pixels; an entry past the data or the sixth row is not hit
	for _, tc := range []struct{ first, n, py, want int }{
		{0, 10, y + 1, 0}, {0, 10, y + 46, 1}, {0, 10, y + 5*46 + 3, 5}, {2, 10, y + 1, 2}, {0, 3, y + 3*46 + 1, -1}, {0, 10, y - 1, -1},
	} {
		if got := HireListRowAt(800, 600, tc.first, tc.n, x+10, tc.py); got != tc.want {
			t.Errorf("row at first=%d n=%d py=%d = %d, want %d", tc.first, tc.n, tc.py, got, tc.want)
		}
	}

	if got := HireListRowAt(800, 600, 0, 10, x-1, y+1); got != -1 {
		t.Errorf("left of the list hit entry %d", got)
	}

	for _, tc := range []struct{ first, n, want int }{{-3, 10, 0}, {9, 10, 4}, {1, 4, 0}, {2, 10, 2}} {
		if got := HireListClampFirst(tc.first, tc.n); got != tc.want {
			t.Errorf("clamp(%d,%d) = %d, want %d", tc.first, tc.n, got, tc.want)
		}
	}
}

func TestHireListChoose(t *testing.T) {
	h := &HireList{hover: -1}

	var got []NPCMenuRow

	entries := []HireListEntry{{Line1: "a", Row: NPCMenuRow{StringID: 3, Action: NPCActionHireOffer}}, {Line1: "b", Row: NPCMenuRow{StringID: 7, Action: NPCActionHireOffer}}}
	h.Open("title", "cancel", entries, func(r NPCMenuRow) { got = append(got, r) })

	if !h.IsOpen() || !h.Contains(400, 300) {
		t.Fatal("not open or the centre of the screen is not on the dialog")
	}

	// a click on the second row (y of the list + 46..91)
	_, ly, _, _ := HireListRect(800, 600)
	if i := h.at(400, ly+50); i != 1 {
		t.Fatalf("click at the second row hit %d", i)
	}

	h.Choose(1)

	if len(got) != 1 || got[0].StringID != 7 || h.IsOpen() {
		t.Errorf("choose: got %v open=%v", got, h.IsOpen())
	}

	// Cancel is the entry after the last one
	h.Open("title", "cancel", entries, func(r NPCMenuRow) { got = append(got, r) })

	_, dy, _, _ := HireListDialogRect(800, 600)
	if i := h.at(400, dy+HireListCancelY); i != len(entries) {
		t.Fatalf("the cancel entry is hit as %d", i)
	}

	h.Choose(len(entries))

	if last := got[len(got)-1]; last.Action != NPCActionCancel {
		t.Errorf("cancel entry gave %v", last.Action)
	}
}

func TestClipPortraitName(t *testing.T) {
	measure := func(s string) int { return 8 * len([]rune(s)) }

	for in, want := range map[string]string{"": "", "Rogue": "Rogue", "Desert Mercenary": "Desert M", "Iron Wolf": "Iron Wol"} {
		if got := ClipPortraitName(in, measure); got != want {
			t.Errorf("clip(%q) = %q, want %q", in, got, want)
		}
	}

	if got := ClipPortraitName("abc", func(string) int { return 1000 }); got != "" {
		t.Errorf("an unfit name left %q", got)
	}
}

func TestMercPanelPictureIsOwn(t *testing.T) {
	if !strings.HasSuffix(strings.ToLower(mercPanelArt()), "npcinv.dc6") {
		t.Errorf("the hireling screen uses %q", mercPanelArt())
	}
}
