package d2monsters

import "testing"

// Dopplezon and Valkyrie are registered in the monsters' target list (TARGETS_AddEntryToList in
// SRVDO_015/016); other summons are not.
func TestHostileTargetsAggroSummonsOnly(t *testing.T) {
	hostile := petUnit(1, 0, 0, "Melee", "")
	plain := petUnit(2, 1, 0, "NecroPet", "minion")
	decoy := petUnit(3, 4, 0, "NecroPet", "minion")
	decoy.ally.opt.DrawsAggro = true

	d := testDirector(hostile, plain, decoy)

	tg, _, ok := d.Nearest(hostile.b)
	if !ok || tg.ID != unitTargetBase+3 {
		t.Fatalf("target %+v ok=%v, want the aggro summon (the nearer plain minion is ignored)", tg, ok)
	}

	decoy.ally.opt.DrawsAggro = false

	if _, _, ok := d.Nearest(hostile.b); ok {
		t.Error("a plain summon drew aggro")
	}
}
