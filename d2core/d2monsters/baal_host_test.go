package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
)

func TestBaalMayCloneRespectsCaps(t *testing.T) {
	for live, want := range map[int]bool{0: true, 1: true, 2: false, 3: false} {
		if got := baalMayClone(live); got != want {
			t.Errorf("live %d: %v", live, got)
		}
	}
}

func TestCloseInDest(t *testing.T) {
	x, y := closeInDest(100, 100, 130, 100, 10)
	if x != 110 || y != 100 {
		t.Errorf("%d,%d", x, y)
	}

	// never past the target
	if x, y = closeInDest(100, 100, 104, 100, 50); x != 104 || y != 100 {
		t.Errorf("overshoot %d,%d", x, y)
	}

	if x, y = closeInDest(5, 5, 5, 5, 3); x != 5 || y != 5 {
		t.Errorf("same cell %d,%d", x, y)
	}
}

func TestMoveSkillAdvance(t *testing.T) {
	if moveSkillAdvance(d2monster.DoJump, 20, 7) != 14 {
		t.Error("jump closes to reach-1")
	}

	if moveSkillAdvance(d2monster.DoJump, 5, 7) != 0 {
		t.Error("jump in reach")
	}

	if moveSkillAdvance(d2monster.DoDiabRun, 40, 7) != diabRunStride {
		t.Error("run stride")
	}

	if moveSkillAdvance(1, 40, 7) != 0 {
		t.Error("plain attack must not move")
	}
}
