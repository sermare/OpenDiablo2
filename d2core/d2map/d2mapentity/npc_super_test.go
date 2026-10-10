package d2mapentity

import "testing"

// A DS1 placement of a super unique carries its SuperUniques.txt key to the monster director, which spawns the boss
// with its name (the Siege on Harrogath quest asks for "Shenk the Overseer").
func TestNPCSuperUniqueKey(t *testing.T) {
	var n NPC

	if n.SuperUnique() != "" {
		t.Fatalf("a plain NPC has the key %q", n.SuperUnique())
	}

	n.SetSuperUnique("Shenk")

	if n.SuperUnique() != "Shenk" {
		t.Errorf("key = %q, want Shenk", n.SuperUnique())
	}
}
