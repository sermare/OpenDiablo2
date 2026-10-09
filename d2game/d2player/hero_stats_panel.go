package d2player

import (
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
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
	statsPanelOffsetX, statsPanelOffsetY = 80, 64
)

const (
	labelLevelX, labelLevelY = 110, 100

	labelHeroNameX, labelHeroNameY   = 165, 72
	labelHeroClassX, labelHeroClassY = 330, 74

	labelExperienceX, labelExperienceY = 200, 100
	labelNextLevelX, labelNextLevelY   = 330, 100

	labelStrengthX, labelStrengthY   = 100, 150
	labelDexterityX, labelDexterityY = 100, 213
	labelVitalityX, labelVitalityY   = 95, 300
	labelEnergyX, labelEnergyY       = 100, 360

	labelDefenseX, labelDefenseY = 280, 260
	labelStaminaX, labelStaminaY = 280, 300
	labelLifeX, labelLifeY       = 280, 322
	labelManaX, labelManaY       = 280, 360

	labelResFireLine1X, labelResFireLine1Y   = 310, 396
	labelResFireLine2X, labelResFireLine2Y   = 310, 403
	labelResColdLine1X, labelResColdLine1Y   = 310, 444
	labelResColdLine2X, labelResColdLine2Y   = 310, 452
	labelResLightLine1X, labelResLightLine1Y = 310, 420
	labelResLightLine2X, labelResLightLine2Y = 310, 428
	labelResPoisLine1X, labelResPoisLine1Y   = 310, 468
	labelResPoisLine2X, labelResPoisLine2Y   = 310, 476
)

const (
	heroStatsCloseButtonX, heroStatsCloseButtonY = 208, 453
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
	Defense      *d2ui.Label
	AttackRating *d2ui.Label
	Damage       *d2ui.Label
	Resist       [4]*d2ui.Label // fire, cold, lightning, poison
}

// Positions of the derived value labels. UNVERIFIED against the original
// layout (no screenshot comparison): the resist numbers sit right of their
// captions, defense right of its caption, attack rating and damage below the
// strength and dexterity values.
const (
	labelDefenseValueX, labelDefenseValueY = 345, 263
	labelDamageValueX, labelDamageValueY   = 140, 177
	labelARValueX, labelARValueY           = 140, 237
	labelResValueX                         = 375
)

