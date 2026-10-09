package d2hero

// CharacterKind is how the character select tells the game modes apart.
type CharacterKind int

// The kinds of character shown in the character select.
const (
	KindSoftcore CharacterKind = iota
	KindHardcore
	KindDeadHardcore
)

// Kind classifies a hero. Only a hardcore hero can be dead for good: a softcore
// hero with the died flag (it died and was saved before the respawn) is a
// normal, playable character.
func (h *HeroState) Kind() CharacterKind {
	switch {
	case h.IsDeadHardcore():
		return KindDeadHardcore
	case h != nil && h.Hardcore:
		return KindHardcore
	default:
		return KindSoftcore
	}
}

// PlayRefusal returns why a character cannot be started, or "" when it can.
// In the original a hardcore character that died is permanently dead: it stays
// in the list (flagged) but cannot be loaded for play.
func (h *HeroState) PlayRefusal() string {
	if h.IsDeadHardcore() {
		return "hardcore character " + h.HeroName + " is dead and cannot be played"
	}

	return ""
}
