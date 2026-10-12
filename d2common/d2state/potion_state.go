package d2state

// EndState removes a state the way ITEM_RemoveOldStatesThenApplyTimedState (0x5bcb80, VERIFIED) does for the
// cstate1/cstate2 of an antidote or thawing potion: the state's stat list goes away, and with the poison state
// the poison damage stream ends too (Cure documents the same pairing). It reports whether something ended.
func (s *Set) EndState(frame int, name string) bool {
	ended := s.Active(frame, name)
	delete(s.states, name)

	if name != "poison" {
		return ended
	}

	kept := s.streams[:0]

	for _, st := range s.streams {
		if st.Kind == "poison" && st.Until > frame {
			ended = true

			continue
		}

		kept = append(kept, st)
	}

	s.streams = kept
	delete(s.frac, "poison")

	return ended
}

// ApplyPotionState is ITEM_ApplyTimedStateFromItemRecord (0x5bc990, VERIFIED): the state ends `frames` after
// the end it already has (a second drink while one works adds the new length to the time left), and its stats
// are the potion's own values, not summed with the old drink. Frames below 1 do nothing. It returns the frame
// the state ends at.
func (s *Set) ApplyPotionState(frame int, name string, frames int, mods []StatMod) (until int, ok bool) {
	if name == "" || frames < 1 {
		return 0, false
	}

	left := 0
	if ex := s.Get(frame, name); ex != nil && ex.Until > frame {
		left = ex.Until - frame
	}

	until = frame + left + frames
	s.Apply(frame, Instance{Name: name, Until: until, Mods: mods})

	return until, true
}
