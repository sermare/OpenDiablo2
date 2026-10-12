package d2state

import "testing"

const panelTable = "state\tid\tarmblue\tarmred\trfblue\trcblue\trlblue\trpblue\trfred\trcred\trlred\trpred\tdmgblue\tdmgred\tattblue\tattred\n" +
	"none\t0\t\t\t\t\t\t\t\t\t\t\t\t\t\t\n" +
	"iron\t1\t1\t\t\t\t\t\t\t\t\t\t\t\t\t\n" +
	"curse\t2\t\t1\t\t\t\t\t1\t1\t1\t\t\t\t\t\n" +
	"fade\t3\t\t\t1\t1\t1\t1\t\t\t\t\t\t\t\t\n" +
	"rage\t4\t\t\t\t\t\t\t\t\t\t\t1\t\t1\t\n" +
	"weak\t5\t\t\t\t\t\t\t\t\t\t\t\t1\t\t1\n" +
	"holyshield\t6\t\t\t\t\t\t\t\t\t\t\t\t\t\t\n"

func TestPanelFlagsFromTable(t *testing.T) {
	defs, err := ParseDefs([]byte(panelTable))
	if err != nil {
		t.Fatal(err)
	}

	s := withDefs(defs)

	if got := s.Panel(0); got != 0 {
		t.Fatalf("no state: %v", got)
	}

	s.Apply(0, Instance{Name: "iron"})

	if f := s.Panel(1); !f.DefenseBlue() || f.DefenseRed() || f.ResistBlue(PanelFire) {
		t.Errorf("iron: %b", f)
	}

	s.Apply(0, Instance{Name: "curse", Until: 10})

	f := s.Panel(1)
	if !f.DefenseBlue() || !f.DefenseRed() {
		t.Errorf("both columns must be readable (red wins in the panel): %b", f)
	}

	want := [4]bool{true, true, true, false} // fire, cold, light red; poison not
	for i, w := range want {
		if f.ResistRed(i) != w {
			t.Errorf("curse resist %d red = %v, want %v", i, f.ResistRed(i), w)
		}
	}

	if f := s.Panel(10); f.DefenseRed() { // the curse expired
		t.Errorf("expired curse still colours: %b", f)
	}

	s.Apply(0, Instance{Name: "fade"})

	f = s.Panel(11)
	for i := 0; i < 4; i++ {
		if !f.ResistBlue(i) {
			t.Errorf("fade resist %d not blue", i)
		}
	}

	s.Apply(0, Instance{Name: "rage"})
	s.Apply(0, Instance{Name: "weak"})

	f = s.Panel(11)
	if !f.DamageBlue() || !f.DamageRed() || !f.AttackBlue() || !f.AttackRed() {
		t.Errorf("damage / attack columns: %b", f)
	}

	if f.ResistBlue(4) || f.ResistRed(-1) {
		t.Error("out of range index")
	}

	if s.HolyShield(0) {
		t.Error("holy shield not applied")
	}

	s.Apply(0, Instance{Name: StateHolyShield})

	if !s.HolyShield(0) {
		t.Error("holy shield applied")
	}
}

func TestPanelFlagsFromColumns(t *testing.T) {
	var c [14]bool
	c[0], c[9], c[13] = true, true, true

	f := PanelFlagsFromColumns(c)
	if !f.DefenseBlue() || !f.ResistRed(PanelPoison) || !f.AttackRed() || f.DefenseRed() || f.DamageBlue() {
		t.Errorf("columns mapping: %b", f)
	}
}

// TestRealPanelRows pins the shipped states.txt rows whose columns the character screen reads (VERIFIED against the
// exe masks 0x118..0x140).
func TestRealPanelRows(t *testing.T) {
	d := realDefs(t)

	for _, n := range []string{"frozenarmor", "chillingarmor", "shout", "defiance", "shiverarmor", "battlecry", "bear", "cloak_of_shadows"} {
		if !d[n].Panel.DefenseBlue() {
			t.Errorf("%s: defense not blue", n)
		}
	}

	for _, n := range []string{"innersight", "convicted", "defense_curse", "cloaked"} {
		if !d[n].Panel.DefenseRed() {
			t.Errorf("%s: defense not red", n)
		}
	}

	if f := d["resistall"].Panel; !f.ResistBlue(PanelFire) || !f.ResistBlue(PanelCold) || !f.ResistBlue(PanelLight) || f.ResistBlue(PanelPoison) {
		t.Errorf("resistall: %b", f)
	}

	if f := d["resistfire"].Panel; !f.ResistBlue(PanelFire) || f.ResistBlue(PanelCold) {
		t.Errorf("resistfire: %b", f)
	}

	if f := d["lowerresist"].Panel; !f.ResistRed(PanelFire) || !f.ResistRed(PanelCold) || !f.ResistRed(PanelLight) || !f.ResistRed(PanelPoison) {
		t.Errorf("lowerresist: %b", f)
	}

	if f := d["convicted"].Panel; f.ResistRed(PanelPoison) || !f.ResistRed(PanelLight) {
		t.Errorf("convicted: %b", f)
	}

	if d[StateHolyShield].ID != 0x65 {
		t.Errorf("holyshield id %d, want 0x65", d[StateHolyShield].ID)
	}
}
