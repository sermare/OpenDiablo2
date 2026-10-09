package d2state

import (
	"reflect"
	"testing"
)

func TestGroupReplaces(t *testing.T) {
	groups := map[string]int{"frozenarmor": 1, "shiverarmor": 1, "quickness": 2}
	s := New()
	s.SetGroups(func(n string) int { return groups[n] })

	s.Apply(0, Instance{Name: "frozenarmor"})
	s.Apply(1, Instance{Name: "quickness"})
	s.Apply(2, Instance{Name: "might"})
	s.Apply(3, Instance{Name: "shiverarmor"})

	if got := s.Names(4); !reflect.DeepEqual(got, []string{"might", "quickness", "shiverarmor"}) {
		t.Errorf("names = %v", got)
	}

	// without a lookup nothing is replaced
	u := New()
	u.Apply(0, Instance{Name: "frozenarmor"})
	u.Apply(0, Instance{Name: "shiverarmor"})

	if len(u.Names(1)) != 2 {
		t.Errorf("ungrouped states = %v", u.Names(1))
	}
}
