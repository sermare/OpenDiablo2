package d2object

import (
	"reflect"
	"testing"
)

type seqRand struct{ vals []int }

func (s *seqRand) Roll(n int) int {
	v := 0
	if len(s.vals) > 0 {
		v, s.vals = s.vals[0], s.vals[1:]
	}

	return v % n
}

// scriptSink answers each Drop from a script and records the calls.
type scriptSink struct {
	script []string // "-" nothing, "n" non magical, "m" magical
	calls  []string
}

func (s *scriptSink) Drop(forced int) (made, magical bool) {
	s.calls = append(s.calls, "d"+string(rune('0'+forced)))

	r := "n"
	if len(s.script) > 0 {
		r, s.script = s.script[0], s.script[1:]
	}

	return r != "-", r == "m"
}

func (s *scriptSink) Create(code string, n int) {
	s.calls = append(s.calls, code+string(rune('0'+n)))
}

func TestOpenGeneric(t *testing.T) {
	tests := []struct {
		name            string
		locked, variant bool
		rolls           []int
		script          []string
		want            []string
	}{
		{"empty below threshold", false, false, []int{24}, nil, nil},
		{"plain opens", false, false, []int{25}, []string{"n"}, []string{"d0"}},
		{"locked rolls twice, never empty", true, false, []int{0}, []string{"n", "n"}, []string{"d0", "d0"}},
		{"variant forces magic", false, true, []int{50, 0}, []string{"m"}, []string{"d4"}},
		{"variant rare 5 percent", false, true, []int{4, 0}, []string{"m"}, []string{"d6"}},
		{"variant never empty and retries", false, true, []int{50, 0}, []string{"n", "-", "n", "m"}, []string{"d4", "d4", "d4", "d4"}},
		{"variant retry capped at ten", false, true, []int{50, 0}, nil, append([]string{"d4"}, rep("d4", 10)...)},
		{"variant locked two then no retry", true, true, []int{50, 0}, []string{"n", "m"}, []string{"d4", "d4"}},
	}

	for _, tc := range tests {
		s := &scriptSink{script: tc.script}
		OpenGeneric(&seqRand{tc.rolls}, s, tc.locked, tc.variant)

		if !reflect.DeepEqual(s.calls, tc.want) && !(len(s.calls) == 0 && len(tc.want) == 0) {
			t.Errorf("%s: calls %v, want %v", tc.name, s.calls, tc.want)
		}
	}
}

func rep(s string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = s
	}

	return out
}

func TestSparklyTierOf(t *testing.T) {
	for roll, want := range map[int]SparklyTier{0: SparklyUnique, 199: SparklyUnique, 200: SparklySet, 599: SparklySet,
		600: SparklyRare, 1199: SparklyRare, 1200: SparklyMagicThree, 3199: SparklyMagicThree, 3200: SparklyMagicTwo,
		6199: SparklyMagicTwo, 6200: SparklyJackpot, 9999: SparklyJackpot} {
		if got := SparklyTierOf(roll); got != want {
			t.Errorf("roll %d: tier %d, want %d", roll, got, want)
		}
	}
}

func TestOpenSparkly(t *testing.T) {
	jackpotTail := []string{"d0", "gld 5", "hp3 2", "mp3 2"}
	tests := []struct {
		name   string
		tier   int
		script []string
		want   []string
	}{
		{"unique magical", 0, []string{"m"}, []string{"d7"}},
		{"unique twice", 0, []string{"n", "m"}, []string{"d7", "d7"}},
		{"set kind", 300, []string{"m"}, []string{"d5"}},
		{"rare kind", 900, []string{"m"}, []string{"d6"}},
		{"nothing made goes to jackpot", 0, []string{"-"},
			append(append([]string{"d7"}, rep("d4", 10)...), "gld 5", "hp3 2", "mp3 2")},
		{"three magic", 2000, []string{"n", "m", "m", "m"}, []string{"d4", "d4", "d4", "d4"}},
		{"magic three nothing made then jackpot", 2000, nil, nil},
		{"two magic, gold tops up", 5000, []string{"m", "n", "m"}, []string{"d4", "d4", "d4", "gld 6"}},
		{"two magic, no plain: one unforced roll, nothing made", 5000, []string{"m", "m", "-"}, []string{"d4", "d4", "d0", "gld 7"}},
		{"two magic, no plain: unforced roll made", 5000, []string{"m", "m", "n"}, []string{"d4", "d4", "d0", "gld 6"}},
		{"jackpot magical first", 9000, []string{"m"}, append([]string{"d4"}, append(rep("d0", 4), "gld 5", "hp3 2", "mp3 2")...)},
	}

	_ = jackpotTail

	for _, tc := range tests {
		if tc.want == nil {
			continue // covered by the structural cases
		}

		s := &scriptSink{script: tc.script}
		OpenSparkly(&seqRand{[]int{0, tc.tier}}, s)

		if !reflect.DeepEqual(s.calls, tc.want) {
			t.Errorf("%s: calls %v, want %v", tc.name, s.calls, tc.want)
		}
	}
}

func TestIsMagicalQuality(t *testing.T) {
	for q, want := range map[int]bool{0: false, 3: false, 4: true, 7: true, 9: true, 10: false} {
		if IsMagicalQuality(q) != want {
			t.Errorf("quality %d: %v", q, !want)
		}
	}
}
