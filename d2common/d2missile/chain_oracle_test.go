package d2missile

import (
	"encoding/json"
	"os"
	"testing"
)

type chainTarget struct{ id int }

func (c *chainTarget) ID() string       { return "t" }
func (c *chainTarget) Serial() int      { return c.id }
func (c *chainTarget) IsPlayer() bool   { return false }
func (c *chainTarget) Alive() bool      { return true }
func (c *chainTarget) Level() int       { return 1 }
func (c *chainTarget) Defense(bool) int { return 0 }

// TestOracleChainNext compares ChainNext with the real target pick callback of Game.exe.
func TestOracleChainNext(t *testing.T) {
	buf, err := os.ReadFile("testdata/chain_golden.json")
	if err != nil {
		t.Fatal(err)
	}

	var g struct {
		Cases [][]json.RawMessage `json:"cases"`
	}

	if err = json.Unmarshal(buf, &g); err != nil {
		t.Fatal(err)
	}

	bad := 0

	for _, c := range g.Cases {
		var hit uint32

		var ids []int

		var want int

		_ = json.Unmarshal(c[0], &hit)
		_ = json.Unmarshal(c[1], &ids)
		_ = json.Unmarshal(c[2], &want)

		var cands []Target
		for _, id := range ids {
			cands = append(cands, &chainTarget{id})
		}

		var hitT Target
		if hit != 0xffffffff {
			hitT = &chainTarget{int(hit)}
		}

		got := 0
		if p := ChainNext(cands, hitT); p != nil {
			got = p.(*chainTarget).id
		}

		// the exe never spawns a bolt at the unit just hit (pUnit != param_3)
		if want == int(hit) && hit != 0xffffffff {
			want = 0
		}

		if got != want {
			bad++

			if bad <= 5 {
				t.Errorf("hit=%d ids=%v: got %d want %d", int32(hit), ids, got, want)
			}
		}
	}

	if bad > 0 {
		t.Errorf("%d chain pick cases differ", bad)
	}
}
