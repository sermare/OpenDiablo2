package diablo2item

import (
	"strconv"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// ExtraMod is a property that a Horadric Cube recipe attached to an item (the
// crafted mods of a recipe, the level requirement of an upgrade): a Properties.txt
// code with the value the recipe rolled.
type ExtraMod struct {
	Code  string `json:"code"`
	Param string `json:"param,omitempty"`
	Min   int    `json:"min,omitempty"`
	Max   int    `json:"max,omitempty"`
	Value int    `json:"value"`
}

// generateCubeProperties turns the recipe mods into properties. The value was
// rolled by the cube, so the property gets min = max = value.
func (i *Item) generateCubeProperties() []*Property {
	var out []*Property

	for _, m := range i.CubeMods {
		prop := i.factory.NewProperty(m.Code, i.paramValue(m.Param), m.Value, m.Value)
		if prop != nil {
			out = append(out, prop)
		}
	}

	return out
}

// paramValue resolves a property parameter: a number, or the name of a skill.
func (i *Item) paramValue(param string) int {
	if param == "" {
		return 0
	}

	if n, err := strconv.Atoi(param); err == nil {
		return n
	}

	for id, sk := range i.factory.asset.Records.Skill.Details {
		if sk.Skill == param {
			return id
		}
	}

	return 0
}

// socketKind picks the Gems.txt column group for the item the gem sits in:
// weapons use the weapon mods, shields the shield mods, all other armor the
// helm mods (the "helm" group of Gems.txt covers every armor piece).
func (i *Item) socketKind() string {
	rec := i.CommonRecord()
	if rec == nil {
		return "helm"
	}

	switch {
	case rec.Source == d2enum.InventoryItemTypeWeapon:
		return "weapon"
	case rec.Type == "shld" || rec.Type == "ashd" || rec.Type == "head" || rec.Type2 == "shld":
		return "shield"
	}

	return "helm"
}

// generateSocketedProperties adds the properties of the gems, runes and jewels
// in the sockets. A runeword item takes its properties from the runeword
// instead (generateRunewordProperties).
func (i *Item) generateSocketedProperties() []*Property {
	if i.Runeword != "" || len(i.SocketCodes) == 0 {
		return nil
	}

	var out []*Property

	kind := i.socketKind()

	for _, code := range i.SocketCodes {
		g := i.gemRecord(code)
		if g == nil {
			continue
		}

		for _, m := range gemMods(g, kind) {
			if m.code == "" {
				continue
			}

			if prop := i.factory.NewProperty(m.code, m.param, m.min, m.max); prop != nil {
				out = append(out, prop)
			}
		}
	}

	return out
}

type gemMod struct {
	code            string
	param, min, max int
}

func gemMods(g *d2records.GemRecord, kind string) []gemMod {
	switch kind {
	case "weapon":
		return []gemMod{
			{g.WeaponMod1Code, g.WeaponMod1Param, g.WeaponMod1Min, g.WeaponMod1Max},
			{g.WeaponMod2Code, g.WeaponMod2Param, g.WeaponMod2Min, g.WeaponMod2Max},
			{g.WeaponMod3Code, g.WeaponMod3Param, g.WeaponMod3Min, g.WeaponMod3Max},
		}
	case "shield":
		return []gemMod{
			{g.ShieldMod1Code, g.ShieldMod1Param, g.ShieldMod1Min, g.ShieldMod1Max},
			{g.ShieldMod2Code, g.ShieldMod2Param, g.ShieldMod2Min, g.ShieldMod2Max},
			{g.ShieldMod3Code, g.ShieldMod3Param, g.ShieldMod3Min, g.ShieldMod3Max},
		}
	}

	return []gemMod{
		{g.HelmMod1Code, g.HelmMod1Param, g.HelmMod1Min, g.HelmMod1Max},
		{g.HelmMod2Code, g.HelmMod2Param, g.HelmMod2Min, g.HelmMod2Max},
		{g.HelmMod3Code, g.HelmMod3Param, g.HelmMod3Min, g.HelmMod3Max},
	}
}

func (i *Item) gemRecord(code string) *d2records.GemRecord {
	for _, g := range i.factory.asset.Records.Item.Gems {
		if g.Code == code {
			return g
		}
	}

	return nil
}

// RunewordRecord returns the Runes.txt row of the item's runeword (found by
// its display name), or nil.
func (i *Item) RunewordRecord() *d2records.RuneRecord {
	if i.Runeword == "" {
		return nil
	}

	for _, r := range i.factory.asset.Records.Item.Runewords {
		if r.RuneName == i.Runeword {
			return r
		}
	}

	return nil
}

func (i *Item) generateRunewordProperties() []*Property {
	rw := i.RunewordRecord()
	if rw == nil {
		return nil
	}

	var out []*Property

	for _, p := range rw.Properties {
		if p.Code == "" {
			continue
		}

		prop := i.factory.NewProperty(p.Code, i.paramValue(p.Parameter), p.Min, p.Max)
		if prop != nil {
			out = append(out, prop)
		}
	}

	return out
}
