package d2player

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2gui"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

const ( // for the dc6 frames
	statsPanelTopLeft = iota
	statsPanelTopRight
	statsPanelBottomLeft
	statsPanelBottomRight
)

const (
	statsPanelOffsetX, statsPanelOffsetY = 80, 60 // art top: Mode800.PanelTop()
)

const (
	labelLevelX, labelLevelY = 112, 97

	labelHeroNameX, labelHeroNameY   = 165, 72
	labelHeroClassX, labelHeroClassY = 330, 74

	labelExperienceX, labelExperienceY = 202, 97
	labelNextLevelX, labelNextLevelY   = 330, 97

	// The captions are centred in the boxes of the label table 0x723a5c (x 10..73 for the four attributes, 174..228
	// for stamina, life and mana, 174..268 for defense, 190..268 for the resistances, plus the panel offset
	// (80, 60)); their y is the bottom of the line. scripts/verify.d/ui-layout.golden pins the resulting anchors.
	labelStrengthX, labelStrengthY   = 122, 150
	labelDexterityX, labelDexterityY = 122, 213
	labelVitalityX, labelVitalityY   = 122, 298
	labelEnergyX, labelEnergyY       = 122, 360

	labelDefenseX, labelDefenseY = 301, 260
	labelStaminaX, labelStaminaY = 281, 298
	labelLifeX, labelLifeY       = 281, 322
	labelManaX, labelManaY       = 281, 360

	// the two-line resistance captions: UI_DrawCharacterStatsPanel 0x4a43f0 splits a caption at its newline and draws the
	// first line at the table's y - 4 and the second at y + 4 (the lines stand 8 apart around the table's y; VERIFIED
	// in the disassembly: "y + -4" before the first draw, "y + 4" before the second)
	labelResFireLine1X, labelResFireLine1Y = 310, 395
	labelResFireLine2X, labelResFireLine2Y = 310, 403
	// the original lists the resistances fire, cold, lightning, poison from top to bottom (string ids 4071 to 4074 at
	// y 346, 370, 395, 419 of the panel, 0x723a5c; stats 39, 43, 41, 45 at 0x723b70)
	labelResColdLine1X, labelResColdLine1Y   = 310, 419
	labelResColdLine2X, labelResColdLine2Y   = 310, 427
	labelResLightLine1X, labelResLightLine1Y = 310, 444
	labelResLightLine2X, labelResLightLine2Y = 310, 452
	labelResPoisLine1X, labelResPoisLine1Y   = 310, 468
	labelResPoisLine2X, labelResPoisLine2Y   = 310, 476
)

const (
	heroStatsCloseButtonX, heroStatsCloseButtonY = 208, 448 // 32x32 standing at y 480: UI_DrawCharacterStatsPanel 0x4a43f0
	addStatSocketOffsetX, addStatSocketOffsetY   = -3, 34
)

const (
	newStatsRemainingPointsFieldX, newStatsRemainingPointsFieldY = 83, 430
	newStatsRemainingPointsLabelX                                = 92
	newStatsRemainingPointsLabel1Y                               = 411
	newStatsRemainingPointsLabel2Y                               = 418
	newStatsRemainingPointsValueX, newStatsRemainingPointsValueY = 188, 411
)

// PanelText represents text on the panel
type PanelText struct {
	X           int
	Y           int
	Text        string
	Font        string
	AlignCenter bool
}

// StatsPanelLabels represents the labels in the status panel
type StatsPanelLabels struct {
	Level        *d2ui.Label
	Experience   *d2ui.Label
	NextLevelExp *d2ui.Label
	Strength     *d2ui.Label
	Dexterity    *d2ui.Label
	Vitality     *d2ui.Label
	Energy       *d2ui.Label
	Health       *d2ui.Label
	MaxHealth    *d2ui.Label
	Mana         *d2ui.Label
	MaxMana      *d2ui.Label
	MaxStamina   *d2ui.Label
	Stamina      *d2ui.Label

	// derived from the equipment (d2statlist totals)
	Defense *d2ui.Label
	Resist  [4]*d2ui.Label // fire, cold, lightning, poison
}

// Positions of the derived value labels: the defense and resistance numbers stand in the boxes of the value table
// 0x723b70 (x 273..307, centre 290 + 80 = 370; defense y 209, resistances y 348, 372, 396, 420, plus 60), the
// anchors are pinned by scripts/verify.d/ui-layout.golden (VERIFIED against the table, w44).
const (
	labelDefenseValueX, labelDefenseValueY = 370, 258 // the box right of the caption: x 273..307 of the panel (0x723b70)
	labelResValueX                         = 370
)

