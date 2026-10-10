package d2gamescreen

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

func TestLootRetryUnpicked(t *testing.T) {
	cases := []struct {
		name          string
		quest, ground bool
		tries         int
		want          bool
	}{
		{"quest item left on the ground", true, true, 1, true},
		{"picked up", true, false, 1, false},
		{"junk is not retried", false, true, 1, false},
		{"out of tries", true, true, maxQuestPickTries, false},
	}

	for _, c := range cases {
		it := &d2mapentity.Item{}
		l := &lootState{tried: map[*d2mapentity.Item]bool{it: true}, tries: map[*d2mapentity.Item]int{it: c.tries}, last: it}

		if got := l.retryUnpicked(c.quest, c.ground); got != c.want || l.tried[it] == c.want {
			t.Errorf("%s: retry=%v tried=%v, want retry=%v", c.name, got, l.tried[it], c.want)
		}

		if l.last != nil {
			t.Errorf("%s: last not cleared", c.name)
		}
	}
}
