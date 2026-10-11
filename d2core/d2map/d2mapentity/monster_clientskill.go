package d2mapentity

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"

// Animation driven client skills of a monster (cltdofunc 59 Diablo Run, 62
// Mosquito, 64 Queen death; see d2monster/clientskill.go). The exe calls the
// function whenever the animation frame changes; the frame edits are applied to
// the composite. Not modelled: the exe's total-frames edits (the animation rate
// stays), and the run's velocity change, which the Director's closeIn covers.

type clientScript struct {
	do      int
	params  [7]int
	stage   int
	counter int
	last    int
}

// SetClientSkill binds a client do-function to the monster's current animation.
// params is skills.txt Param1..6 at indexes 1..6; counter is the skill's
// TargetType slot (Mosquito bites left, Queen death passes).
func (m *Monster) SetClientSkill(do int, params [7]int, counter int) {
	switch do {
	case d2monster.CltDoDiabRun, d2monster.CltDoMosquito, d2monster.CltDoQueenDeath:
		m.cskill = &clientScript{do: do, params: params, counter: counter, last: -1}
	default:
		m.cskill = nil
	}
}

// ClearClientSkill drops the bound client skill.
func (m *Monster) ClearClientSkill() { m.cskill = nil }

// ClientSkillCounter is the script's counter slot (0 when none is bound).
func (m *Monster) ClientSkillCounter() int {
	if m.cskill == nil {
		return 0
	}

	return m.cskill.counter
}

// runClientSkill runs the bound script once per animation frame change.
func (m *Monster) runClientSkill() {
	s := m.cskill
	if s == nil {
		return
	}

	frame := m.composite.GetCurrentFrame()
	if frame == s.last {
		return
	}

	s.last = frame

	var e d2monster.FrameEdit

	near := func(d int) bool { return m.ClientNear != nil && m.ClientNear(d) }

	switch s.do {
	case d2monster.CltDoDiabRun:
		e, s.stage = d2monster.DiabRunClient(s.params, frame, s.stage, near(1))
	case d2monster.CltDoMosquito:
		e, s.counter = d2monster.MosquitoClient(s.params[1], s.counter, m.ClientNear != nil, near(1))
		if !e.Handled {
			m.cskill = nil
		}
	case d2monster.CltDoQueenDeath:
		if m.mode != d2monster.ModeDying {
			return
		}

		e, s.counter = d2monster.QueenDeathClient(frame, s.counter)
	}

	if e.SetSeq && e.Seq >= 0 && e.Seq < m.composite.GetFrameCount() {
		m.composite.SetCurrentFrame(e.Seq)
		s.last = e.Seq
	}

	if e.ResetMode {
		m.cskill = nil

		if !m.SetMode(d2monster.ModeDead) {
			m.mode = d2monster.ModeDead
		}
	}
}
