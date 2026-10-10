package d2skills

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// TestHurtPlayerScale feeds rolled damage structs (8.8 fixed point) through
// hurtPlayer the way Meteor's impact, its burning ground ticks and the shards of
// Blizzard arrive, and checks what the defender is sent.
func TestHurtPlayerScale(t *testing.T) {
	tests := []struct {
		name         string
		d            d2combat.Damage
		n            int
		wantSent     int // OnPvPHit calls
		wantTotal    int // whole points sent in all
		wantFirstRaw int
	}{
		{"Meteor impact (fire 2450)", d2combat.Damage{Fire: 2450 << 8}, 1, 1, 416, 2450},
		{"Blizzard shard (cold)", d2combat.Damage{Cold: 1605 << 8}, 1, 1, 272, 1605},
		// meteorfire at L20: (15+92)<<3 .. (25+92)<<3 = 856..936, every tick is 0.57..0.62 life at 17 percent
		{"one burning ground tick is only carried", d2combat.Damage{Fire: 856}, 1, 0, 0, 0},
		{"100 burning ground ticks", d2combat.Damage{Fire: 896}, 100, 59, 59, 4},
		{"no damage", d2combat.Damage{}, 3, 0, 0, 0},
	}

	for _, tt := range tests {
		e := newTestEngine()
		e.Logger = d2util.NewLogger()

		src, dst := &d2mapentity.Player{}, &d2mapentity.Player{}

		sent, total, firstRaw := 0, 0, 0

		e.OnPvPHit = func(h PvPHit) {
			if sent == 0 {
				firstRaw = h.Raw
			}

			sent++
			total += h.Parts.Total()
		}

		for i := 0; i < tt.n; i++ {
			d := tt.d
			e.hurtPlayer(dst, src, &d, "Meteor")
		}

		if total != tt.wantTotal || sent != tt.wantSent || firstRaw != tt.wantFirstRaw {
			t.Errorf("%s: sent %d hits, %d points, first raw %d; want %d, %d, %d", tt.name, sent, total, firstRaw,
				tt.wantSent, tt.wantTotal, tt.wantFirstRaw)
		}
	}
}
