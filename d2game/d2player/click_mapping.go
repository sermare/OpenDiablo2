package d2player

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"

// WorldAction is what a click on the game world does.
type WorldAction int

// Click actions.
const (
	// WorldNone: the click does nothing.
	WorldNone WorldAction = iota
	// WorldMove walks to the clicked spot.
	WorldMove
	// WorldAttack walks up to the monster under the cursor and attacks it.
	WorldAttack
	// WorldCastLeft uses the left skill at the clicked spot (the hero may walk into range).
	WorldCastLeft
	// WorldStandStill uses the left skill at the spot without moving (Shift+click).
	WorldStandStill
	// WorldCastRight uses the right skill at the spot.
	WorldCastRight
)

func (a WorldAction) String() string {
	return [...]string{"none", "move", "attack", "cast-left", "stand-still", "cast-right"}[a]
}

// Skill ids of skills.txt that are the plain "walk or hit" skill: with one of
// them on the left button a click on the ground only walks, a click on a monster attacks.
const (
	skillIDAttack        = 0
	skillIDLeftHandSwing = 5
)

// IsBasicAttackSkill reports whether the skill id is the basic Attack
// (or the assassin's Left Hand Swing), which walks when clicked on the ground.
func IsBasicAttackSkill(id int) bool {
	return id == skillIDAttack || id == skillIDLeftHandSwing
}

// EffectiveButton maps the physical click to the button the game sees. On
// macOS a trackpad has one button: Control+click is the right button (as the
// system itself treats it); a two-finger click already arrives as the right
// button. Other systems pass the button through. The Control key stays down, so
// callers must not also read it as "run" for that click.
func EffectiveButton(b d2enum.MouseButton, mod d2enum.KeyMod, goos string) d2enum.MouseButton {
	if goos == "darwin" && b == d2enum.MouseButtonLeft && mod&d2enum.KeyModControl != 0 {
		return d2enum.MouseButtonRight
	}

	return b
}

// WorldClickInput describes a click on the world (the HUD and panels are handled before this).
type WorldClickInput struct {
	Button      d2enum.MouseButton // the effective button (see EffectiveButton)
	Mod         d2enum.KeyMod
	OverMonster bool // a living monster is under the cursor
	LeftSkillID int  // skills.txt id of the left skill

	// InTown is true while the hero is in town, and LeftSkillInTown whether the left skill may be used there
	// (skills.txt InTown). In town a left skill that cannot be used just walks, as in the original: a spell on
	// the left button never leaves the hero unable to walk around the camp.
	InTown          bool
	LeftSkillInTown bool
}

// ResolveWorldClick decides what a click on the world does, following the
// original's rules: left = the left skill (the plain Attack walks, or attacks a
// monster), Shift+left = use the left skill standing still, right = the right skill.
func ResolveWorldClick(in WorldClickInput) WorldAction {
	switch in.Button {
	case d2enum.MouseButtonRight:
		return WorldCastRight
	case d2enum.MouseButtonLeft:
	default:
		return WorldNone
	}

	if in.Mod&d2enum.KeyModShift != 0 {
		return WorldStandStill
	}

	if IsBasicAttackSkill(in.LeftSkillID) {
		if in.OverMonster {
			return WorldAttack
		}

		return WorldMove
	}

	if in.InTown && !in.LeftSkillInTown {
		return WorldMove
	}

	return WorldCastLeft
}
