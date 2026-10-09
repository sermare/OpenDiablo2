package d2player

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// Skill selection, hotkeys and skill points of the hero. In the original the
// client asks the server (C2S 0x3C) and only changes its selection when the
// server confirms (S2C 0x23); in single player the server is in process, so
// the same checks (d2hero.Selectable) run here directly and the result is
// persisted with the next save (HeroState.SkillBar and the skill points).

// firstHotkeyEvent and lastHotkeyEvent bound the hotkey commands (UseSkill1..16).
const (
	firstHotkeyEvent = d2enum.UseSkill1
	lastHotkeyEvent  = d2enum.UseSkill16
)

func (g *GameControls) bar() *d2hero.SkillBar { return g.hero.SyncSkillBar() }

// playClick plays the button click.
func (g *GameControls) playClick() {
	if g.clickSfx != nil {
		g.clickSfx.Play()
	}

	g.Infof("SKILLBAR click-sound %s", d2resource.SFXButtonClick)
}

// hotkeyName returns the name of the key bound to a hotkey slot ("F1"), or "".
func (g *GameControls) hotkeyName(slot int) string {
	b := g.keyMap.GetKeysForGameEvent(firstHotkeyEvent + d2enum.GameEvent(slot))
	if b == nil || b.Primary < 0 {
		return ""
	}

	return g.keyMap.KeyToString(b.Primary)
}

// slotOfKeyName maps a key name ("F1", case-insensitive) to its hotkey slot,
// or -1.
func (g *GameControls) slotOfKeyName(name string) int {
	name = strings.TrimSpace(name)

	for slot := 0; slot < d2hero.HotkeyCount; slot++ {
		if k := g.hotkeyName(slot); k != "" && strings.EqualFold(k, name) {
			return slot
		}
	}

	return -1
}

// slotOfKey maps a pressed key to its hotkey slot, or -1.
func (g *GameControls) slotOfKey(k d2enum.Key) int {
	ev := g.keyMap.getGameEvent(k)
	if ev >= firstHotkeyEvent && ev <= lastHotkeyEvent {
		return int(ev - firstHotkeyEvent)
	}

	return -1
}

func handName(left bool) string {
	if left {
		return "left"
	}

	return "right"
}

func (g *GameControls) skillName(id int) string {
	if rec := g.asset.Records.Skill.Details[id]; rec != nil {
		return rec.Skill
	}

	return "?"
}

// skillIDByName finds a skill by its skills.txt name (case-insensitive) or id.
func (g *GameControls) skillIDByName(name string) (int, error) {
	name = strings.TrimSpace(name)

	for _, rec := range g.asset.Records.Skill.Details {
		if strings.EqualFold(rec.Skill, name) {
			return rec.ID, nil
		}
	}

	if id, err := strconv.Atoi(name); err == nil && g.asset.Records.Skill.Details[id] != nil {
		return id, nil
	}

	return -1, fmt.Errorf("unknown skill %q", name)
}

// logSkillState writes the skill selection, the hotkeys and the unused points
// as one SKILLBAR line (what the autotests read).
func (g *GameControls) logSkillState(why string) {
	b := g.bar()

	var keys []string

	for slot, h := range b.Hotkeys {
		if h.Empty() {
			continue
		}

		name := g.hotkeyName(slot)
		if name == "" {
			name = "slot" + strconv.Itoa(slot)
		}

		keys = append(keys, fmt.Sprintf("%s=%s", name, g.skillName(h.Skill)))
	}

	g.Infof("SKILLBAR state (%s) left=%s(%d) right=%s(%d) swap=%s/%s hotkeys=[%s] unused_points=%d", why,
		g.skillName(b.Left.Skill), b.Left.Skill, g.skillName(b.Right.Skill), b.Right.Skill,
		g.skillName(b.LeftSwap.Skill), g.skillName(b.RightSwap.Skill), strings.Join(keys, " "), g.hero.Stats.SkillPoints)
}

