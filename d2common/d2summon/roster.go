package d2summon

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
)

// Pet is one living minion of an owner.
type Pet struct {
	ID   uint32
	Type string // skills.txt pettype: the group the limit applies to
	Key  string // monstats id
	Kind string // minion, trap, totem, wall
	// Born is the frame of creation, Expires the frame the minion vanishes
	// (0 = it lasts until it dies).
	Born, Expires int
}

// Unlimited is the pettype of summons that are not counted ("none": bone
// walls and prisons, which carry their own piece count).
const Unlimited = "none"

// Plan is what the engine must do to carry out a SummonOrder.
type Plan struct {
	// Spawn is the number of minions to create now.
	Spawn int
	// Evict lists the minions that must be removed first (oldest first).
	Evict []uint32
}

// Roster tracks the minions of one owner and enforces the pettype limits.
type Roster struct {
	pets []Pet
}

// Pets returns a copy of the living minions, oldest first.
func (r *Roster) Pets() []Pet { return append([]Pet(nil), r.pets...) }

// Count is the number of living minions of a pettype.
func (r *Roster) Count(petType string) int {
	n := 0

	for _, p := range r.pets {
		if sameType(p.Type, petType) {
			n++
		}
	}

	return n
}

// Total is the number of living minions.
func (r *Roster) Total() int { return len(r.pets) }

func sameType(a, b string) bool { return strings.EqualFold(a, b) }

func counted(petType string) bool {
	return petType != "" && !strings.EqualFold(petType, Unlimited)
}

// Plan decides how many minions a cast creates and which existing ones make
// room. skills.txt rules (VERIFIED): petmax is the number of live minions of
// one pettype, shared by every skill of that pettype (all four golems share
// "golem", all assassin sentries "assassintrap", Plague Poppy / Vines / Cycle
// of Life share "vine", the three totems "totem", the shadows
// "shadowwarrior").
//
// Two cases (the second is UNVERIFIED in the exe, but matches the play
// experience of golems, shadows and sentries):
//   - a multi-summon (Count > 1, the Raven) tops the group up to Max and
//     creates nothing when it is already full;
//   - a single summon at the limit removes the oldest minion of the type.
func (r *Roster) Plan(o *d2skill.SummonOrder) Plan {
	want := o.Count
	if want < 1 {
		want = 1
	}

	if !counted(o.PetType) {
		return Plan{Spawn: want}
	}

	limit := o.Max
	if limit < 1 {
		limit = 1
	}

	alive := r.idsOf(o.PetType)

	if want > 1 {
		free := limit - len(alive)
		if free < 0 {
			free = 0
		}

		if want > free {
			want = free
		}

		return Plan{Spawn: want}
	}

	var p Plan

	p.Spawn = 1

	// the limit may be below the live count (respec, lost item bonus): trim
	// down to limit-1 so that the new minion fits
	for len(alive) > limit-1 {
		p.Evict = append(p.Evict, alive[0])
		alive = alive[1:]
	}

	return p
}

func (r *Roster) idsOf(petType string) []uint32 {
	var ids []uint32

	for _, p := range r.pets {
		if sameType(p.Type, petType) {
			ids = append(ids, p.ID)
		}
	}

	return ids
}

// Apply records a carried-out plan: the evicted minions leave, the new ones
// (ids supplied by the engine) join. frame is the current frame.
func (r *Roster) Apply(o *d2skill.SummonOrder, p Plan, newIDs []uint32, frame int) {
	for _, id := range p.Evict {
		r.Remove(id)
	}

	for _, id := range newIDs {
		pet := Pet{ID: id, Type: o.PetType, Key: o.Key, Kind: o.Kind, Born: frame}
		if o.Frames > 0 {
			pet.Expires = frame + o.Frames
		}

		r.pets = append(r.pets, pet)
	}
}

// Remove drops a minion (it died or was dismissed) and reports whether it was
// known.
func (r *Roster) Remove(id uint32) bool {
	for i, p := range r.pets {
		if p.ID == id {
			r.pets = append(r.pets[:i], r.pets[i+1:]...)

			return true
		}
	}

	return false
}

// Expire removes the minions whose time ran out and returns their ids.
func (r *Roster) Expire(frame int) []uint32 {
	var gone []uint32

	kept := r.pets[:0]

	for _, p := range r.pets {
		if p.Expires != 0 && frame >= p.Expires {
			gone = append(gone, p.ID)

			continue
		}

		kept = append(kept, p)
	}

	r.pets = kept

	return gone
}

// Clear removes every minion (the owner died, left the game or changed
// area without them) and returns their ids.
func (r *Roster) Clear() []uint32 {
	ids := make([]uint32, len(r.pets))
	for i, p := range r.pets {
		ids[i] = p.ID
	}

	r.pets = nil

	return ids
}
