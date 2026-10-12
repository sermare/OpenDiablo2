package d2monster

import "testing"

// Wave 45: the table rows of the Uber bosses (read from Game.exe 0x739c08): UberDiablo and UberBaal carry the
// second think of Diablo and BaalCrab, UberMephisto and UberIzual carry none.
func TestUberSecondThinks(t *testing.T) {
	for name, want := range map[string]bool{"UberDiablo": true, "UberBaal": true, "UberMephisto": false, "UberIzual": false} {
		if got := HasThink2(name); got != want {
			t.Errorf("HasThink2(%s) = %v, want %v", name, got, want)
		}
	}
}