// resistValueY are the y positions of the fire, cold, lightning and poison values.
var resistValueY = [4]int{400, 424, 448, 472}

// The four boxes right of Strength and Dexterity show the description of the hand skills
// (SKILLDESC_DrawSkillDescription 0x4ea490 called for rows 0 and 6 of the table at 0x72bff8 by 0x4ea940, VERIFIED): the
// left hand skill in the upper box of each pair, the right hand skill in the lower one. A damage box carries the skill
// name (row 0, bottom y 93) over the caption "Damage" (string 0xfdd, bottom y 101) and the value in its right part
// (x 263..307, y 98); an attack rating box the caption (string 0xfdf, x 162..270) and the value (x 270..310, y 160).
// The row triples are {x1, y, x2}, six rows per hand (the right hand's start at row 6, 24 px lower). The damage and
// attack rating of a skill come from a per skill function of skilldesc.txt (descdam / descatt); this port prints the
// weapon's damage and the character's attack rating for skills that attack with the weapon and leaves other skills
// with the name only. NOT PORTED (read in SKILLDESC_DrawSkillDescription 0x4ea490): the exe shows the Damage line when
// the skill is usable (SKILL_GetSkillUsabilityStatus in {0,1,6,8}, via CLIENT_GetPlayerCastAnimMode 0x4d6d40, which
// is a usability code and not an animation mode) and the skilldesc descdam id has a function in the table at 0x72bf20
// (ids 1..24 and 37..41, 29 functions at 0x4e61xx..0x4e8exx), and the Attack Rating block when the descatt id (table
// 0x72bfb0, 0x48 entries) has one. Those functions include spells, so spell damage lines exist in the exe; this port
// has only the weapon and thrown skills (Anim attack/throw), a known gap. Anchors: the x is centred in x1..x2 (UI_DrawTextLine
// 0x4a3750, w44) and the y handed to D2WIN_DrawText is the table y, a baseline; the row table at 0x72bff8
// ({x1,y,x2}, 12 bytes) reads 93/101/98 for name/caption/value, rows 3,4,5 = 160/163/160 for the two attack rating caption
// lines and the value. Which of rows 3/4 each caption line takes was not recoverable (register arguments).
const (
	handBoxNameX1, handBoxNameX2     = 162, 258
	handDamageValueX1, handDamageX2  = 263, 307
	handARCaptionX1, handARCaptionX2 = 162, 270
	handARValueX1, handARValueX2     = 270, 310

	handNameY, handCaptionY, handDamageValueY = 93, 101, 98
	handARY                                   = 160
	handRowShift                              = 24 // the right hand's rows are 24 px lower (y 117/125/122, AR 184)
	panelColumnX                              = 80 // table x to panel x (the art's left edge)
	panelTextY                                = 53 // table y to the label y of the Go panel
)

// HandSkillInfo describes a hand skill for the character sheet.
type HandSkillInfo struct {
	Name   string // translated skill name (empty: no skill)
	Weapon bool   // the skill attacks with the weapon: damage and attack rating are listed
	// Per skill numbers of the boxes (0x4e5860, 0x4e62b0; wave 46). SrcDam is skills.txt srcdam (0: unknown, the
	// whole weapon range); ToHitBonus is SKILL_GetToHitBonus at the skill's level (percent added to the attack
	// rating pool); Bit8 and StateBonus are the state based damage bonus of a skill with flag bit 8 (the blue
	// colour); SkillMin/SkillMax are the skill's own physical damage in whole points.
	SrcDam, ToHitBonus, StateBonus int
	Bit8, Thrown                   bool
	SkillMin, SkillMax             int
}

// handLabels are the labels of one hand's pair of boxes.
type handLabels struct {
	name, damageCaption, damage, arCaption, arCaption2, ar *d2ui.Label
}

