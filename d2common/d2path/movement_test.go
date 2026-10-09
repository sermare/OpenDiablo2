package d2path

import (
	"bufio"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestSpeeds(t *testing.T) {
	tests := []struct {
		name string
		mode MoveMode
		pct  int
		vel  int
		perS float64
	}{
		{"walk", ModeWalk, 100, 6 << 8, 9.375},
		{"run", ModeRun, 100, 9 << 8, 14.0625},
		{"run +50%", ModeRun, 150, (9 << 8) * 150 / 100, 21.09375},
		{"run slowed to floor", ModeRun, 25, (9 << 8) / 4, 3.515625},
		{"leap", ModeLeap, 40, LeapVelocity, 25},
		{"teleport", ModeTeleport, 100, 0, 0},
	}

	for _, tc := range tests {
		v := Velocity(PlayerBase(ClassSorceress), tc.mode, tc.pct)
		if v != tc.vel {
			t.Errorf("%s: vel %d want %d", tc.name, v, tc.vel)
		}

		if got := SubtilesPerSecond(v); math.Abs(got-tc.perS) > 1e-9 {
			t.Errorf("%s: %v subtiles/s want %v", tc.name, got, tc.perS)
		}
	}
}

func TestSpeedPercentFloor(t *testing.T) {
	for stat, want := range map[int]int{0: 100, 30: 130, -50: 50, -75: 25, -200: 25} {
		if got := SpeedPercent(stat); got != want {
			t.Errorf("stat %d: %d want %d", stat, got, want)
		}
	}
}

func TestAllClassesBase(t *testing.T) {
	for c := ClassAmazon; c <= ClassAssassin; c++ {
		if b := PlayerBase(c); b.Walk != 6 || b.Run != 9 {
			t.Errorf("class %d: %+v", c, b)
		}
	}

	if (PlayerBase(99) != BaseVelocity{}) {
		t.Error("unknown class must have no velocity")
	}
}

func TestAdvanceDiagonal(t *testing.T) {
	// Direction (0.7071, 0.7071) in 4.12 is 2896.
	x, y := Advance(0, 0, 9<<8, 2896, 2896)
	if x != y || x <= 0 {
		t.Fatalf("diagonal step %d,%d", x, y)
	}

	if f := float64(x) / 65536; math.Abs(f-0.5625*0.7071) > 0.001 {
		t.Errorf("diag step %v", f)
	}
}

func TestAccelerate(t *testing.T) {
	vel, c := 0, 0
	for i := 0; i < 5; i++ {
		vel, c = Accelerate(vel, 300, 100, c)
	}

	if vel != 100 || c != 0 {
		t.Fatalf("after 5 frames vel=%d c=%d", vel, c)
	}

	for i := 0; i < 20; i++ {
		vel, c = Accelerate(vel, 300, 100, c)
	}

	if vel != 300 {
		t.Errorf("cap: %d", vel)
	}
}

func TestTeleportBlockedDest(t *testing.T) {
	g := NewCellGrid(0, 0, 10, 10)
	g.Set(5, 5, FlagWalk)

	p, ok := TeleportTo(g, MaskPlayer, Point{5, 5}, 3)
	if !ok || p == (Point{5, 5}) || abs(p.X-5) > 1 || abs(p.Y-5) > 1 {
		t.Fatalf("got %v %v", p, ok)
	}

	if p, _ := TeleportTo(g, MaskPlayer, Point{2, 2}, 3); p != (Point{2, 2}) {
		t.Errorf("free dest moved to %v", p)
	}
}

// Real-data check: monstats Velocity/Run columns give sane tick speeds.
func TestMonstatsVelocities(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	f, err := os.Open(filepath.Join(dir, "monsters", "d2data", "monstats.txt"))
	if err != nil {
		t.Skip(err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	sc.Scan()

	vi, ri := -1, -1

	for i, h := range strings.Split(sc.Text(), "\t") {
		switch h {
		case "Velocity":
			vi = i
		case "Run":
			ri = i
		}
	}

	if vi < 0 || ri < 0 {
		t.Fatal("columns missing")
	}

	n := 0

	for sc.Scan() {
		c := strings.Split(sc.Text(), "\t")
		if len(c) <= ri {
			continue
		}

		w, _ := strconv.Atoi(c[vi])
		r, _ := strconv.Atoi(c[ri])
		b := BaseVelocity{w, r}

		for _, m := range []MoveMode{ModeWalk, ModeRun} {
			if s := SubtilesPerSecond(Velocity(b, m, 100)); s < 0 || s > 40 {
				t.Errorf("%s mode %d: %v", c[0], m, s)
			}
		}

		n++
	}

	if n < 100 {
		t.Errorf("only %d rows", n)
	}
}