// SelectSkill makes a skill the active one of the left or right button. It
// fails (and changes nothing) for a skill the hero cannot use on that button.
func (g *GameControls) SelectSkill(left bool, id int) error {
	s := g.hero.Skills[id]
	if !d2hero.Selectable(s, left) {
		err := fmt.Errorf("skill %s (%d) cannot be used on the %s button", g.skillName(id), id, handName(left))
		g.Infof("SKILLBAR select FAIL hand=%s id=%d: %v", handName(left), id, err)

		return err
	}

	if left {
		g.hero.LeftSkill = s
	} else {
		g.hero.RightSkill = s
	}

	g.bar()
	g.playClick()
	g.Infof("SKILLBAR select hand=%s skill=%q id=%d level=%d", handName(left), s.Skill, id, s.SkillPoints)
	g.logSkillState("select")
	g.saveHero()

	return nil
}

// AssignHotkey puts a skill on a hotkey slot; left says which button the key
// selects it on.
func (g *GameControls) AssignHotkey(slot, id int, left bool) error {
	s := g.hero.Skills[id]
	if !d2hero.Selectable(s, left) {
		err := fmt.Errorf("skill %s (%d) cannot be assigned to a hotkey for the %s button", g.skillName(id), id, handName(left))
		g.Infof("SKILLBAR hotkey FAIL slot=%d: %v", slot, err)

		return err
	}

	if err := g.bar().Assign(slot, id, left); err != nil {
		return err
	}

	g.Infof("SKILLBAR hotkey key=%s slot=%d skill=%q id=%d hand=%s", g.hotkeyName(slot), slot, s.Skill, id, handName(left))
	g.logSkillState("hotkey")
	g.saveHero()

	return nil
}

// PressHotkey selects the skill on a hotkey slot (on the button the skill was
// assigned for; if the hero can no longer use it there, on the other one).
func (g *GameControls) PressHotkey(slot int) error {
	slotSkill, ok := g.bar().Press(slot)
	if !ok {
		g.Infof("SKILLBAR press key=%s slot=%d empty", g.hotkeyName(slot), slot)

		return errors.New("hotkey is empty")
	}

	g.Infof("SKILLBAR press key=%s slot=%d skill=%q", g.hotkeyName(slot), slot, g.skillName(slotSkill.Skill))

	left := slotSkill.Left
	if !d2hero.Selectable(g.hero.Skills[slotSkill.Skill], left) {
		left = !left
	}

	return g.SelectSkill(left, slotSkill.Skill)
}

// onSkillKey handles a hotkey key: while the pointer is over a skill (in the
// popup or the skill tree) the key assigns that skill, otherwise it selects the
// assigned skill, as in the original.
func (g *GameControls) onSkillKey(slot int) {
	if p := g.hud.skillSelectMenu.OpenPanel(); p != nil {
		if s := p.Hovered(); s != nil {
			_ = g.AssignHotkey(slot, s.ID, p.IsLeft())
			p.RefreshHover(s.ID)

			return
		}
	}

	if s := g.skilltree.HoveredSkill(); s != nil {
		_ = g.AssignHotkey(slot, s.ID, false)

		return
	}

	_ = g.PressHotkey(slot)
}

// UseActiveSkill casts the skill on the left or right button at a position
// (tile coordinates) through the normal skill pipeline.
func (g *GameControls) UseActiveSkill(left bool, x, y float64) {
	s := g.hero.RightSkill
	if left {
		s = g.hero.LeftSkill
	}

	g.inputListener.OnPlayerCast(s.ID, x, y)
}

