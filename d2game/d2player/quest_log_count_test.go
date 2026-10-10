package d2player

import (
	"strings"
	"testing"
)

// Pages with a "%d" (Rescue on Mount Arreat, the seals of Terror's End) must not show the raw printf token.
func TestFillCount(t *testing.T) {
	got := fillCount("qstsa5q22", "Rescue %d more Soldiers in the Frigid Highlands.")
	if strings.Contains(got, "%d") || !strings.Contains(got, "15") {
		t.Errorf("fillCount = %q", got)
	}

	if got := fillCount("qstsa5q21", "Find the Soldiers in the Frigid Highlands."); got != "Find the Soldiers in the Frigid Highlands." {
		t.Errorf("a page without a count changed: %q", got)
	}
}