// NewHeroStatsPanel creates a new hero status panel
func NewHeroStatsPanel(asset *d2asset.AssetManager,
	ui *d2ui.UIManager,
	heroName string,
	heroClass d2enum.Hero,
	l d2util.LogLevel,
	heroState *d2hero.HeroStatsState) *HeroStatsPanel {
	originX := 0
	originY := 0

	hsp := &HeroStatsPanel{
		asset:     asset,
		uiManager: ui,
		originX:   originX,
		originY:   originY,
		heroState: heroState,
		heroName:  heroName,
		heroClass: heroClass,
		labels:    &StatsPanelLabels{},
	}

	hsp.Logger = d2util.NewLogger()
	hsp.Logger.SetLevel(l)
	hsp.Logger.SetPrefix(logPrefix)

	return hsp
}

// HeroStatsPanel represents the hero status panel
type HeroStatsPanel struct {
	asset           *d2asset.AssetManager
	uiManager       *d2ui.UIManager
	panel           *d2ui.Sprite
	heroState       *d2hero.HeroStatsState
	heroName        string
	heroClass       d2enum.Hero
	labels          *StatsPanelLabels
	hands           [2]handLabels
	handSkills      func() [2]HandSkillInfo
	onCloseCb       func()
	panelGroup      *d2ui.WidgetGroup
	newStatPoints   *d2ui.WidgetGroup
	remainingPoints *d2ui.Label

	originX int
	originY int
	isOpen  bool

	clickMod d2enum.KeyMod // modifiers held at the last mouse press (shift/ctrl multi-spend)

	*d2util.Logger
}

// Load the data for the hero status panel
func (s *HeroStatsPanel) Load() {
	var err error

	s.panelGroup = s.uiManager.NewWidgetGroup(d2ui.RenderPriorityHeroStatsPanel)
	s.newStatPoints = s.uiManager.NewWidgetGroup(d2ui.RenderPriorityHeroStatsPanel)

	frame := s.uiManager.NewUIFrame(d2ui.FrameLeft)
	s.panelGroup.AddWidget(frame)

	s.panel, err = s.uiManager.NewSprite(d2resource.InventoryCharacterPanel, d2resource.PaletteSky)
	if err != nil {
		s.Error(err.Error())
	}

	w, h := frame.GetSize()
	staticPanel := s.uiManager.NewCustomWidgetCached(s.renderStaticMenu, w, h)
	s.panelGroup.AddWidget(staticPanel)

	closeButton := s.uiManager.NewButton(d2ui.ButtonTypeSquareClose, "")
	closeButton.SetVisible(false)
	closeButton.SetPosition(heroStatsCloseButtonX, heroStatsCloseButtonY)
	closeButton.OnActivated(func() { s.Close() })
	s.panelGroup.AddWidget(closeButton)

	s.loadNewStatPoints()
	s.setLayout()

	s.initStatValueLabels()
	s.panelGroup.SetVisible(false)
}

func (s *HeroStatsPanel) loadNewStatPoints() {
	field, err := s.uiManager.NewSprite(d2resource.HeroStatsPanelStatsPoints, d2resource.PaletteSky)
	if err != nil {
		s.Error(err.Error())
	}

	field.SetPosition(newStatsRemainingPointsFieldX, newStatsRemainingPointsFieldY+panelShiftY)
	s.newStatPoints.AddWidget(field)

	label1 := s.uiManager.NewLabel(d2resource.Font6, d2resource.PaletteSky)
	label1.SetPosition(newStatsRemainingPointsLabelX, newStatsRemainingPointsLabel1Y+panelShiftY)
	label1.SetText(s.asset.TranslateString("strchrstat"))
	label1.Color[0] = d2util.Color(d2gui.ColorRed)
	s.newStatPoints.AddWidget(label1)

	label2 := s.uiManager.NewLabel(d2resource.Font6, d2resource.PaletteSky)
	label2.SetPosition(newStatsRemainingPointsLabelX, newStatsRemainingPointsLabel2Y+panelShiftY)
	label2.SetText(s.asset.TranslateString("strchrrema"))
	label2.Color[0] = d2util.Color(d2gui.ColorRed)
	s.newStatPoints.AddWidget(label2)

	s.remainingPoints = s.uiManager.NewLabel(d2resource.Font16, d2resource.PaletteSky)
	s.remainingPoints.SetText(strconv.Itoa(s.heroState.StatsPoints))
	s.remainingPoints.SetPosition(newStatsRemainingPointsValueX, newStatsRemainingPointsValueY+panelShiftY)
	s.remainingPoints.Alignment = d2ui.HorizontalAlignCenter
	s.newStatPoints.AddWidget(s.remainingPoints)

	buttons := []struct {
		x  int
		y  int
		cb func()
	}{
		{205, 140, func() {
			s.heroState.Strength++
		}},
		{205, 202, func() {
			s.heroState.Dexterity++
		}},
		{205, 288, func() {
			s.heroState.Vitality++
		}},
		{205, 350, func() {
			s.heroState.Energy++
		}},
	}

	var socket *d2ui.Sprite

	var button *d2ui.Button

	for _, i := range buttons {
		currentValue := i

		socket, err = s.uiManager.NewSprite(d2resource.HeroStatsPanelSocket, d2resource.PaletteSky)
		if err != nil {
			s.Error(err.Error())
		}

		socket.SetPosition(i.x+addStatSocketOffsetX, i.y+addStatSocketOffsetY+panelShiftY)
		s.newStatPoints.AddWidget(socket)

		button = s.uiManager.NewButton(d2ui.ButtonTypeAddSkill, d2resource.PaletteSky)
		button.SetPosition(i.x, i.y+panelShiftY)
		button.OnActivated(func() {
			n := StatSpendCount(s.clickMod, s.heroState.StatsPoints)
			if n == 0 {
				return
			}

			for k := 0; k < n; k++ {
				currentValue.cb()
				s.heroState.StatsPoints--
			}

			if s.heroState.Recalc != nil {
				s.heroState.Recalc() // life, mana, defense, attack rating follow the new attribute
			}

			s.remainingPoints.SetText(strconv.Itoa(s.heroState.StatsPoints))
			s.setStatValues()
			s.setLayout()
		})
		s.newStatPoints.AddWidget(button)
	}
}

