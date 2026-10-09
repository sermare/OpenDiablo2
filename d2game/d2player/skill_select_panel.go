package d2player

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

const (
	skillIconWidth    = 48
	screenWidth       = 800
	screenHeight      = 600
	skillIconHeight   = 48
	rightPanelEndX    = 720
	leftPanelStartX   = 90
	skillPanelOffsetY = 465
	skillListsLength  = 5 // 0 to 4. 0 - General Skills, 1 to 3 - Class-specific skills(based on the 3 different skill trees), 4 - Other skills
)

// tooltipGap is the space between the popup's tooltip and its top row.
const tooltipGap = 6

// NewHeroSkillsPanel creates the skill popup of the left or the right button.
func NewHeroSkillsPanel(asset *d2asset.AssetManager,
	ui *d2ui.UIManager,
	hero *d2mapentity.Player,
	l d2util.LogLevel,
	isLeftPanel bool) *SkillPanel {
	hoverTooltip := ui.NewTooltip(d2resource.Font16, d2resource.PaletteStatic, d2ui.TooltipXLeft, d2ui.TooltipYTop)

	skillPanel := &SkillPanel{
		asset:        asset,
		ui:           ui,
		isLeftPanel:  isLeftPanel,
		hero:         hero,
		hoverTooltip: hoverTooltip,
		sprites:      map[string]*d2ui.Sprite{},
	}

	skillPanel.levelLabel = ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
	skillPanel.keyLabel = ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)

	skillPanel.Logger = d2util.NewLogger()
	skillPanel.Logger.SetLevel(l)
	skillPanel.Logger.SetPrefix(logPrefix)

	return skillPanel
}

// SkillPanel is the skill popup shown when the player clicks the left or right skill button: a
// grid of the skills that button can use, laid out by skilldesc.txt page, row and column.
type SkillPanel struct {
	asset        *d2asset.AssetManager
	hero         *d2mapentity.Player
	ui           *d2ui.UIManager
	cells        []popupCell
	sprites      map[string]*d2ui.Sprite
	hovered      *popupCell
	hoverTooltip *d2ui.Tooltip
	levelLabel   *d2ui.Label
	keyLabel     *d2ui.Label
	isOpen       bool
	isLeftPanel  bool

	// keyName returns the name of the key a skill's hotkey slot is bound to ("F1"), or "".
	keyName func(slot int) string
	// onSelect is called when the player picks a skill.
	onSelect func(left bool, id int)

	*d2util.Logger
}

// Open opens the popup and lays the skills out again.
func (s *SkillPanel) Open() {
	s.isOpen = true
	s.RegenerateImageCache()
}

// Close the popup.
func (s *SkillPanel) Close() {
	s.isOpen = false
	s.hovered = nil
}

// IsOpen returns true if the popup is open.
func (s *SkillPanel) IsOpen() bool { return s.isOpen }

// Toggle opens or closes the popup.
func (s *SkillPanel) Toggle() {
	if s.isOpen {
		s.Close()
	} else {
		s.Open()
	}
}

// RegenerateImageCache lays the popup out again (the hero learned a skill).
func (s *SkillPanel) RegenerateImageCache() {
	s.cells = layoutSkillPopup(s.hero.Skills, s.isLeftPanel)
	s.hovered = nil
}

// IsInRect returns whether the X Y coordinates are on an icon of the popup.
func (s *SkillPanel) IsInRect(x, y int) bool { return s.isOpen && cellAt(s.cells, x, y) != nil }

// Cells returns the laid out icons (for tests and the autotest log).
func (s *SkillPanel) Cells() []popupCell { return s.cells }

// Hovered returns the skill under the mouse, or nil.
func (s *SkillPanel) Hovered() *d2hero.HeroSkill {
	if !s.isOpen || s.hovered == nil {
		return nil
	}

	return s.hovered.Skill
}

// IsLeft says whether this is the left button's popup.
func (s *SkillPanel) IsLeft() bool { return s.isLeftPanel }

// cellOfSkill finds the cell of a skill.
func (s *SkillPanel) cellOfSkill(id int) *popupCell {
	for i := range s.cells {
		if s.cells[i].Skill.ID == id {
			return &s.cells[i]
		}
	}

	return nil
}

func (s *SkillPanel) sprite(class string) *d2ui.Sprite {
	path := getSkillResourceByClass(class)
	if sp, ok := s.sprites[path]; ok {
		return sp
	}

	sp, err := s.ui.NewSprite(path, d2resource.PaletteSky)
	if err != nil {
		s.Error(err.Error())
	}

	s.sprites[path] = sp

	return sp
}

