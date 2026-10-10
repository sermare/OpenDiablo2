package d2audio

import (
	"strings"
	"testing"
)

func TestCategory(t *testing.T) {
	tests := []struct {
		handle      string
		music, loop bool
		want        string
	}{
		{"music_town_1", true, true, CatMusic},
		{"scene_wilderness_day", false, true, CatAmbience},
		{"event_wild_day_1", false, false, CatEvent},
		{"medium_walk_dirt_3", false, false, CatFootstep},
		{"weapon_1hs_small_1", false, false, CatSwing},
		{"cursor_button_click", false, false, CatUI},
		{"sorceress_firebolt_impact_2", false, false, CatHit},
		{"sorceress_firebolt_2", false, true, CatSkill},
		{"sorceress_cast_fire", false, false, CatSkill},
		{"zombie_attack_5", false, false, CatMonster},
		{"akara_greeting_time_2", false, false, CatSpeech},
		{"something_else", false, false, CatOther},
	}

	for _, tt := range tests {
		if got := Category(tt.handle, tt.music, tt.loop); got != tt.want {
			t.Errorf("Category(%q) = %s, want %s", tt.handle, got, tt.want)
		}
	}
}

func TestAudioStatsString(t *testing.T) {
	a := AudioStats{
		Started:  map[string]int{"music": 1, "footstep": 3},
		Peak:     map[string]float64{"music": 0.129, "footstep": 0.5},
		Flow:     map[string]int64{"music": 4096},
		FlowPeak: map[string]int{"music": 900},
		Music:    0.3, Sfx: 1,
	}

	got := a.String()

	for _, want := range []string{"music_master=0.30", "sfx_master=1.00", "footstep=3(vol 0.500 flow 0B amp 0)",
		"music=1(vol 0.129 flow 4096B amp 900)"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q missing %q", got, want)
		}
	}
}
