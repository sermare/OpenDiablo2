package d2realmclient

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
)

// applySave stores the engine's view of the hero in the saved state and
// writes it, the way the game server handles a SavePlayer packet for the other
// connection types: the realm keeps no engine heroes, so the process that owns
// the hero saves it. The waypoints are the saved state's (the realm does not
// track them yet).
func (c *Connection) applySave(sp d2netpacket.SavePlayerPacket) {
	st := c.state
	if st == nil || sp.Player == nil {
		return
	}

	st.LeftSkill = sp.Player.LeftSkill.Shallow.SkillID
	st.RightSkill = sp.Player.RightSkill.Shallow.SkillID

	if sp.Player.Skills != nil {
		st.Skills = sp.Player.Skills
	}

	if bar := sp.Player.SkillBar; bar != nil {
		bar.Left.Skill, bar.Right.Skill = st.LeftSkill, st.RightSkill
		st.SkillBar = bar
	}

	st.Stats = sp.Player.Stats
	st.Gold = sp.Player.Gold

	if m := sp.Player.Merc; m != nil {
		st.Merc = m
	}

	// the client always sends Normal; do not demote an imported Nightmare/Hell hero
	if sp.Difficulty != d2enum.DifficultyNormal {
		st.Difficulty = sp.Difficulty
	}

	st.Containers = sp.Player.Containers

	if qp := sp.Player.Progress; qp != nil {
		p := st.EnsureProgress()
		p.Quests, p.NPC = qp.Quests, qp.NPC
	}

	if sp.Player.Death != nil {
		st.Death = sp.Player.Death
		st.Hardcore = st.Hardcore || sp.Player.Hardcore
		st.EquipmentChanged()

		if sp.Player.Equipment != nil {
			st.Equipment = *sp.Player.Equipment
		}
	}

	c.factory.RecalcStats(st)

	if err := c.factory.Save(st); err != nil {
		c.Errorf("saving the hero: %s", err)
	}

	c.saveD2S()
}

// saveD2S writes a hero imported from a real .d2s back to a .d2s (next to its
// save, or in OD2_D2S_WRITEBACK).
func (c *Connection) saveD2S() {
	if len(c.state.D2SBase) == 0 {
		return
	}

	res, err := c.factory.SaveD2S(c.state)
	if res != nil {
		for _, w := range res.Warnings {
			c.Warningf("D2S export: %s", w)
		}
	}

	if err != nil {
		c.Errorf("D2S export of %s failed: %v", c.state.HeroName, err)

		return
	}

	c.Infof("D2S EXPORT path=%s", res.Path)
	c.Infof("D2S EXPORT reparse: %s", res.Summary)
}