// Render gets called on every frame.
func (s *SkillPanel) Render(target d2interface.Surface) error {
	if !s.isOpen {
		return nil
	}

	for i := range s.cells {
		c := &s.cells[i]

		sp := s.sprite(c.Skill.Charclass)
		if sp == nil || sp.GetFrameCount() <= c.Skill.IconCel {
			continue // non-player skills have no icon
		}

		if err := sp.SetCurrentFrame(c.Skill.IconCel); err != nil {
			return err
		}

		sp.SetPosition(c.X, c.Y+skillIconHeight)
		sp.Render(target)

		s.levelLabel.SetText(strconv.Itoa(c.Skill.SkillPoints))
		s.levelLabel.SetPosition(c.X+skillIconWidth-14, c.Y+skillIconHeight-8)
		s.levelLabel.Render(target)

		if s.keyName != nil {
			if slot := s.hero.SkillBar.HotkeyOf(c.Skill.ID); slot >= 0 {
				if name := s.keyName(slot); name != "" {
					s.keyLabel.SetText(name)
					s.keyLabel.SetPosition(c.X+2, c.Y+12)
					s.keyLabel.Render(target)
				}
			}
		}
	}

	if s.hovered != nil {
		s.hoverTooltip.Render(target)
	}

	return nil
}

// HandleClick picks the skill under the mouse, if any, and tells the owner.
// It reports whether the click was on an icon.
func (s *SkillPanel) HandleClick(x, y int) bool {
	if !s.isOpen {
		return false
	}

	c := cellAt(s.cells, x, y)
	if c == nil {
		return false
	}

	if s.onSelect != nil {
		s.onSelect(s.isLeftPanel, c.Skill.ID)
	}

	return true
}

// HandleMouseMove updates the hovered icon and its tooltip.
func (s *SkillPanel) HandleMouseMove(x, y int) bool {
	if !s.isOpen {
		return false
	}

	c := cellAt(s.cells, x, y)
	if c == nil {
		s.hovered = nil
		return false
	}

	s.hover(c)

	return true
}

// HoverSkill puts the mouse on the icon of a skill (for the autotests); it
// reports false if the popup does not show the skill.
func (s *SkillPanel) HoverSkill(id int) bool {
	c := s.cellOfSkill(id)
	if c == nil {
		return false
	}

	s.hover(c)

	return true
}

func (s *SkillPanel) hover(c *popupCell) {
	if s.hovered == c {
		return
	}

	s.hovered = c
	s.hoverTooltip.SetText(s.tooltipText(c.Skill))

	// the tooltip sits above the top row so it hides no icon
	top := c.Y

	for i := range s.cells {
		if s.cells[i].Y < top {
			top = s.cells[i].Y
		}
	}

	_, h := s.hoverTooltip.GetSize()
	s.hoverTooltip.SetPosition(c.X, top-h-tooltipGap)
}

// tooltipText is the name, the short description and the level of a skill,
// and the key it is on.
func (s *SkillPanel) tooltipText(sk *d2hero.HeroSkill) string {
	return skillTooltip(s.asset, sk, s.hero.SkillBar, s.keyName)
}

// skillTooltip builds the tooltip of a skill icon (popup and skill tree).
func skillTooltip(asset *d2asset.AssetManager, sk *d2hero.HeroSkill, bar *d2hero.SkillBar, keyName func(int) string) string {
	name := asset.TranslateString(sk.NameKey)
	if name == "" || name == sk.NameKey {
		name = sk.Skill
	}

	lines := []string{name}

	if short := asset.TranslateString(sk.ShortKey); short != "" && short != sk.ShortKey {
		lines = append(lines, short)
	}

	lines = append(lines, fmt.Sprintf("Skill Level: %d", sk.SkillPoints))

	if bar != nil && keyName != nil {
		if slot := bar.HotkeyOf(sk.ID); slot >= 0 {
			if key := keyName(slot); key != "" {
				lines = append(lines, "Hotkey: "+key)
			}
		}
	}

	return strings.Join(lines, "\n")
}

func getSkillResourceByClass(class string) string {
	switch class {
	case "bar":
		return d2resource.BarbarianSkills
	case "nec":
		return d2resource.NecromancerSkills
	case "pal":
		return d2resource.PaladinSkills
	case "ass":
		return d2resource.AssassinSkills
	case "sor":
		return d2resource.SorcererSkills
	case "ama":
		return d2resource.AmazonSkills
	case "dru":
		return d2resource.DruidSkills
	default:
		return d2resource.GenericSkills
	}
}

// RefreshHover lays the popup out again and keeps the mouse on a skill (its
// tooltip shows the hotkey just assigned).
func (s *SkillPanel) RefreshHover(id int) {
	s.RegenerateImageCache()
	s.HoverSkill(id)
}
