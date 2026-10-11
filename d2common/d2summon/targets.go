package d2summon

import "errors"

// The target registry of GAME\Targets.cpp (0x5af460-0x5af750; notes in gaps-slice-G.md). A game keeps 10
// buckets: 0..7 belong to the players (one slot per player) and hold their minions and converted monsters,
// 8 and 9 are global lists for the AI-forcing curses (Attract, Confuse, ...). A unit is in at most one
// bucket; its bucket index is stored on the unit (0xb = none), which is how conversion revert, owner death
// and monster death find and clean it.

const (
	// OwnerBuckets is the number of per-player buckets.
	OwnerBuckets = 8
	// Buckets is the total number of buckets.
	Buckets = 10
	// NoBucket is the bucket index of a unit that is in none (the exe's 0xb).
	NoBucket = 0xb
	// MaxPlayers is the limit of players the registry accepts (Game +0x8c < 9 in the exe).
	MaxPlayers = 8
)

// The registry errors.
var (
	ErrNoSlot   = errors.New("d2summon: no free player slot")
	ErrBucket   = errors.New("d2summon: bad bucket")
	ErrInBucket = errors.New("d2summon: unit is already in a bucket")
	ErrNoOwner  = errors.New("d2summon: the bucket has no owner")
)

// Entry is one registered unit with the caller's tag (the skill or AI that put it there; meaning
// unverified in the exe).
type Entry struct {
	Unit uint32
	Tag  string
}

type bucket struct {
	owner   uint32
	hasHead bool
	entries []Entry // owner buckets: oldest insertion last (new entries go right after the head); global: newest first
}

// Targets is the registry.
type Targets struct {
	b      [Buckets]bucket
	of     map[uint32]int
	nplays int
}

// NewTargets returns an empty registry.
func NewTargets() *Targets { return &Targets{of: map[uint32]int{}} }

// BucketOf is the bucket a unit is in, or NoBucket.
func (t *Targets) BucketOf(unit uint32) int {
	if i, ok := t.of[unit]; ok {
		return i
	}

	return NoBucket
}

// AddPlayer gives a player the first free owner slot (TARGETS_AddUnitToTargetList) and returns it.
func (t *Targets) AddPlayer(unit uint32) (int, error) {
	if _, ok := t.of[unit]; ok {
		return 0, ErrInBucket
	}

	if t.nplays >= MaxPlayers {
		return 0, ErrNoSlot
	}

	for i := 0; i < OwnerBuckets; i++ {
		if !t.b[i].hasHead {
			t.b[i] = bucket{owner: unit, hasHead: true}
			t.of[unit] = i
			t.nplays++

			return i, nil
		}
	}

	return 0, ErrNoSlot
}

// AddEntry puts a minion or converted monster in an owner bucket, right after the head
// (TARGETS_AddEntryToList).
func (t *Targets) AddEntry(unit uint32, tag string, slot int) error {
	if slot < 0 || slot >= OwnerBuckets || !t.b[slot].hasHead {
		return ErrNoOwner
	}

	if _, ok := t.of[unit]; ok {
		return ErrInBucket
	}

	t.b[slot].entries = append([]Entry{{unit, tag}}, t.b[slot].entries...)
	t.of[unit] = slot

	return nil
}

// AddGlobal puts a monster in global bucket 8 or 9; the newest entry comes first (TARGETS_AddEntryToListB).
func (t *Targets) AddGlobal(unit uint32, tag string, slot int) error {
	if slot < OwnerBuckets || slot >= Buckets {
		return ErrBucket
	}

	if _, ok := t.of[unit]; ok {
		return ErrInBucket
	}

	t.b[slot].entries = append([]Entry{{unit, tag}}, t.b[slot].entries...)
	t.of[unit] = slot

	return nil
}

// Members lists the entries of a bucket (not the owner), newest first.
func (t *Targets) Members(slot int) []Entry {
	if slot < 0 || slot >= Buckets {
		return nil
	}

	return append([]Entry(nil), t.b[slot].entries...)
}

// Remove takes a unit out of its bucket (TARGETS_RemoveUnitFromTargetList: a monster died, or a conversion
// reverted). A player is removed together with its whole bucket. It reports whether the unit was registered.
func (t *Targets) Remove(unit uint32) bool {
	i, ok := t.of[unit]
	if !ok {
		return false
	}

	if i < OwnerBuckets && t.b[i].hasHead && t.b[i].owner == unit {
		t.FreeBucket(i)

		return true
	}

	e := t.b[i].entries
	for k := range e {
		if e[k].Unit == unit {
			t.b[i].entries = append(e[:k:k], e[k+1:]...)

			break
		}
	}

	delete(t.of, unit)

	return true
}

// FreeBucket empties a bucket (TARGETS_FreeBucketForUnit): every member goes back to NoBucket and the
// owner's head is dropped. It returns the members released (not the owner), newest first.
func (t *Targets) FreeBucket(slot int) []Entry {
	if slot < 0 || slot >= Buckets {
		return nil
	}

	b := &t.b[slot]
	out := append([]Entry(nil), b.entries...)

	for _, e := range b.entries {
		delete(t.of, e.Unit)
	}

	if b.hasHead {
		delete(t.of, b.owner)

		t.nplays--
	}

	*b = bucket{}

	return out
}

// OwnerDied releases the minions and converted monsters of a dead player and returns them; the player keeps
// no slot afterwards (the game re-adds a respawned player).
func (t *Targets) OwnerDied(owner uint32) []Entry {
	i, ok := t.of[owner]
	if !ok || i >= OwnerBuckets {
		return nil
	}

	return t.FreeBucket(i)
}

// FreeAll empties every bucket (TARGETS_FreeAllBuckets) and returns the number of units released.
func (t *Targets) FreeAll() int {
	n := len(t.of)
	for i := range t.b {
		t.b[i] = bucket{}
	}

	t.of = map[uint32]int{}
	t.nplays = 0

	return n
}
