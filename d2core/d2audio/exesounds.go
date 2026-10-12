package d2audio

// Sound rules read from Game.exe 1.14b (wave 29). Sounds.txt ids are the row
// ordinals (the Index column equals the row number in the shipped table).
// Addresses are for the notes in w29-sound2.md.

// Fixed Sounds.txt ids the client plays without a unit.
const (
	SndCursorPass        = 1   // cursor_pass: game menu selection moved (UI_HandleGameMenuClick 0x479460)
	SndButtonClick       = 4   // cursor_button_click: mini panel, inventory, stat/skill button, ui button press
	SndTabSwitch         = 6   // cursor_switch: skill tree tab change (UI_HandleSkillTreeMouseDown 0x4a8188)
	SndItemFlippy        = 216 // item_flippy: item hits the ground (ITEM_PlayGroundDropSound 0x4be190), frame 0
	SndItemPickup        = 235 // item_pickup: any item enters the cursor, shift-click to belt (ITEM_InitClientItem 0x4be1f0)
	DefaultDropSfxFrames = 12  // delay of the drop sound after the flippy when no dropsfxframe is given
)

// ItemSoundRow is the sound columns of one items table row (armor, weapons,
// misc), a uniqueitems row or a setitems row, as Sounds.txt ids (0 = none).
type ItemSoundRow struct {
	Drop, Use, DropFrame int
}

// ItemSounds picks the sounds of an item the way ITEM_GetDropSoundId 0x4be630,
// ITEM_GetDropSfxFrame 0x4be0d0 and ITEM_GetDropSoundAndType 0x4bdfb0 do:
// the base row gives the drop sound when it is above 0 and the use sound only
// then; a unique (quality 7) or set (quality 5) row replaces both, and the
// frame, when its own drop sound is above 0 (its use sound may then be 0).
// special is nil for other qualities.
func ItemSounds(base ItemSoundRow, special *ItemSoundRow) (drop, use, frame int) {
	frame = DefaultDropSfxFrames

	if base.Drop > 0 {
		drop, use = base.Drop, base.Use

		if base.DropFrame != 0 {
			frame = base.DropFrame
		}
	}

	if special != nil && special.Drop > 0 {
		drop, use = special.Drop, special.Use

		if special.DropFrame != 0 {
			frame = special.DropFrame
		}
	}

	return drop, use, frame
}

// GroundDrop is one sound of an item landing on the ground.
type GroundDrop struct {
	ID    int
	Delay int // game frames after the item enters the ground mode
}

// GroundDropPlan is what ITEM_PlayGroundDropSound plays: the flippy sound at
// once (no unit) and the item's drop sound after frame frames (a non-zero
// third argument of the request, VERIFIED w39: SOUND_PlayRequest 0x4b6350 stores counter+arg as the start;
// the counter DAT_007b4a44 (read by 0x47d690) is incremented once per SOUND_FrameUpdate 0x47eb70, which the idle frame 0x44a6d0 calls, so its unit is idle/render frames, w44 VERIFIED).
func GroundDropPlan(drop, frame int) []GroundDrop {
	out := []GroundDrop{{ID: SndItemFlippy}}

	if drop > 0 {
		out = append(out, GroundDrop{ID: drop, Delay: frame})
	}

	return out
}

// ImpactSounds is UNITSND_LookupSoundForUnitMode 0x4c7080: the two impact
// layers played when a unit starts its death or get-hit mode, from the
// monstats2 HitClass byte (low nibble = body layer, high nibble = overlay).
// unitType is 0 player, 1 monster, other for objects; extra is the unexplained
// third argument (UNVERIFIED: treated as 0 for monsters). Both results are
// Sounds.txt ids, 0 = none.
func ImpactSounds(hitClass, unitType, extra int) (body, layer int) {
	switch hitClass & 0xf {
	case 1:
		body = 0x141
	case 2, 3, 4, 5:
		body = 0x155
	case 6, 7:
		body = 0x14f
	case 8, 9:
		body = 0x149
	case 10, 11:
		body = 0x15b
	case 12:
		body = 0x145
	}

	notPlayerOrMon := unitType != 0 && unitType != 1

	switch hitClass & 0xf0 {
	case 0x10:
		if body != 0 {
			layer = 0x17b
		}
	case 0x20:
		layer = 0x161
		if body != 0 {
			layer = 0x164
		}
	case 0x30:
		if body == 0 {
			layer = 0x167
		} else if notPlayerOrMon || extra != 0 {
			layer = 0x16a
		}
	case 0x40:
		layer = 0x170
		if body != 0 {
			layer = 0x174
		}
	case 0x50:
		if body == 0 || notPlayerOrMon || extra != 0 {
			layer = 0x178
		}
	case 0x60:
		layer = 0x17e
	case 0x70:
		layer = 0x18b
	case 0x80:
		layer = 0x956
	case 0xb0:
		layer = 0x187
	}

	return body, layer
}

// MonsterSoundSlots are the byte offsets of the MonSounds record the exe reads
// (UNITSND_PlayUnitModeStartSound 0x4c8e90, UNITSND_PlayMonsterSpawnSound
// 0x4c8e00, CLIENT_PlaySoundById 0x4c8890): Skill1..4 at 0x44..0x50, Init at
// 0x70, Taunt at 0x74, Flee at 0x78 (the idle/flee vocal, throttled).
const (
	MonSndInitOffset  = 0x70
	MonSndTauntOffset = 0x74
	MonSndFleeOffset  = 0x78
)

// MonsterInitSound says whether the Init vocal plays for a freshly created
// client monster: the row has an Init sound, the monster is alive and the
// flag in the unit (field 0x80) is not set yet.
func MonsterInitSound(hasInit, dead, alreadyPlayed bool) bool {
	return hasInit && !dead && !alreadyPlayed
}

// FleeThrottle is UNITSND_PlayMonsterIdleSoundThrottled 0x4c8400: the flee/idle
// vocal plays only when more than 3 ticks passed since the last unit sound
// (global), and starts 6 + random frames late.
func FleeThrottle(ticksSinceLast int) bool { return ticksSinceLast > 3 }
