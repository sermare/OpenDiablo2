package d2player

import "testing"

func TestGoldDialogGolden(t *testing.T) {
	l := GoldDialog(GoldDlgDrop, 250, 150)

	got := []string{l.Frame.String(), l.Edit.String(), l.Ok.String(), l.Cancel.String()}
	want := []string{
		"golddlg frame 215 140 250 150",
		"golddlg edit 258 228 100 20",
		"golddlg ok 250 255 32 32",
		"golddlg cancel 355 255 32 32",
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%d: got %q want %q", i, got[i], want[i])
		}
	}

	if l.Spinner.X != 223 || l.Spinner.Y != 219 {
		t.Errorf("spinner %v", l.Spinner)
	}

	if l.TextBaseline0 != 164 || l.TextID != 0xfc1 || l.Packet != 0x50 || l.StatLimit != 14 {
		t.Errorf("drop: %+v", l)
	}

	w := GoldDialog(GoldDlgWithdraw, 250, 150)
	if w.TextID != 0xfd2 || w.StatLimit != 15 || w.PacketArg != 0x13 {
		t.Errorf("withdraw: %+v", w)
	}
}