// SpendSkillPoint puts one unused skill point into a skill, under the
// skills.txt rules (character level, required skills, maximum level).
func (g *GameControls) SpendSkillPoint(id int) error {
	token := strings.ToLower(g.hero.Class.GetToken3())

	if err := d2hero.SpendSkillPoint(g.hero.Skills, g.hero.Stats, token, id); err != nil {
		g.Infof("SKILLBAR spend FAIL skill=%q id=%d: %v", g.skillName(id), id, err)

		return err
	}

	g.skilltree.refresh()
	g.hud.skillSelectMenu.RegenerateImageCache()
	g.setAddButtons()
	g.playClick()
	g.Infof("SKILLBAR spend skill=%q id=%d level=%d unused_points=%d", g.skillName(id), id,
		g.hero.Skills[id].SkillPoints, g.hero.Stats.SkillPoints)
	g.saveHero()

	return nil
}

// GrantLevels gives the hero level-ups: the skill and stat points they bring.
func (g *GameControls) GrantLevels(n int) {
	d2hero.GrantLevelUp(g.hero.Stats, n)
	g.hero.Stats.NextLevelExp = g.asset.Records.GetExperienceBreakpoint(g.hero.Class, g.hero.Stats.Level)
	g.skilltree.refresh()
	g.setAddButtons()
	g.Infof("SKILLBAR levelup +%d level=%d unused_points=%d stat_points=%d", n, g.hero.Stats.Level,
		g.hero.Stats.SkillPoints, g.hero.Stats.StatsPoints)
}

// onSkillPopupPick is called when the player clicks an icon of a popup.
func (g *GameControls) onSkillPopupPick(left bool, id int) {
	_ = g.SelectSkill(left, id)
}

// swapSkillSets switches the skills with the weapon set (the swap key).
func (g *GameControls) swapSkillSets() {
	b := g.bar()
	b.SwapSets()

	// the swap set may hold skills the hero cannot use (empty file, respec)
	for _, left := range []bool{true, false} {
		id := b.Active(left)
		if s := g.hero.Skills[id]; d2hero.Selectable(s, left) {
			if left {
				g.hero.LeftSkill = s
			} else {
				g.hero.RightSkill = s
			}
		} else {
			b.Select(left, 0)

			if left {
				g.hero.LeftSkill = g.hero.Skills[0]
			} else {
				g.hero.RightSkill = g.hero.Skills[0]
			}
		}
	}

	g.logSkillState("weapon swap")
}

// AutoSkill implements the skill: steps of OD2_AUTOSCRIPT. op is one of
// left, right (select), popup (left, right, close), hover, click (pick the
// icon in the open popup like a mouse click), use (cast the left/right skill)
// and spend.
func (g *GameControls) AutoSkill(op, arg string) error {
	switch op {
	case "left", "right":
		id, err := g.skillIDByName(arg)
		if err != nil {
			return err
		}

		return g.SelectSkill(op == "left", id)
	case "popup":
		return g.autoPopup(arg)
	case "hover", "click":
		return g.autoHoverClick(op, arg)
	case "use":
		pos := g.hero.Position.World()
		g.Infof("SKILLBAR use hand=%s skill=%q", arg, g.skillName(g.activeSkillID(arg == "left")))
		g.UseActiveSkill(arg == "left", pos.X()+2, pos.Y()+2)

		return nil
	case "spend":
		id, err := g.skillIDByName(arg)
		if err != nil {
			return err
		}

		return g.SpendSkillPoint(id)
	case "nospend":
		id, err := g.skillIDByName(arg)
		if err != nil {
			return err
		}

		if err := g.SpendSkillPoint(id); err == nil {
			return fmt.Errorf("spending a point in %s was allowed, expected a refusal", arg)
		}

		return nil
	}

	return fmt.Errorf("unknown skill op %q", op)
}

func (g *GameControls) activeSkillID(left bool) int {
	if left {
		return g.hero.LeftSkill.ID
	}

	return g.hero.RightSkill.ID
}

