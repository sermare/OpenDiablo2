package d2automap

// Store keeps the revealed map of every level the hero has visited in this
// session, so that a level shows what was explored when the hero comes back
// (the original keeps one cell record per level; the saved game does not keep
// it, the fork keeps it in memory only: see docs/automap.md).
type Store struct {
	models map[int]*Model
}

// NewStore returns an empty store.
func NewStore() *Store { return &Store{models: map[int]*Model{}} }

// Level returns the model of a level, creating it on first use.
func (s *Store) Level(id int) *Model {
	m := s.models[id]
	if m == nil {
		m = NewModel()
		s.models[id] = m
	}

	return m
}

// Has reports whether a level has been visited (has a model).
func (s *Store) Has(id int) bool { _, ok := s.models[id]; return ok }

// Levels returns how many levels have a model.
func (s *Store) Levels() int { return len(s.models) }

// Forget drops the model of one level (a level that is generated anew).
func (s *Store) Forget(id int) { delete(s.models, id) }
