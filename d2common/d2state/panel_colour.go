package d2state

// Character screen colour flags of states.txt. VERIFIED in Game.exe 1.14b: UI_DrawCharacterStatsPanel (0x4a43f0)
// colours the defense and the four resistances by asking whether the player has a state of a given mask. Each
// STATE_UnitHasStateInMask<off> (0x63b650 .. 0x63b770) loads the pointer at [0x740d78 + off] and ANDs it with the
// unit's state bits (STATE_TestMaskOverlap). The mask array of DATATBL_LoadStatesTable (0x617c50) number n sits at
// off = 0xcc + 4*n, and n is the bit index the loader registers for the column:
//
//	off   n   column      off   n   column
//	0x118 19  armblue     0x130 25  armred
//	0x11c 20  rfblue      0x134 26  rfred
//	0x120 21  rcblue      0x138 27  rcred
//	0x124 22  rlblue      0x13c 28  rlred
//	0x128 23  rpblue      0x140 29  rpred
//
// (The same rule gives curse = 0xf8, plrstaydeath = 0x100 and udead = 0x150 that other code already relies on.)
// The damage and attack rating boxes of the hand skills use the columns dmgblue (0xe8) / dmgred (0xec) and
// attblue (0xf0) / attred (0xf4) the same way (SKILLDESC_ComputeWeaponDamageRange 0x4e5860 and
// SKILLDESC_ComputeAttackRatingBonus 0x4e62b0: blue = font colour 3, red = 1, red wins). stambarblue (n 24) belongs
// to the stamina bar.

// PanelFlags is the set of character screen colour columns of one or more states.
type PanelFlags uint16

// Panel colour flags.
const (
	PanelArmBlue PanelFlags = 1 << iota
	PanelArmRed
	PanelFireBlue
	PanelColdBlue
	PanelLightBlue
	PanelPoisonBlue
	PanelFireRed
	PanelColdRed
	PanelLightRed
	PanelPoisonRed
	PanelDmgBlue // dmgblue, mask 0xe8 (n 7)
	PanelDmgRed  // dmgred, mask 0xec (n 8)
	PanelAttBlue // attblue, mask 0xf0 (n 9)
	PanelAttRed  // attred, mask 0xf4 (n 10)
)

// Resist indices of the panel, the order of d2statlist.Totals.Resist.
const (
	PanelFire = iota
	PanelCold
	PanelLight
	PanelPoison
)

// StateHolyShield is states.txt row 0x65 (101), the state the defense colour rule tests together with a shield.
const StateHolyShield = "holyshield"

// ResistBlue and ResistRed report whether the flags colour resistance i (PanelFire..PanelPoison).
func (f PanelFlags) ResistBlue(i int) bool { return i >= 0 && i < 4 && f&(PanelFireBlue<<uint(i)) != 0 }

// ResistRed is the red twin of ResistBlue.
func (f PanelFlags) ResistRed(i int) bool { return i >= 0 && i < 4 && f&(PanelFireRed<<uint(i)) != 0 }

// DefenseBlue and DefenseRed report the armblue / armred columns.
func (f PanelFlags) DefenseBlue() bool { return f&PanelArmBlue != 0 }

// DefenseRed is the red twin of DefenseBlue.
func (f PanelFlags) DefenseRed() bool { return f&PanelArmRed != 0 }

// Panel is the union of the colour flags of the unit's active states.
func (s *Set) Panel(frame int) PanelFlags {
	var out PanelFlags

	for n, in := range s.states {
		if d, ok := s.defs[n]; ok && in.Active(frame) {
			out |= d.Panel
		}
	}

	return out
}

// HolyShield reports whether state 0x65 is on (the shield clause of the defense colour).
func (s *Set) HolyShield(frame int) bool { return s.Active(frame, StateHolyShield) }

// DamageBlue and DamageRed report dmgblue / dmgred, the colour of the damage box.
func (f PanelFlags) DamageBlue() bool { return f&PanelDmgBlue != 0 }

// DamageRed is the red twin of DamageBlue.
func (f PanelFlags) DamageRed() bool { return f&PanelDmgRed != 0 }

// AttackBlue and AttackRed report attblue / attred, the colour of the attack rating box.
func (f PanelFlags) AttackBlue() bool { return f&PanelAttBlue != 0 }

// AttackRed is the red twin of AttackBlue.
func (f PanelFlags) AttackRed() bool { return f&PanelAttRed != 0 }
