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
	// stat 0x43 carries the 100% base; frw is diminished with k=150.
	for _, c := range []struct{ stat, frw, want int }{
		{100, 0, 100}, {130, 0, 130}, {50, 0, 50}, {25, 0, 25}, {-100, 0, 25},
		{100, 150, 175}, // 150*150/300 = 75
		{100, 30, 125},  // 150*30/180 = 25
		{50, 30, 75},    // cold slow offsets FRW
	} {
		if got := SpeedPercent(c.stat, c.frw); got != c.want {
			t.Errorf("stat %d frw %d: %d want %d", c.stat, c.frw, got, c.want)
		}
	}

	if Step(100, 0, 1<<12) != Step(100, StepScale, 1<<12) || Step(100, -5, 1<<12) != Step(100, StepScale, 1<<12) {
		t.Error("scale below 1 must default to 0x400")
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
	vel, acc, c := 0, 100, 0
	for i := 0; i < 5; i++ {
		vel, acc, c = Accelerate(vel, 300, acc, c)
	}

	if vel != 100 || c != 0 || acc != 100 {
		t.Fatalf("after 5 frames vel=%d acc=%d c=%d", vel, acc, c)
	}

	for i := 0; i < 20; i++ {
		vel, acc, c = Accelerate(vel, 300, acc, c)
	}

	if vel != 300 || acc != 0 {
		t.Errorf("cap: vel %d acc %d (accel is zeroed at the cap)", vel, acc)
	}

	// zero accel: counter untouched
	if v, a, cc := Accelerate(50, 300, 0, 3); v != 50 || a != 0 || cc != 3 {
		t.Errorf("zero accel: %d %d %d", v, a, cc)
	}

	// deceleration clamps at zero and keeps its accel
	vel, acc, c = 20, -50, 0
	for i := 0; i < 5; i++ {
		vel, acc, c = Accelerate(vel, 300, acc, c)
	}

	if vel != 0 || acc != -50 || c != 0 {
		t.Errorf("decel clamp: %d %d %d", vel, acc, c)
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
