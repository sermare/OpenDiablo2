package d2autoscript

import "testing"

func TestParsePadSteps(t *testing.T) {
	tests := []struct {
		spec    string
		arg     string
		wantErr bool
	}{
		{"pad:connect", "connect", false},
		{"pad:press=A", "press=A", false},
		{"pad:hold=DUP", "hold=DUP", false},
		{"pad:stick=left,1,0", "stick=left,1,0", false},
		{"pad:disconnect", "disconnect", false},
		{"pad", "", true},
		{"pad:", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			steps, err := Parse(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("want an error, got %+v", steps)
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}

			if s := steps[0]; s.Kind != KindPad || s.Arg != tt.arg {
				t.Fatalf("step = kind %q arg %q", s.Kind, s.Arg)
			}
		})
	}
}
