package d2player

import "testing"

func TestGoldDialogSized(t *testing.T) {
	l := GoldDialogSized(GoldDlgDeposit, 250, 150, GoldDlgSizes{OkW: 40, OkH: 24, CancelW: 36, CancelH: 28, SpinnerW: 14, SpinnerH: 20})

	if l.Ok.String() != "golddlg ok 250 263 40 24" || l.Cancel.String() != "golddlg cancel 355 259 36 28" ||
		l.Spinner.String() != "golddlg spinner 223 219 14 20" {
		t.Errorf("sized: %v %v %v", l.Ok, l.Cancel, l.Spinner)
	}

	if l.Packet != 0x4f || l.PacketArg != 0x14 {
		t.Errorf("packet %+v", l)
	}
}