func (s *HeroStatsPanel) setLayout() {
	s.newStatPoints.SetVisible(s.heroState.StatsPoints > 0 && s.IsOpen())
}

// SetClickMod records the modifier keys held at the latest mouse press.
func (s *HeroStatsPanel) SetClickMod(m d2enum.KeyMod) { s.clickMod = m }

// IsOpen returns true if the hero status panel is open
func (s *HeroStatsPanel) IsOpen() bool {
	return s.isOpen
}

// Toggle toggles the visibility of the hero status panel
func (s *HeroStatsPanel) Toggle() {
	if s.isOpen {
		s.Close()
	} else {
		s.Open()
	}
}

// Open opens the hero status panel
func (s *HeroStatsPanel) Open() {
	s.isOpen = true
	s.panelGroup.SetVisible(true)
	s.setLayout()
}

// Close closed the hero status panel
func (s *HeroStatsPanel) Close() {
	s.isOpen = false
	s.panelGroup.SetVisible(false)
	s.setLayout()
	s.onCloseCb()
}

// SetOnCloseCb the callback run on closing the HeroStatsPanel
func (s *HeroStatsPanel) SetOnCloseCb(cb func()) {
	s.onCloseCb = cb
}

// Advance updates labels on the panel
func (s *HeroStatsPanel) Advance(elapsed float64) {
	if !s.isOpen {
		return
	}

	s.setStatValues()
}

func (s *HeroStatsPanel) renderStaticMenu(target d2interface.Surface) {
	s.renderStaticPanelFrames(target)
	s.renderStaticLabels(target)
}

// nolint:dupl // see quest_log.go.renderStaticPanelFrames comment
func (s *HeroStatsPanel) renderStaticPanelFrames(target d2interface.Surface) {
	frames := []int{
		statsPanelTopLeft,
		statsPanelTopRight,
		statsPanelBottomRight,
		statsPanelBottomLeft,
	}

	currentX := s.originX + statsPanelOffsetX
	currentY := s.originY + statsPanelOffsetY

	for _, frameIndex := range frames {
		if err := s.panel.SetCurrentFrame(frameIndex); err != nil {
			s.Error(err.Error())
		}

		w, h := s.panel.GetCurrentFrameSize()

		switch frameIndex {
		case statsPanelTopLeft:
			s.panel.SetPosition(currentX, currentY+h)
			currentX += w
		case statsPanelTopRight:
			s.panel.SetPosition(currentX, currentY+h)
			currentY += h
		case statsPanelBottomRight:
			s.panel.SetPosition(currentX, currentY+h)
		case statsPanelBottomLeft:
			s.panel.SetPosition(currentX-w, currentY+h)
		}

		s.panel.Render(target)
	}
}

