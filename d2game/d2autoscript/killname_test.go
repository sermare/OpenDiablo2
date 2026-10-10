package d2autoscript

import "testing"

// kill:name=<text>[,seconds] fights the monsters of one name (a quest boss somewhere in a big level).
func TestParseKillName(t *testing.T) {
	steps, err := Parse("kill:name=Shenk the Overseer,200;kill:name=Izual")
	if err != nil {
		t.Fatal(err)
	}

	if len(steps) != 2 || steps[0].Target != "name" || steps[0].Arg != "Shenk the Overseer" || steps[0].Seconds != 200 {
		t.Errorf("first step %+v", steps[0])
	}

	if steps[1].Arg != "Izual" || steps[1].Seconds != DefaultKillSeconds {
		t.Errorf("second step %+v", steps[1])
	}

	if _, err := Parse("kill:name="); err == nil {
		t.Error("kill:name= without a name must be refused")
	}
}
