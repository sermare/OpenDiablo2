package d2monster

// FaithfulW44 lists the handlers wave 44 newly counts after a line-by-line comparison with the decompile of the exe
// (every roll, scratch field, skill slot and parameter index checked): Andariel 0x5f4960, Smith 0x5e2770,
// Duriel 0x5f5880 (the first-tick Holy Freeze start is hosted now: AuraStarter / RightSkillChecker in
// d2monsters/moncurse.go), FetishBlowgun 0x5e00b0 (the strafe flag argument of 0x5de5e0 is not modelled, as for
// the other callers) and Minion 0x5e09c0 (a dead or missing commanded unit pops the command: the Director's
// UnitTarget answers not-found for dead units, which closes the gap noted by w38).
// Still open: Izual 0x5f7b30 and Summoner 0x5f7720 (logic equal, but the quest hooks 0x5b1fb0 / 0x599f70 have no
// host: QuestBossHook).
func FaithfulW44() []string {
	return []string{"Andariel", "Smith", "Duriel", "FetishBlowgun", "Minion"}
}