// statsLabelSpec is one static caption of the character panel; name is the key of the layout log.
type statsLabelSpec struct {
	name        string
	x, y        int
	txt         string
	font        string
	centerAlign bool
}

// staticLabelSpecs lists the captions of the panel in the order they are drawn.
func (s *HeroStatsPanel) staticLabelSpecs() []statsLabelSpec {
	fr := strings.Split(s.asset.TranslateString("strchrfir"), "\n")
	lr := strings.Split(s.asset.TranslateString("strchrlit"), "\n")
	cr := strings.Split(s.asset.TranslateString("strchrcol"), "\n")
	pr := strings.Split(s.asset.TranslateString("strchrpos"), "\n")

	return []statsLabelSpec{
		{"name", labelHeroNameX, labelHeroNameY, s.heroName, d2resource.Font16, true},
		{"class", labelHeroClassX, labelHeroClassY, s.asset.TranslateString(s.heroClass), d2resource.Font16, true},

		{"level", labelLevelX, labelLevelY, s.asset.TranslateString("strchrlvl"), d2resource.Font6, true},
		{"experience", labelExperienceX, labelExperienceY, s.asset.TranslateString("strchrexp"), d2resource.Font6, true},
		{"nextlevel", labelNextLevelX, labelNextLevelY, s.asset.TranslateString("strchrnxtlvl"), d2resource.Font6, true},
		{"strength", labelStrengthX, labelStrengthY, s.asset.TranslateString("strchrstr"), d2resource.Font6, true},
		{"dexterity", labelDexterityX, labelDexterityY, s.asset.TranslateString("strchrdex"), d2resource.Font6, true},
		{"vitality", labelVitalityX, labelVitalityY, s.asset.TranslateString("strchrvit"), d2resource.Font6, true},
		{"energy", labelEnergyX, labelEnergyY, s.asset.TranslateString("strchreng"), d2resource.Font6, true},
		{"defense", labelDefenseX, labelDefenseY, s.asset.TranslateString("strchrdef"), d2resource.Font6, true},
		{"stamina", labelStaminaX, labelStaminaY, s.asset.TranslateString("strchrstm"), d2resource.Font6, true},
		{"life", labelLifeX, labelLifeY, s.asset.TranslateString("strchrlif"), d2resource.Font6, true},
		{"mana", labelManaX, labelManaY, s.asset.TranslateString("strchrman"), d2resource.Font6, true},

		// can't use "Fire\nResistance" because line spacing is too big and breaks the layout
		{"fire1", labelResFireLine1X, labelResFireLine1Y, fr[0], d2resource.Font6, true},
		{"fire2", labelResFireLine2X, labelResFireLine2Y, fr[len(fr)-1], d2resource.Font6, true},

		{"cold1", labelResColdLine1X, labelResColdLine1Y, cr[0], d2resource.Font6, true},
		{"cold2", labelResColdLine2X, labelResColdLine2Y, cr[len(cr)-1], d2resource.Font6, true},

		{"lightning1", labelResLightLine1X, labelResLightLine1Y, lr[0], d2resource.Font6, true},
		{"lightning2", labelResLightLine2X, labelResLightLine2Y, lr[len(lr)-1], d2resource.Font6, true},

		{"poison1", labelResPoisLine1X, labelResPoisLine1Y, pr[0], d2resource.Font6, true},
		{"poison2", labelResPoisLine2X, labelResPoisLine2Y, pr[len(pr)-1], d2resource.Font6, true},
	}
}

func (s *HeroStatsPanel) renderStaticLabels(target d2interface.Surface) {
	var label *d2ui.Label

	// all static labels are not stored since we use them only once to generate the image cache
	staticLabelConfigs := s.staticLabelSpecs()

	for _, cfg := range staticLabelConfigs {
		label = s.createTextLabel(PanelText{
			cfg.x, cfg.y,
			cfg.txt,
			cfg.font,
			cfg.centerAlign,
		})

		label.Render(target)
	}
}

