package d2inventory

import "strings"

// Describe lists the filled equipment slots as "slot=Item Name", for logs and
// the autotest PANEL lines. RightArm holds the gloves of an imported save.
func (e *CharacterEquipment) Describe() string {
	var parts []string

	if e == nil {
		return ""
	}

	if e.Head != nil {
		parts = append(parts, "head="+e.Head.ItemName)
	}

	if e.Torso != nil {
		parts = append(parts, "torso="+e.Torso.ItemName)
	}

	if e.Legs != nil {
		parts = append(parts, "legs="+e.Legs.ItemName)
	}

	if e.RightArm != nil {
		parts = append(parts, "gloves="+e.RightArm.ItemName)
	}

	if e.LeftArm != nil {
		parts = append(parts, "leftArm="+e.LeftArm.ItemName)
	}

	if e.RightHand != nil {
		parts = append(parts, "rightHand="+e.RightHand.ItemName)
	}

	if e.LeftHand != nil {
		parts = append(parts, "leftHand="+e.LeftHand.ItemName)
	}

	if e.Shield != nil {
		parts = append(parts, "shield="+e.Shield.ItemName)
	}

	return strings.Join(parts, ", ")
}
