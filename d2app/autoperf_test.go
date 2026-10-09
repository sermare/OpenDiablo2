package d2app

import (
	"math"
	"testing"
)

func TestPerfStats(t *testing.T) {
	hundred := make([]float64, 100)
	for i := range hundred {
		hundred[i] = float64(100 - i) // unsorted on purpose: 100, 99, ... 1
	}

	tests := []struct {
		name              string
		in                []float64
		avg, p95, p99, mx float64
	}{
		{"empty", nil, 0, 0, 0, 0},
		{"single", []float64{2.5}, 2.5, 2.5, 2.5, 2.5},
		{"hundred", hundred, 50.5, 95, 99, 100},
		{"spike", []float64{1, 1, 1, 1, 21}, 5, 1, 1, 21},
	}

	for _, tt := range tests {
		avg, p95, p99, mx := perfStats(tt.in)
		if math.Abs(avg-tt.avg) > 1e-9 || p95 != tt.p95 || p99 != tt.p99 || mx != tt.mx {
			t.Errorf("%s: got avg=%v p95=%v p99=%v max=%v, want %v %v %v %v",
				tt.name, avg, p95, p99, mx, tt.avg, tt.p95, tt.p99, tt.mx)
		}
	}
}

func TestPerfStatsKeepsInput(t *testing.T) {
	in := []float64{3, 1, 2}
	perfStats(in)

	if in[0] != 3 || in[1] != 1 || in[2] != 2 {
		t.Errorf("perfStats sorted its input: %v", in)
	}
}