func (s *HeroStatsPanel) initStatValueLabels() {
	valueLabelConfigs := []struct {
		assignTo **d2ui.Label
		value    int
		x, y     int
	}{
		{&s.labels.Level, s.heroState.Level, 113, 107},
		{&s.labels.Experience, s.heroState.Experience, 203, 107},
		{&s.labels.NextLevelExp, s.heroState.NextLevelExp, 331, 107},
		{&s.labels.Strength, s.heroState.Strength, 175, 147},
		{&s.labels.Dexterity, s.heroState.Dexterity, 175, 209},
		{&s.labels.Vitality, s.heroState.Vitality, 175, 295},
		{&s.labels.Energy, s.heroState.Energy, 175, 355},
		{&s.labels.MaxStamina, s.heroState.MaxStamina, 330, 295},
		{&s.labels.Stamina, int(s.heroState.Stamina), 370, 295},
		{&s.labels.MaxHealth, s.heroState.MaxHealth, 330, 318},
		{&s.labels.Health, s.heroState.Health, 370, 318},
		{&s.labels.MaxMana, s.heroState.MaxMana, 330, 355},
		{&s.labels.Mana, s.heroState.Mana, 370, 355},
	}

	for _, cfg := range valueLabelConfigs {
		*cfg.assignTo = s.createStatValueLabel(cfg.value, cfg.x, cfg.y)
	}

	small := func(x, y int) *d2ui.Label {
		return s.createTextLabel(PanelText{X: x, Y: y, Text: "0", Font: d2resource.Font6, AlignCenter: true})
	}

	s.labels.Defense = s.createStatValueLabel(0, labelDefenseValueX, labelDefenseValueY)

	for h := range s.hands {
		dy := h * handRowShift
		mid := func(x1, x2 int) int { return (x1+x2)/2 + panelColumnX }
		mk := func(x, y int) *d2ui.Label {
			return s.createTextLabel(PanelText{X: x, Y: y + dy + panelTextY, Font: d2resource.Font6, AlignCenter: true})
		}

		nameX, arX := mid(handBoxNameX1, handBoxNameX2), mid(handARCaptionX1, handARCaptionX2)

		s.hands[h] = handLabels{
			name:          mk(nameX, handNameY),
			damageCaption: mk(nameX, handCaptionY),
			damage:        mk(mid(handDamageValueX1, handDamageX2), handDamageValueY),
			arCaption:     mk(arX, handARY-3),
			arCaption2:    mk(arX, handARY+5),
			ar:            mk(mid(handARValueX1, handARValueX2), handARY),
		}
	}

	for i := range s.labels.Resist {
		s.labels.Resist[i] = small(labelResValueX, resistValueY[i])
	}

	s.setDerivedValues()
}

// setAttributeValues prints strength, dexterity, vitality and energy. The exe shows the current value (with the
// bonuses of items and states) and colours it against the allocated base: blue above, red below (VERIFIED 0x4a43f0).
func (s *HeroStatsPanel) setAttributeValues() {
	st := s.heroState
	cur := [4]int{st.Strength, st.Dexterity, st.Vitality, st.Energy}

	if t := st.Totals; t != nil && t.Str+t.Dex+t.Vit+t.Ene > 0 {
		cur = [4]int{t.Str, t.Dex, t.Vit, t.Ene}
	}

	base := [4]int{st.Strength, st.Dexterity, st.Vitality, st.Energy}

	for i, l := range []*d2ui.Label{s.labels.Strength, s.labels.Dexterity, s.labels.Vitality, s.labels.Energy} {
		l.SetText(strconv.Itoa(cur[i]))
		setStatColour(l, StatValueColour(cur[i], base[i]).FontIndex())
	}
}

// resistLine is the text and font colour of a resistance row; maxTotal is the cap Totals reports (75 + bonus).
func resistLine(shown, maxTotal int) (string, int) {
	return resistLineStates(shown, maxTotal, false, false)
}

// resistLineStates is resistLine with the blue / red state masks of the element (d2state.PanelFlags).
func resistLineStates(shown, maxTotal int, raised, lowered bool) (string, int) {
	bonus := maxTotal - 75

	return strconv.Itoa(ResistDisplayed(shown, bonus)), ResistColour(shown, bonus, raised, lowered)
}

// setStatColour tints a value label with one of the exe's font colour indices.
func boxFont(c d2statlist.BoxColour) int {
	switch c {
	case d2statlist.BoxRed:
		return StatFontRed
	case d2statlist.BoxBlue:
		return StatFontBlue
	}

	return StatFontWhite
}

func setStatColour(l *d2ui.Label, idx int) {
	l.Color[0] = StatFontRGBA(idx)
}

