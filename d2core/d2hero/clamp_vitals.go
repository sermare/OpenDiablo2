package d2hero

// ClampVitalsToMax lowers the current stamina, mana and life to their maximum, in that order, like
// STATS_ClampVitalsToMax (0x6278e0, VERIFIED order: stat 10, then 8, then 6). The game runs it whenever a
// stat list is removed or a base value falls (taking off a Life item, a buff ending), so the current values
// never exceed the new maxima. Life is never below zero here (death is decided elsewhere).
func (st *HeroStatsState) ClampVitalsToMax() {
	if st.Stamina > float64(st.MaxStamina) {
		st.Stamina = float64(st.MaxStamina)
	}

	if st.Mana > st.MaxMana {
		st.Mana = st.MaxMana
	}

	if st.Health > st.MaxHealth {
		st.Health = st.MaxHealth
	}
}