// resistValueY are the y positions of the fire, cold, lightning and poison values.
var resistValueY = [4]int{400, 448, 424, 472}

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
	onCloseCb       func()
	panelGroup      *d2ui.WidgetGroup
	newStatPoints   *d2ui.WidgetGroup
	remainingPoints *d2ui.Label

	originX int
	originY int
	isOpen  bool

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

	field.SetPosition(newStatsRemainingPointsFieldX, newStatsRemainingPointsFieldY)
	s.newStatPoints.AddWidget(field)

	label1 := s.uiManager.NewLabel(d2resource.Font6, d2resource.PaletteSky)
	label1.SetPosition(newStatsRemainingPointsLabelX, newStatsRemainingPointsLabel1Y)
	label1.SetText(s.asset.TranslateString("strchrstat"))
	label1.Color[0] = d2util.Color(d2gui.ColorRed)
	s.newStatPoints.AddWidget(label1)

	label2 := s.uiManager.NewLabel(d2resource.Font6, d2resource.PaletteSky)
	label2.SetPosition(newStatsRemainingPointsLabelX, newStatsRemainingPointsLabel2Y)
	label2.SetText(s.asset.TranslateString("strchrrema"))
	label2.Color[0] = d2util.Color(d2gui.ColorRed)
	s.newStatPoints.AddWidget(label2)

	s.remainingPoints = s.uiManager.NewLabel(d2resource.Font16, d2resource.PaletteSky)
	s.remainingPoints.SetText(strconv.Itoa(s.heroState.StatsPoints))
	s.remainingPoints.SetPosition(newStatsRemainingPointsValueX, newStatsRemainingPointsValueY)
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
		{205, 201, func() {
			s.heroState.Dexterity++
		}},
		{205, 286, func() {
			s.heroState.Vitality++
		}},
		{205, 347, func() {
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

		socket.SetPosition(i.x+addStatSocketOffsetX, i.y+addStatSocketOffsetY)
		s.newStatPoints.AddWidget(socket)

		button = s.uiManager.NewButton(d2ui.ButtonTypeAddSkill, d2resource.PaletteSky)
		button.SetPosition(i.x, i.y)
		button.OnActivated(func() {
			currentValue.cb()
			s.heroState.StatsPoints--

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

func (s *HeroStatsPanel) renderStaticLabels(target d2interface.Surface) {
	var label *d2ui.Label

	fr := strings.Split(s.asset.TranslateString("strchrfir"), "\n")
	lr := strings.Split(s.asset.TranslateString("strchrlit"), "\n")
	cr := strings.Split(s.asset.TranslateString("strchrcol"), "\n")
	pr := strings.Split(s.asset.TranslateString("strchrpos"), "\n")
	// all static labels are not stored since we use them only once to generate the image cache
	var staticLabelConfigs = []struct {
		x, y        int
		txt         string
		font        string
		centerAlign bool
	}{
		{labelHeroNameX, labelHeroNameY, s.heroName, d2resource.Font16, true},
		{labelHeroClassX, labelHeroClassY, s.asset.TranslateString(s.heroClass), d2resource.Font16, true},

		{labelLevelX, labelLevelY, s.asset.TranslateString("strchrlvl"), d2resource.Font6, true},
		{labelExperienceX, labelExperienceY, s.asset.TranslateString("strchrexp"), d2resource.Font6, true},
		{labelNextLevelX, labelNextLevelY, s.asset.TranslateString("strchrnxtlvl"), d2resource.Font6, true},
		{labelStrengthX, labelStrengthY, s.asset.TranslateString("strchrstr"), d2resource.Font6, false},
		{labelDexterityX, labelDexterityY, s.asset.TranslateString("strchrdex"), d2resource.Font6, false},
		{labelVitalityX, labelVitalityY, s.asset.TranslateString("strchrvit"), d2resource.Font6, false},
		{labelEnergyX, labelEnergyY, s.asset.TranslateString("strchreng"), d2resource.Font6, false},
		{labelDefenseX, labelDefenseY, s.asset.TranslateString("strchrdef"), d2resource.Font6, false},
		{labelStaminaX, labelStaminaY, s.asset.TranslateString("strchrstm"), d2resource.Font6, true},
		{labelLifeX, labelLifeY, s.asset.TranslateString("strchrlif"), d2resource.Font6, true},
		{labelManaX, labelManaY, s.asset.TranslateString("strchrman"), d2resource.Font6, true},

		// can't use "Fire\nResistance" because line spacing is too big and breaks the layout
		{labelResFireLine1X, labelResFireLine1Y, fr[0], d2resource.Font6, true},
		{labelResFireLine2X, labelResFireLine2Y, fr[len(fr)-1], d2resource.Font6, true},

		{labelResColdLine1X, labelResColdLine1Y, cr[0], d2resource.Font6, true},
		{labelResColdLine2X, labelResColdLine2Y, cr[len(cr)-1], d2resource.Font6, true},

		{labelResLightLine1X, labelResLightLine1Y, lr[0], d2resource.Font6, true},
		{labelResLightLine2X, labelResLightLine2Y, lr[len(lr)-1], d2resource.Font6, true},

		{labelResPoisLine1X, labelResPoisLine1Y, pr[0], d2resource.Font6, true},
		{labelResPoisLine2X, labelResPoisLine2Y, pr[len(pr)-1], d2resource.Font6, true},
	}

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
		{&s.labels.Level, s.heroState.Level, 112, 110},
		{&s.labels.Experience, s.heroState.Experience, 200, 110},
		{&s.labels.NextLevelExp, s.heroState.NextLevelExp, 330, 110},
		{&s.labels.Strength, s.heroState.Strength, 175, 147},
		{&s.labels.Dexterity, s.heroState.Dexterity, 175, 207},
		{&s.labels.Vitality, s.heroState.Vitality, 175, 295},
		{&s.labels.Energy, s.heroState.Energy, 175, 355},
		{&s.labels.MaxStamina, s.heroState.MaxStamina, 330, 295},
		{&s.labels.Stamina, int(s.heroState.Stamina), 370, 295},
		{&s.labels.MaxHealth, s.heroState.MaxHealth, 330, 320},
		{&s.labels.Health, s.heroState.Health, 370, 320},
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
	s.labels.Damage = small(labelDamageValueX, labelDamageValueY)
	s.labels.AttackRating = small(labelARValueX, labelARValueY)

	for i := range s.labels.Resist {
		s.labels.Resist[i] = small(labelResValueX, resistValueY[i])
	}

	s.setDerivedValues()
}

// setDerivedValues shows the defense, attack rating, damage and resistances
// computed from the equipment.
func (s *HeroStatsPanel) setDerivedValues() {
	t := s.heroState.Totals
	if t == nil || s.labels.Defense == nil {
		return
	}

	s.labels.Defense.SetText(strconv.Itoa(t.Defense))
	s.labels.AttackRating.SetText("AR " + strconv.Itoa(t.AttackRating))
	s.labels.Damage.SetText("Dmg " + strconv.Itoa(t.DamageMin) + "-" + strconv.Itoa(t.DamageMax))

	for i, l := range s.labels.Resist {
		l.SetText(strconv.Itoa(t.ResistShown[i]))
	}
}

func (s *HeroStatsPanel) setStatValues() {
	s.labels.Level.SetText(strconv.Itoa(s.heroState.Level))
	s.labels.Experience.SetText(strconv.Itoa(s.heroState.Experience))
	s.labels.NextLevelExp.SetText(strconv.Itoa(s.heroState.NextLevelExp))

	s.labels.Strength.SetText(strconv.Itoa(s.heroState.Strength))
	s.labels.Dexterity.SetText(strconv.Itoa(s.heroState.Dexterity))
	s.labels.Vitality.SetText(strconv.Itoa(s.heroState.Vitality))
	s.labels.Energy.SetText(strconv.Itoa(s.heroState.Energy))

	s.labels.MaxHealth.SetText(strconv.Itoa(s.heroState.MaxHealth))
	s.labels.Health.SetText(strconv.Itoa(s.heroState.Health))

	s.labels.MaxStamina.SetText(strconv.Itoa(s.heroState.MaxStamina))
	s.labels.Stamina.SetText(strconv.Itoa(int(s.heroState.Stamina)))

	s.labels.MaxMana.SetText(strconv.Itoa(s.heroState.MaxMana))
	s.labels.Mana.SetText(strconv.Itoa(s.heroState.Mana))

	s.setDerivedValues()
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
	label.SetPosition(element.X, element.Y)
	s.panelGroup.AddWidget(label)

	return label
}