// setDerivedValues shows the defense, attack rating, damage and resistances
// computed from the equipment.
func (s *HeroStatsPanel) setDerivedValues() {
	t := s.heroState.Totals
	if t == nil || s.labels.Defense == nil {
		return
	}

	var flags d2state.PanelFlags

	holy := false

	if f := s.heroState.PanelStates; f != nil {
		flags, holy = f()
	}

	s.labels.Defense.SetText(strconv.Itoa(t.Defense))
	setStatColour(s.labels.Defense, DefenseColour(DefenseRaised(flags.DefenseBlue(), t.ShieldWorn, holy), flags.DefenseRed()))
	s.setHandBoxes(t.DamageMin, t.DamageMax, t.AttackRating, flags)

	// the cap is 75 plus the max-resist bonus, at most 95; the screen prints the clamped number and colours it
	// gold at the cap, red below zero (VERIFIED 0x4a43f0); the states.txt rfblue.. / rfred.. columns turn it blue / red (d2state/panel_colour.go)
	for i, l := range s.labels.Resist {
		txt, col := resistLineStates(t.ResistShown[i], t.MaxResist[i], flags.ResistBlue(i), flags.ResistRed(i))
		l.SetText(txt)
		setStatColour(l, col)
	}
}

// SetHandSkills gives the panel the hero's left and right skills.
func (s *HeroStatsPanel) SetHandSkills(f func() [2]HandSkillInfo) { s.handSkills = f }

// setHandBoxes fills the four boxes right of Strength and Dexterity.
func (s *HeroStatsPanel) setHandBoxes(dmin, dmax, ar int, flags d2state.PanelFlags) {
	var skills [2]HandSkillInfo
	if s.handSkills != nil {
		skills = s.handSkills()
	}

	capDamage := s.asset.TranslateString("strchrskm")
	// string 0xfe1 ("strchrrat") is the caption when the skill has no to-hit element, 0xfdf ("%s\nAttack Rating") otherwise
	capAR := strings.ReplaceAll(s.asset.TranslateString("strchrrat"), "%s", "Attack") // the %s argument: the exe passes a wide string buffer (auStack_268, SKILLDESC_DrawSkillDescription 0x4ea490) that the
	// descatt function fills (its 4th output); which word it holds was not read, "Attack" is an OD2 guess

	for h, l := range s.hands {
		if l.name == nil {
			continue
		}

		sk := skills[h]
		l.name.SetText(sk.Name)

		damage, rating, dcap, acap := "", "", "", ""

		dmgCol, arCol := DefenseColour(flags.DamageBlue(), flags.DamageRed()), DefenseColour(flags.AttackBlue(), flags.AttackRed())

		if sk.Name != "" && sk.Weapon {
			lo, hi, rt := dmin, dmax, ar

			if t := s.heroState.Totals; t != nil {
				srcDam := sk.SrcDam
				if srcDam == 0 {
					srcDam = 128 // unknown: the whole weapon range
				}

				if tmin, tmax, ok := t.ThrownBox(0); ok && sk.Thrown {
					lo, hi = tmin, tmax // 0x4e7470: the thrown range, no flat damage
				} else {
					box := t.SkillDamageBox(d2statlist.SkillBox{
						SrcDam: srcDam, SkillMin: sk.SkillMin, SkillMax: sk.SkillMax, Bit8: sk.Bit8, StateBonus: sk.StateBonus,
						DmgBlue: flags.DamageBlue(), DmgRed: flags.DamageRed(),
					})
					lo, hi, dmgCol = box.Min, box.Max, boxFont(box.Colour)
				}

				var col d2statlist.BoxColour

				rt, col = d2statlist.SkillAttackRating(t.AttackRatingFlat, t.ToHitPct, 0, sk.ToHitBonus, false,
					flags.AttackBlue(), flags.AttackRed())
				arCol = boxFont(col)
			}

			damage = strconv.Itoa(lo) + "-" + strconv.Itoa(hi)
			rating = strconv.Itoa(rt)
			dcap, acap = capDamage, capAR
		}

		l.damageCaption.SetText(dcap)
		l.damage.SetText(damage)
		// a two line caption: the lines stand 8 px apart around the box middle (OD2 layout: the exe's rows 3 and 4 of the table at 0x72bff8 are y 160 and 163, see above; the row each caption line uses was not recovered)
		first, second := acap, ""
		if i := strings.Index(acap, "\n"); i >= 0 {
			first, second = acap[:i], acap[i+1:]
		}

		l.arCaption.SetText(first)
		l.arCaption2.SetText(second)
		l.ar.SetText(rating)
		setStatColour(l.damage, dmgCol)
		setStatColour(l.ar, arCol)
	}
}

