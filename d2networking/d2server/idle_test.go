package d2server

import (
	"testing"
	"time"
)

func TestIdleTrackerStamp(t *testing.T) {
	clock := time.Unix(100, 0)
	tr := newIdleTracker(func() time.Time { return clock })

	if _, ok := tr.Last("a"); ok {
		t.Fatal("no stamp yet")
	}

	tr.Stamp("a")

	clock = clock.Add(5 * time.Second)
	tr.Stamp("b")

	if v, _ := tr.Last("a"); !v.Equal(time.Unix(100, 0)) {
		t.Fatalf("a: %v", v)
	}

	tr.Stamp("a")

	if v, _ := tr.Last("a"); !v.Equal(time.Unix(105, 0)) {
		t.Fatalf("restamp: %v", v)
	}

	tr.Forget("a")

	if _, ok := tr.Last("a"); ok {
		t.Fatal("forgotten")
	}
}

func TestIdleExpired(t *testing.T) {
	now := time.Unix(1000, 0)
	ago := func(d time.Duration) time.Time { return now.Add(-d) }

	tests := []struct {
		name    string
		last    time.Time
		slow    bool
		backlog int
		marked  bool
		want    bool
	}{
		{"fresh", ago(time.Second), false, 0, false, false},
		{"45 s exactly", ago(45 * time.Second), false, 0, false, false},
		{"45 s and a bit", ago(45*time.Second + time.Millisecond), false, 0, false, true},
		{"slow, short, backlog", ago(11 * time.Second), true, 11, false, true},
		{"slow, short, backlog 10", ago(11 * time.Second), true, 10, false, false},
		{"not slow", ago(30 * time.Second), false, 99, false, false},
		{"marked", ago(time.Second), false, 0, true, true},
		{"stamp in the future", now.Add(time.Hour), true, 99, true, false},
		{"stamp now", now, true, 99, true, false},
	}

	for _, tc := range tests {
		if got := IdleExpired(now, tc.last, tc.slow, tc.backlog, tc.marked); got != tc.want {
			t.Errorf("%s: %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestSaveChunkOK(t *testing.T) {
	tests := []struct {
		n    int
		want bool
	}{{0, true}, {4096, true}, {0x1fff, true}, {0x2000, false}, {0x10000, false}, {-1, false}}
	for _, tc := range tests {
		if SaveChunkOK(tc.n) != tc.want {
			t.Errorf("%#x", tc.n)
		}
	}

	if !SaveChunkOK(chunkSize) {
		t.Fatal("the server's own chunk size must fit")
	}
}
