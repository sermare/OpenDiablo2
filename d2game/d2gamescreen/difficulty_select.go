package d2gamescreen

import (
	"os"
	"strconv"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2difficulty"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2screen"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2client/d2clientconnectiontype"
)

const (
	diffLabelX, diffLabelY   = 400, 150
	diffBtnX                 = 264
	diffBtnY0, diffBtnStep   = 220, 55
	diffCancelX, diffCancelY = 340, 430
)

// DifficultySelect is the screen shown when a character that has unlocked a
// higher difficulty is loaded: Normal is always available, Nightmare once
// Normal is finished and Hell once Nightmare is (d2difficulty.Unlocked). The
// choice is saved in the hero file and, on the next export, in the active
// difficulty byte of the .d2s header.
type DifficultySelect struct {
	background *d2ui.Sprite
	title      *d2ui.Label
	buttons    [d2difficulty.Count]*d2ui.Button
	cancel     *d2ui.Button

	autoPick d2difficulty.Level
	autoSet  bool

	hero     *d2hero.HeroState
	factory  *d2hero.HeroStateFactory
	unlocked [d2difficulty.Count]bool

	filePath       string
	connectionType d2clientconnectiontype.ClientConnectionType
	connectionHost string

	asset     *d2asset.AssetManager
	navigator d2interface.Navigator
	uiManager *d2ui.UIManager

	*d2util.Logger
}

// CreateDifficultySelect creates the screen for the hero saved in filePath.
func CreateDifficultySelect(
	navigator d2interface.Navigator,
	asset *d2asset.AssetManager,
	ui *d2ui.UIManager,
	filePath string,
	connType d2clientconnectiontype.ClientConnectionType,
	host string,
	l d2util.LogLevel,
) (*DifficultySelect, error) {
	factory, err := d2hero.NewHeroStateFactory(asset)
	if err != nil {
		return nil, err
	}

	v := &DifficultySelect{
		hero: factory.LoadHeroState(filePath), factory: factory, filePath: filePath,
		connectionType: connType, connectionHost: host,
		asset: asset, navigator: navigator, uiManager: ui,
	}

	v.Logger = d2util.NewLogger()
	v.Logger.SetPrefix(logPrefix)
	v.Logger.SetLevel(l)

	return v, nil
}

// AutoDifficulty reads OD2_AUTODIFFICULTY=0|1|2, the difficulty test scenarios
// pick on the difficulty screen without clicking.
func AutoDifficulty() (d2difficulty.Level, bool) {
	n, err := strconv.Atoi(os.Getenv("OD2_AUTODIFFICULTY"))
	if err != nil || n < 0 || n > 2 {
		return 0, false
	}

	return d2difficulty.Level(n), true
}

// NeedsDifficultyChoice reports whether a hero has more than one difficulty to
// choose from.
func NeedsDifficultyChoice(h *d2hero.HeroState) bool {
	if h == nil {
		return false
	}

	u := h.UnlockedDifficulties()

	return u[d2difficulty.Nightmare]
}

func difficultyName(asset *d2asset.AssetManager, l d2difficulty.Level) string {
	key := [...]int{d2enum.NormalLabel, d2enum.NightmareLabel, d2enum.HellLabel}[l]

	return asset.TranslateString(key)
}

// OnLoad creates the buttons.
func (v *DifficultySelect) OnLoad(_ d2screen.LoadingState) {
	var err error

	v.background, err = v.uiManager.NewSprite(d2resource.GameSelectScreen, d2resource.PaletteSky)
	if err != nil {
		v.Error(err.Error())
	} else {
		v.background.SetPosition(backgroundX, backgroundY)
	}

	v.title = v.uiManager.NewLabel(d2resource.Font30, d2resource.PaletteStatic)
	v.title.Alignment = d2ui.HorizontalAlignCenter
	v.title.SetText(v.asset.TranslateString(d2enum.SelectDifficultyLabel))
	v.title.Color[0] = d2util.Color(lightBrown)
	v.title.SetPosition(diffLabelX, diffLabelY)

	if v.hero != nil {
		v.unlocked = v.hero.UnlockedDifficulties()
	} else {
		v.unlocked[d2difficulty.Normal] = true
	}

	v.Infof("DIFFICULTY screen hero=%q unlocked=%v current=%v", heroName(v.hero), v.unlocked, v.currentLevel())

	for l := d2difficulty.Normal; l < d2difficulty.Count; l++ {
		level := l
		b := v.uiManager.NewButton(d2ui.ButtonTypeWide, difficultyName(v.asset, level))
		b.SetPosition(diffBtnX, diffBtnY0+int(level)*diffBtnStep)
		b.SetEnabled(v.unlocked[level])
		b.OnActivated(func() { v.Choose(level) })
		v.buttons[level] = b
	}

	if lv, ok := AutoDifficulty(); ok {
		v.Infof("DIFFICULTY auto pick %v", lv)

		if !v.unlocked[lv] {
			v.Warningf("DIFFICULTY auto pick %v refused: not unlocked", lv)
			lv = v.currentLevel()
		}

		v.autoPick, v.autoSet = lv, true
	}

	v.cancel = v.uiManager.NewButton(d2ui.ButtonTypeMedium, v.asset.TranslateString(d2enum.CancelLabel))
	v.cancel.SetPosition(diffCancelX, diffCancelY)
	v.cancel.OnActivated(func() { v.navigator.ToCharacterSelect(v.connectionType, v.connectionHost) })
}

func heroName(h *d2hero.HeroState) string {
	if h == nil {
		return ""
	}

	return h.HeroName
}

func (v *DifficultySelect) currentLevel() d2difficulty.Level {
	if v.hero == nil {
		return d2difficulty.Normal
	}

	return d2difficulty.Clamp(int(v.hero.Difficulty))
}

// Choose makes a difficulty the hero's, saves the hero and starts the game.
// A difficulty that is not unlocked is ignored.
func (v *DifficultySelect) Choose(l d2difficulty.Level) {
	if v.hero == nil || l < d2difficulty.Normal || l >= d2difficulty.Count || !v.unlocked[l] {
		return
	}

	if err := v.hero.ChooseDifficulty(l, false); err != nil {
		v.Warningf("DIFFICULTY %v", err)

		return
	}

	if err := v.factory.Save(v.hero); err != nil {
		v.Errorf("DIFFICULTY could not save %s: %v", v.hero.HeroName, err)
	}

	v.Infof("DIFFICULTY chosen hero=%q difficulty=%v", v.hero.HeroName, l)
	v.navigator.ToCreateGame(v.filePath, v.connectionType, v.connectionHost)
}

// Render draws the screen.
func (v *DifficultySelect) Render(screen d2interface.Surface) {
	if v.background != nil {
		v.background.RenderSegmented(screen, 4, 3, 0)
	}

	v.title.Render(screen)
}

// Advance runs the OD2_AUTODIFFICULTY pick, if any.
func (v *DifficultySelect) Advance(_ float64) error {
	if v.autoSet {
		v.autoSet = false
		v.Choose(v.autoPick)
	}

	return nil
}
