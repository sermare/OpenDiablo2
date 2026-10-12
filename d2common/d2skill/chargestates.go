package d2skill

// ChargeState is one charge-up state a unit holds, resolved to its builder skill: the inputs of the character
// screen's SKILLDESC_GetStateBasedDamageBonus (0x4e4030, wave 47). The exe reads the builder and its level from
// stats 0x15e / 0x15f on the state's list and the count from the builder's aurastat1; here the builder is found
// through the state name like the cast code does (builderOf) and its level is the unit's skill level.
type ChargeState struct {
	PrgDam         int
	Param1, Param2 int
	Level, Count   int
}

// ChargeStatesOf lists the charge-up states in stats (ItemStatCost name -> value, the hero's state stats) with a
// positive count. level gives the hero's level of a skill id (at least 1 is used for a builder the hero knows
// nothing of, as the finisher code does).
func ChargeStatesOf(reg *Registry, stats map[string]int, level func(id int) int) []ChargeState {
	var out []ChargeState

	for _, st := range chargeStats {
		n := stats[st]
		if n < 1 {
			continue
		}

		for _, id := range reg.IDs() {
			sk := reg.ByID(id)
			if sk.AuraState != st || (sk.SrvDoFunc != 34 && sk.SrvDoFunc != 35) {
				continue
			}

			lvl := 1
			if level != nil && level(id) > lvl {
				lvl = level(id)
			}

			out = append(out, ChargeState{PrgDam: sk.PrgDam, Param1: sk.Params[1], Param2: sk.Params[2], Level: lvl, Count: n})

			break
		}
	}

	return out
}
