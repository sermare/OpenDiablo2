package d2autoscript

import (
	"reflect"
	"testing"
)

func TestParseClickStep(t *testing.T) {
	for _, spec := range []string{"click:left", "click:right", "click:left+shift@400,300", "click:left+ctrl"} {
		steps, err := Parse(spec)
		if err != nil || steps[0].Kind != KindClick {
			t.Fatalf("%s: %v %+v", spec, err, steps)
		}
	}

	for _, spec := range []string{"click", "click:", "click:middle", "click:up"} {
		if _, err := Parse(spec); err == nil {
			t.Fatalf("%s: want an error", spec)
		}
	}
}

func TestParseSkillSteps(t *testing.T) {
	tests := []struct {
		spec    string
		kind    Kind
		op, arg string
		wantErr bool
	}{
		{"skill:left=Fire Bolt", KindSkill, "left", "Fire Bolt", false},
		{"skill:right=Fire Ball", KindSkill, "right", "Fire Ball", false},
		{"skill:Popup=left", KindSkill, "popup", "left", false},
		{"skill:use=right", KindSkill, "use", "right", false},
		{"skill:spend=Frost Nova", KindSkill, "spend", "Frost Nova", false},
		{"hotkey:F1=Fire Ball", KindHotkey, "F1", "Fire Ball", false},
		{"hotkey:F8=Fire Bolt@left", KindHotkey, "F8", "Fire Bolt@left", false},
		{"press:F1", KindPress, "F1", "F1", false},
		{"skill:left", "", "", "", true},
		{"skill:left=", "", "", "", true},
		{"skill:dance=now", "", "", "", true},
		{"hotkey:F1", "", "", "", true},
		{"press", "", "", "", true},
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

			if s := steps[0]; s.Kind != tt.kind || s.Op != tt.op || s.Arg != tt.arg {
				t.Fatalf("step = kind %q op %q arg %q", s.Kind, s.Op, s.Arg)
			}
		})
	}
}

type skillHost struct {
	recordHost
	calls []string
}

func (h *skillHost) Skill(op, arg string) error {
	h.calls = append(h.calls, "skill "+op+"="+arg)
	return nil
}
func (h *skillHost) Hotkey(k, s string) error {
	h.calls = append(h.calls, "hotkey "+k+"="+s)
	return nil
}
func (h *skillHost) Press(k string) error { h.calls = append(h.calls, "press "+k); return nil }

func TestSkillStepsReachTheHost(t *testing.T) {
	steps, err := Parse("skill:left=Attack;skill:right=Fire Ball;hotkey:F1=Fire Bolt;press:F1;skill:use=right")
	if err != nil {
		t.Fatal(err)
	}

	h := &skillHost{}
	r := NewRunner(steps, h)

	for i := 0; i < 10; i++ {
		r.Advance(0.1)
	}

	want := []string{"skill left=Attack", "skill right=Fire Ball", "hotkey F1=Fire Bolt", "press F1", "skill use=right"}
	if !reflect.DeepEqual(h.calls, want) {
		t.Errorf("calls = %v, want %v", h.calls, want)
	}

	if r.Failed() {
		t.Error("script failed")
	}
}

func TestSkillStepNeedsASkillHost(t *testing.T) {
	steps, _ := Parse("press:F1")
	r := NewRunner(steps, &recordHost{})

	for i := 0; i < 3; i++ {
		r.Advance(0.1)
	}

	if !r.Failed() {
		t.Error("a host without skill support must fail the step")
	}
}

func TestParseHoldStep(t *testing.T) {
	steps, err := Parse("hold:5,left@560,340")
	if err != nil || steps[0].Kind != KindHold || steps[0].Seconds != 5 || steps[0].Arg != "left@560,340" {
		t.Fatalf("hold step: %+v %v", steps, err)
	}

	for _, bad := range []string{"hold:left@1,2", "hold:0,left", "hold:3,up"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("%q parsed, want an error", bad)
		}
	}
}