func (s *HeroStatsPanel) setStatValues() {
	// the "Stat Points Remaining" number follows the hero (a level up while the panel exists showed the old count;
	// wave 41 hunt: 5 points per level but the box read 0 beside the red + buttons)
	if s.remainingPoints != nil {
		s.remainingPoints.SetText(strconv.Itoa(s.heroState.StatsPoints))
	}

	s.labels.Level.SetText(strconv.Itoa(s.heroState.Level))
	s.labels.Experience.SetText(strconv.Itoa(s.heroState.Experience))
	s.labels.NextLevelExp.SetText(strconv.Itoa(s.heroState.NextLevelExp))

	s.setAttributeValues()

	s.labels.MaxHealth.SetText(strconv.Itoa(s.heroState.MaxHealth))
	s.labels.Health.SetText(strconv.Itoa(s.heroState.Health))

	s.labels.MaxStamina.SetText(strconv.Itoa(s.heroState.MaxStamina))
	s.labels.Stamina.SetText(strconv.Itoa(int(s.heroState.Stamina)))

	s.labels.MaxMana.SetText(strconv.Itoa(s.heroState.MaxMana))
	s.labels.Mana.SetText(strconv.Itoa(s.heroState.Mana))

	// VERIFIED 0x4a43f0: current life, mana and stamina are white, the maxima blue / red against their base
	setStatColour(s.labels.Health, StatFontWhite)
	setStatColour(s.labels.Mana, StatFontWhite)
	setStatColour(s.labels.Stamina, StatFontWhite)
	setStatColour(s.labels.MaxHealth, VitalColour(statIDMaxLife, s.heroState.MaxHealth, s.heroState.BaseMaxHealth))
	setStatColour(s.labels.MaxMana, VitalColour(statIDMaxMana, s.heroState.MaxMana, s.heroState.BaseMaxMana))
	setStatColour(s.labels.MaxStamina, VitalColour(statIDMaxStamina, s.heroState.MaxStamina, s.heroState.BaseMaxStamina))

	s.setDerivedValues()
}

// Summary describes the values the panel shows, for the autotest PANEL log line.
// shownExp and shownNext are the label texts as drawn.
func (s *HeroStatsPanel) Summary() string {
	s.setStatValues()

	st := s.heroState

	return fmt.Sprintf("level=%d name=%q class=%s str=%d dex=%d vit=%d ene=%d hp=%d/%d mana=%d/%d stamina=%d/%d "+
		"exp=%d next=%d statpoints=%d skillpoints=%d shownExp=%s shownNext=%s hands=%s",
		st.Level, s.heroName, s.heroClass, st.Strength, st.Dexterity, st.Vitality, st.Energy,
		st.Health, st.MaxHealth, st.Mana, st.MaxMana, int(st.Stamina), st.MaxStamina,
		st.Experience, st.NextLevelExp, st.StatsPoints, st.SkillPoints,
		s.labels.Experience.GetText(), s.labels.NextLevelExp.GetText(), s.handSummary())
}

// handSummary lists the texts of the hand skill boxes ("name|damage caption|damage|AR caption|AR" per hand).
func (s *HeroStatsPanel) handSummary() string {
	var out []string

	for _, l := range s.hands {
		if l.name == nil {
			continue
		}

		out = append(out, fmt.Sprintf("%q|%q|%q|%q|%q", l.name.GetText(), l.damageCaption.GetText(), l.damage.GetText(),
			l.arCaption.GetText(), l.ar.GetText()))
	}

	return strings.Join(out, ",")
}

func (s *HeroStatsPanel) createStatValueLabel(stat, x, y int) *d2ui.Label {
	text := strconv.Itoa(stat)
	return s.createTextLabel(PanelText{X: x, Y: y, Text: text, Font: d2resource.Font16, AlignCenter: true})
}

func (s *HeroStatsPanel) createTextLabel(element PanelText) *d2ui.Label {
	label := s.uiManager.NewLabel(element.Font, d2resource.PaletteStatic)
	if element.AlignCenter {
		label.Alignment = d2ui.HorizontalAlignCenter
	}

	label.SetText(element.Text)
	label.SetPosition(element.X, element.Y+panelShiftY)
	s.panelGroup.AddWidget(label)

	return label
}