func (g *GameControls) autoPopup(which string) error {
	switch which {
	case "left", "right":
		g.clearScreen()

		if which == "left" {
			g.hud.skillSelectMenu.OpenLeftPanel()
		} else {
			g.hud.skillSelectMenu.OpenRightPanel()
		}

		p := g.hud.skillSelectMenu.OpenPanel()

		var names []string

		for _, c := range p.Cells() {
			names = append(names, fmt.Sprintf("%s@r%dc%d(lvl%d)", c.Skill.Skill, c.Row, c.Col, c.Skill.SkillPoints))
		}

		g.Infof("SKILLBAR popup hand=%s icons=%d [%s]", which, len(names), strings.Join(names, " "))
	case "close":
		g.hud.skillSelectMenu.ClosePanels()
		g.Infof("SKILLBAR popup closed")
	default:
		return fmt.Errorf("popup wants left, right or close, not %q", which)
	}

	return nil
}

func (g *GameControls) autoHoverClick(op, name string) error {
	id, err := g.skillIDByName(name)
	if err != nil {
		return err
	}

	if p := g.hud.skillSelectMenu.OpenPanel(); p != nil {
		c := p.cellOfSkill(id)
		if c == nil {
			return fmt.Errorf("popup does not show %s", name)
		}

		p.HandleMouseMove(c.X+skillIconWidth/2, c.Y+skillIconHeight/2)
		g.Infof("SKILLBAR %s popup=%s skill=%q", op, handName(p.IsLeft()), name)

		if op == "click" {
			p.HandleClick(c.X+skillIconWidth/2, c.Y+skillIconHeight/2)
			g.hud.skillSelectMenu.ClosePanels()
		}

		return nil
	}

	if op == "hover" && g.skilltree.IsOpen() {
		if !g.skilltree.HoverSkill(id) {
			return fmt.Errorf("skill tree tab does not show %s", name)
		}

		g.Infof("SKILLBAR hover skilltree skill=%q", name)

		return nil
	}

	return errors.New("no skill popup is open")
}

// AutoHotkey implements hotkey:F1=<skill>[@left].
func (g *GameControls) AutoHotkey(key, name string) error {
	left := false

	if i := strings.LastIndex(name, "@"); i >= 0 {
		left, name = strings.EqualFold(strings.TrimSpace(name[i+1:]), "left"), name[:i]
	}

	slot := g.slotOfKeyName(key)
	if slot < 0 {
		return fmt.Errorf("no hotkey is bound to %q", key)
	}

	id, err := g.skillIDByName(name)
	if err != nil {
		return err
	}

	return g.AssignHotkey(slot, id, left)
}

// AutoPress implements press:F1: the key goes through the same handler as a
// real key press (including the assignment while hovering a skill).
func (g *GameControls) AutoPress(key string) error {
	slot := g.slotOfKeyName(key)
	if slot < 0 {
		return fmt.Errorf("no hotkey is bound to %q", key)
	}

	g.onSkillKey(slot)

	return nil
}

// skillTreeClick handles a mouse click on an icon of the open skill tree: the
// left button puts an unused point into the skill, the right button selects it
// as the right skill (if the hero can use it there). It reports whether
// the click was on an icon. (UNVERIFIED: the original's exact click rules of
// the tree were not read from the binary.)
func (g *GameControls) skillTreeClick(event d2interface.MouseEvent) bool {
	si := g.skilltree.iconAt(event.X(), event.Y())
	if si == nil {
		return false
	}

	switch event.Button() {
	case d2enum.MouseButtonLeft:
		_ = g.SpendSkillPoint(si.skill.ID)
	case d2enum.MouseButtonRight:
		_ = g.SelectSkill(false, si.skill.ID)
	}

	return true
}

func (g *GameControls) commandLevelUp(term d2interface.Terminal) func(args []string) error {
	return func(args []string) error {
		n, err := strconv.Atoi(args[0])
		if err != nil || n < 1 {
			term.Errorf("invalid argument")
			return nil
		}

		g.GrantLevels(n)

		return nil
	}
}
