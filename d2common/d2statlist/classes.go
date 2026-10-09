package d2statlist

import "strings"

// ClassNames are the charstats.txt class names in class id order.
var ClassNames = []string{"Amazon", "Sorceress", "Necromancer", "Paladin", "Barbarian", "Druid", "Assassin"}

// ParseClasses reads charstats.txt (columns str, dex, int, vit, stamina,
// hpadd, ToHitFactor, LifePerLevel, StaminaPerLevel, ManaPerLevel,
// LifePerVitality, StaminaPerVitality, ManaPerMagic, BlockFactor).
func ParseClasses(data []byte) (map[string]Class, error) {
	rows, col, err := readTSV(data)
	if err != nil {
		return nil, err
	}

	out := map[string]Class{}

	for _, r := range rows {
		name := tsvCell(r, col, "class")
		id := -1

		for i, n := range ClassNames {
			if strings.EqualFold(n, name) {
				id = i
			}
		}

		if id < 0 {
			continue
		}

		out[ClassNames[id]] = Class{
			ID:      id,
			InitStr: tsvInt(r, col, "str"), InitDex: tsvInt(r, col, "dex"),
			InitVit: tsvInt(r, col, "vit"), InitEne: tsvInt(r, col, "int"),
			InitStamina: tsvInt(r, col, "stamina"), HpAdd: tsvInt(r, col, "hpadd"),
			LifePerVit: tsvInt(r, col, "lifepervitality"), ManaPerEne: tsvInt(r, col, "manapermagic"),
			StaminaPerVit: tsvInt(r, col, "staminapervitality"),
			LifePerLevel:  tsvInt(r, col, "lifeperlevel"), ManaPerLevel: tsvInt(r, col, "manaperlevel"),
			StaminaPerLevel: tsvInt(r, col, "staminaperlevel"),
			ToHitFactor:     tsvInt(r, col, "tohitfactor"), BlockFactor: tsvInt(r, col, "blockfactor"),
		}
	}

	return out, nil
}
