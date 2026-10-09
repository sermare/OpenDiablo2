package d2gamescreen

import "testing"

func TestLevelStore(t *testing.T) {
	var s levelStore

	if s.take(9) != nil {
		t.Fatal("empty store returned a level")
	}

	s.put(9, &savedLevel{visits: 1, opened: map[savedObject]bool{{5, 1, 2}: true}})
	s.put(10, &savedLevel{visits: 1})

	if l := s.take(9); l == nil || !l.opened[savedObject{5, 1, 2}] || l.opened[savedObject{5, 2, 1}] {
		t.Fatalf("level 9 not stored as put: %+v", l)
	}

	// leaving a level again replaces its state and keeps the visit count
	s.take(9).visits = 3
	s.put(9, &savedLevel{visits: 1})

	if l := s.take(9); l.visits != 3 || len(l.opened) != 0 {
		t.Fatalf("second save: visits %d opened %d", l.visits, len(l.opened))
	}

	if s.take(11) != nil {
		t.Fatal("unvisited level has state")
	}
}
