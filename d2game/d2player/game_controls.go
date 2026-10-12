package d2player

import (
	"fmt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2display"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2input/d2gamepad"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2geom"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2vendor"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2herostats"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2maprenderer"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

const (
	logPrefix = "Player"
)

// Panel represents the panel at the bottom of the game screen
type Panel interface {
	IsOpen() bool
	Open()
	Close()
}

const mouseBtnActionsThreshold = 0.25

const (
	// Since they require special handling, not considering (1) globes, (2) content of the mini panel, (3) belt
	leftSkill actionableType = iota
	xp
	stamina
	rightSkill
	hpGlobe
	manaGlobe
)

// Hit areas of the always visible interface, as the original tests them (uilayout.go: HUDRects, with the
// functions they were read from). Left, top, width, height in pixels at 800x600.
const (
	leftSkillX,
	leftSkillY,
	leftSkillWidth,
	leftSkillHeight = 117, 552, 49, 49

	xpX,
	xpY,
	xpWidth,
	xpHeight = 254, 557, 124, 10

	staminaX,
	staminaY,
	staminaWidth,
	staminaHeight = 273, 573, 103, 19

	rightSkillX,
	rightSkillY,
	rightSkillWidth,
	rightSkillHeight = 635, 552, 49, 49

	hpGlobeX,
	hpGlobeY,
	hpGlobeWidth,
	hpGlobeHeight = 30, 525, 81, 61

	manaGlobeX,
	manaGlobeY,
	manaGlobeWidth,
	manaGlobeHeight = 689, 525, 81, 61
)

const (
	menuBottomRectX,
	menuBottomRectY,
	menuBottomRectW,
	menuBottomRectH = 0, 550, 800, 50

	menuLeftRectX,
	menuLeftRectY,
	menuLeftRectW,
	menuLeftRectH = 0, 0, 400, 600

	menuRightRectX,
	menuRightRectY,
	menuRightRectW,
	menuRightRectH = 400, 0, 400, 600
)

// NewGameControls creates a GameControls instance and returns a pointer to it
// nolint:funlen // doesn't make sense to split this up
func NewGameControls(
	asset *d2asset.AssetManager,
	renderer d2interface.Renderer,
	hero *d2mapentity.Player,
	mapEngine *d2mapengine.MapEngine,
	escapeMenu *EscapeMenu,
	mapRenderer *d2maprenderer.MapRenderer,
	inputListener inputCallbackListener,
	term d2interface.Terminal,
	ui *d2ui.UIManager,
	keyMap *KeyMap,
	audioProvider d2interface.AudioProvider,
	l d2util.LogLevel,
	isSinglePlayer bool,
	players map[string]*d2mapentity.Player,
) (*GameControls, error) {
	var inventoryRecordKey string

	switch hero.Class {
	case d2enum.HeroAssassin:
		inventoryRecordKey = "Assassin2"
	case d2enum.HeroAmazon:
		inventoryRecordKey = "Amazon2"
	case d2enum.HeroBarbarian:
		inventoryRecordKey = "Barbarian2"
	case d2enum.HeroDruid:
		inventoryRecordKey = "Druid2"
	case d2enum.HeroNecromancer:
		inventoryRecordKey = "Necromancer2"
	case d2enum.HeroPaladin:
		inventoryRecordKey = "Paladin2"
	case d2enum.HeroSorceress:
		inventoryRecordKey = "Sorceress2"
	default:
		return nil, fmt.Errorf("unknown hero class: %d", hero.Class)
	}

	actionableRegions := []actionableRegion{
		{leftSkill, d2geom.Rectangle{
			Left:   leftSkillX,
			Top:    leftSkillY,
			Width:  leftSkillWidth,
			Height: leftSkillHeight,
		}},
		{xp, d2geom.Rectangle{
			Left:   xpX,
			Top:    xpY,
			Width:  xpWidth,
			Height: xpHeight,
		}},
		{stamina, d2geom.Rectangle{
			Left:   staminaX,
			Top:    staminaY,
			Width:  staminaWidth,
			Height: staminaHeight,
		}},
		{rightSkill, d2geom.Rectangle{
			Left:   rightSkillX,
			Top:    rightSkillY,
			Width:  rightSkillWidth,
			Height: rightSkillHeight,
		}},
		{hpGlobe, d2geom.Rectangle{
			Left:   hpGlobeX,
			Top:    hpGlobeY,
			Width:  hpGlobeWidth,
			Height: hpGlobeHeight,
		}},
		{manaGlobe, d2geom.Rectangle{
			Left:   manaGlobeX,
			Top:    manaGlobeY,
			Width:  manaGlobeWidth,
			Height: manaGlobeHeight,
		}},
	}
	inventoryRecord := asset.Records.Layout.Inventory[inventoryRecordKey]

	heroStatsPanel := NewHeroStatsPanel(asset, ui, hero.Name(), hero.Class, l, hero.Stats)

	questLog := NewQuestLog(asset, ui, l, audioProvider, hero.Act)

	inventory, err := NewInventory(asset, ui, l, hero.Gold, inventoryRecord)
	if err != nil {
		return nil, err
	}

	skilltree := newSkillTree(hero.Skills, hero.Class, hero.Stats, asset, l, ui)

	miniPanel := newMiniPanel(asset, ui, l, isSinglePlayer)

	heroState, err := d2hero.NewHeroStateFactory(asset)
	if err != nil {
		return nil, err
	}

	helpOverlay := NewHelpOverlay(asset, ui, l, keyMap)

	const blackAlpha50percent = 0x0000007f

	gc := &GameControls{
		asset:          asset,
		ui:             ui,
		renderer:       renderer,
		hero:           hero,
		heroState:      heroState,
		escapeMenu:     escapeMenu,
		inputListener:  inputListener,
		mapRenderer:    mapRenderer,
		mapEngine:      mapEngine,
		inventory:      inventory,
		skilltree:      skilltree,
		heroStatsPanel: heroStatsPanel,
		mercPanel:      NewMercPanel(asset, ui, l),
		mercPortrait:   NewMercPortrait(asset, ui),
		questLog:       questLog,
		HelpOverlay:    helpOverlay,
		NPCMenu:        NewNPCMenu(asset, ui),
		HireList:       NewHireList(ui),
		Speech:         NewSpeechBubble(ui),
		Waypoints:      NewWaypointPanel(asset, ui),
		keyMap:         keyMap,
		bottomMenuRect: &d2geom.Rectangle{
			Left:   menuBottomRectX,
			Top:    menuBottomRectY,
			Width:  menuBottomRectW,
			Height: menuBottomRectH,
		},
		leftMenuRect: &d2geom.Rectangle{
			Left:   menuLeftRectX,
			Top:    menuLeftRectY,
			Width:  menuLeftRectW,
			Height: menuLeftRectH,
		},
		rightMenuRect: &d2geom.Rectangle{
			Left:   menuRightRectX,
			Top:    menuRightRectY,
			Width:  menuRightRectW,
			Height: menuRightRectH,
		},
		actionableRegions:      actionableRegions,
		lastLeftBtnActionTime:  0,
		lastRightBtnActionTime: 0,
		isSinglePlayer:         isSinglePlayer,
	}

	trade, err := NewTradeWindow(asset, ui, l, inventory, hero, gc.saveHero, gc.onCloseTrade)
	if err != nil {
		return nil, err
	}

	gc.Trade = trade
	trade.OnImpossible = func() { gc.sayHeroVoice(0x13) }
	trade.OnCantCarry = func() { gc.sayHeroVoice(0x17) }
	gc.PTrade = newPlayerTradeWindow(asset, ui, l, gc)
	gc.Identify = NewIdentifyWindow(asset, ui, l, inventory, hero, gc.saveHero, gc.onCloseTrade)

	inventory.savedItems = hero.Containers != nil
	inventory.itemHook = gc.itemTooltipLines
	inventory.item.DescribeContext = gc.describeContext
	gc.stash = NewContainerPanel(asset, ui, l, inventory, stashKind, gc.saveHero)
	gc.cube = NewContainerPanel(asset, ui, l, inventory, cubeKind, gc.saveHero)
	gc.cube.SetOnTransmute(gc.onTransmuteButton)
	gc.loadStashCoin()
	gc.inventory.goldClick = gc.onInventoryGoldClick
	gc.inventory.moveGoldPanel.SetOnConfirm(func(k d2inventory.GoldDialogKind, n int) { gc.GoldMove(k, n) })
	trade.SetStashGold(&hero.StashGold)
	gc.stash.itemHook, gc.cube.itemHook = gc.itemTooltipLines, gc.itemTooltipLines
	gc.Trade.compareHook = gc.compareLines
	gc.cmpTooltip = ui.NewTooltip(d2resource.FontFormal11, d2resource.PaletteStatic, d2ui.TooltipXCenter, d2ui.TooltipYBottom)
	gc.belt = NewBeltPanel(asset, ui, l, inventory, gc.beltBoxes, gc.saveHero)
	trade.autoBelt = func(it InventoryItem) bool { _, ok := gc.belt.AutoBelt(it); return ok }
	gc.wireTints()

	if !isSinglePlayer {
		PartyPanel := NewPartyPanel(asset, ui, hero.Name(), l, hero, hero.Stats, players)
		gc.PartyPanel = PartyPanel
	}

	hud := NewHUD(asset, ui, hero, miniPanel, actionableRegions, mapEngine, l, gc, mapRenderer)
	gc.hud = hud
	gc.automap = newAutomap(gc, term)

	hoverLabel := hud.nameLabel
	hoverLabel.SetBackgroundColor(d2util.Color(blackAlpha50percent))

	gc.heroStatsPanel.SetOnCloseCb(gc.onCloseHeroStatsPanel)
	gc.heroStatsPanel.SetHandSkills(gc.handSkillInfo)
	gc.mercPanel.SetOnCloseCb(gc.updateLayout)
	gc.stash.SetOnClose(gc.updateLayout) // closing the stash or cube moves the mini panel strip back
	gc.cube.SetOnClose(gc.updateLayout)
	gc.questLog.SetOnCloseCb(gc.onCloseQuestLog)
	gc.inventory.SetOnCloseCb(gc.onCloseInventory)
	gc.skilltree.SetOnCloseCb(gc.onCloseSkilltree)

	// skill selection: the popups pick through SelectSkill, the icons show the hotkeys
	hud.skillSelectMenu.SetCallbacks(gc.onSkillPopupPick, gc.hotkeyName)
	gc.skilltree.tooltipText = func(s *d2hero.HeroSkill) string {
		return skillTreeTooltip(asset, s, d2hero.EffectiveSkillLevel(gc.hero.Stats, gc.hero.Class, s), gc.hero.SkillBar, gc.hotkeyName, gc.hero.Skills, gc.hero.Stats)
	}

	if audioProvider != nil {
		if sfx, err := audioProvider.LoadSound(d2resource.SFXButtonClick, false, false); err == nil {
			gc.clickSfx = sfx
		}
	}

	gc.escapeMenu.SetOnCloseCb(gc.hud.miniPanel.restoreDisabled)
	gc.HelpOverlay.SetOnCloseCb(gc.hud.miniPanel.restoreDisabled)

	err = gc.bindTerminalCommands(term)
	if err != nil {
		return nil, err
	}

	gc.Logger = d2util.NewLogger()
	gc.Logger.SetLevel(l)
	gc.Logger.SetPrefix(logPrefix)

	return gc, nil
}

// GameControls represents the game's controls on the screen
type GameControls struct {
	keyMap            *KeyMap
	actionableRegions []actionableRegion
	asset             *d2asset.AssetManager
	renderer          d2interface.Renderer // https://github.com/OpenDiablo2/OpenDiablo2/issues/798
	inputListener     inputCallbackListener
	isFriendly        func(m *d2mapentity.Monster) bool // the hero's own units are not attacked by a click
	hero              *d2mapentity.Player
	heroState         *d2hero.HeroStateFactory
	mapRenderer       *d2maprenderer.MapRenderer
	escapeMenu        *EscapeMenu
	ui                *d2ui.UIManager
	inventory         *Inventory
	cmpCache          *compareCache // the comparison lines of the last hovered item (item_compare.go)
	cmpTooltip        *d2ui.Tooltip // the held item and the ground item under the pointer (item_compare.go)
	hud               *HUD
	questItemUse      func(code string) bool                   // Book of Skill, Potion of Life, Scroll of Resistance
	healShare         func(heal int) int                       // Blood Golem share of potion healing (SetHealShare)
	statePotion       func(p d2inventory.StatePotion) []string // antidote / thawing / stamina states (SetStatePotion)
	skilltree         *skillTree
	heroStatsPanel    *HeroStatsPanel
	mercPanel         *MercPanel
	mercPortrait      *MercPortrait
	PartyPanel        *PartyPanel
	questLog          *QuestLog
	HelpOverlay       *HelpOverlay
	NPCMenu           *NPCMenu
	HireList          *HireList
	Waypoints         *WaypointPanel
	Trade             *TradeWindow
	Identify          *IdentifyWindow
	// OnTownPortal is called when a scroll or tome of town portal is right clicked.
	OnTownPortal func(src *diablo2item.Item)
	PTrade       *PlayerTradeWindow // trade with another player
	relation     func(p *d2mapentity.Player) d2enum.PlayersRelationships
	stash        *ContainerPanel
	stashCoin    *stashCoin

	// OnDropGold leaves a gold pile of the amount on the ground at the hero (the drop gold dialog; set by the game screen).
	OnDropGold            func(amount int)
	cube                  *ContainerPanel
	cubeData              *cubeData
	cubePortal            func(kind string) error
	cubePortalCheck       func(kind string) error
	cubeRNG               *d2rand.Seed
	cubeClassic           bool
	cubeLadder            bool
	cubeLast              *TransmuteResult
	belt                  *BeltPanel
	itemOrigin            map[InventoryItem]*d2s.Item
	equipSound            func(handle string)
	heroVoice             func(id int)
	equipTouched          bool
	equipNoSave           bool // the equip autotest saves once at the end
	mercHost              MercGearHost
	equipRand             *rand.Rand
	equipStatus           map[d2equip.Loc]d2hero.EquipStatus
	regen                 d2inventory.Regen
	regenHP, regenMana    float64           // fractions of points not yet applied
	ammoCap               int               // stack size a replenishing thrown weapon regains up to (ammo_replenish.go)
	ammoAcc               float64           // seconds towards its next piece
	repair                repairState       // broken worn items and stat 252 timers (repair_restore.go)
	vitals                d2herostats.Regen // natural life/mana regeneration (natural_regen.go)
	bottomMenuRect        *d2geom.Rectangle
	leftMenuRect          *d2geom.Rectangle
	rightMenuRect         *d2geom.Rectangle
	lastMouseX            int
	lastMouseY            int
	lastLeftBtnActionTime float64
	watchLog              func(format string, args ...interface{})
	heroWatch             heroWatch                 // armed by the test-only heromoved/herostill commands (hero_watch.go)
	autoHold              *autoHold                 // a button held by an OD2_AUTOSCRIPT hold: step (synthetic_input.go)
	injector              d2interface.InputInjector // set by the game screen: scripted input goes through the input manager
	// heldLeftWalk is true while the left button is held down on a click that began as a plain
	// ground click (a walk or a skill use). Only such a hold repeats; a click that began on an NPC,
	// object or item interacts once.
	heldLeftWalk           bool
	kbd                    kbdState // keyboard cell cursor over the item grids (kbd_grid.go)
	lastRightBtnActionTime float64
	FreeCam                bool
	isSinglePlayer         bool
	mapEngine              *d2mapengine.MapEngine
	automap                *Automap                // see automap.go
	clickSfx               d2interface.SoundEffect // the button click (skill selection)

	// Speech shows the subtitle of NPC speech and short notices (quest log updated).
	Speech *SpeechBubble
	// OnSpeechSkip is called when Esc or Space cuts an NPC line (voice and subtitle); set by the game screen.
	OnSpeechSkip func()

	*d2util.Logger
}

type actionableType int

type actionableRegion struct {
	actionableTypeID actionableType
	rect             d2geom.Rectangle
}

// SkillResource represents a Skill with its corresponding icon sprite, path to DC6 file and icon number.
// SkillResourcePath points to a DC6 resource which contains the icons of multiple skills as frames.
// The IconNumber is the frame at which we can find our skill sprite in the DC6 file.
type SkillResource struct {
	SkillResourcePath string // path to a skills DC6 file(see getSkillResourceByClass)
	IconNumber        int    // the index of the frame in the DC6 file
	SkillIcon         *d2ui.Sprite
}

// OnKeyRepeat is called to handle repeated key presses
// IsEscapeMenuOpen says whether the game menu is showing.
func (g *GameControls) IsEscapeMenuOpen() bool { return g.escapeMenu != nil && g.escapeMenu.IsOpen() }

func (g *GameControls) OnKeyRepeat(event d2interface.KeyEvent) bool {
	if g.FreeCam {
		var moveSpeed float64 = 8
		if event.KeyMod() == d2enum.KeyModShift {
			moveSpeed *= 2
		}

		if event.Key() == d2enum.KeyDown {
			v := d2vector.NewVector(0, moveSpeed)
			g.mapRenderer.MoveCameraTargetBy(v)

			return true
		}

		if event.Key() == d2enum.KeyUp {
			v := d2vector.NewVector(0, -moveSpeed)
			g.mapRenderer.MoveCameraTargetBy(v)

			return true
		}

		if event.Key() == d2enum.KeyRight {
			v := d2vector.NewVector(moveSpeed, 0)
			g.mapRenderer.MoveCameraTargetBy(v)

			return true
		}

		if event.Key() == d2enum.KeyLeft {
			v := d2vector.NewVector(-moveSpeed, 0)
			g.mapRenderer.MoveCameraTargetBy(v)

			return true
		}
	}

	return false
}

// skipSpeechOnKey is the exe's speech key handler (input table 0x721940: keys 0x1b Esc and 0x20 Space ->
// handler 0x49de80, VERIFIED): while an NPC line is on screen the key is consumed, so Esc does not open the
// game menu nor Space clear the screen; after 100 ms it also cuts the line (voice and subtitle) and ends the
// talk. The next Esc then behaves as usual.
func (g *GameControls) skipSpeechOnKey(key d2enum.Key) bool {
	if (key != d2enum.KeyEscape && key != d2enum.KeySpace) || g.Speech == nil || !g.Speech.Active() {
		return false
	}

	if g.Speech.Skippable() {
		g.Speech.Clear()

		if g.OnSpeechSkip != nil {
			g.OnSpeechSkip()
		}

	}

	return true
}

// OnKeyDown handles key presses
func (g *GameControls) OnKeyDown(event d2interface.KeyEvent) bool {
	if event.Key() == d2enum.KeyEscape && g.Waypoints.IsOpen() {
		g.Waypoints.Close()
		return true
	}

	if g.HireList.IsOpen() && g.HireList.OnKeyDown(event) {
		return true
	}

	if event.Key() == d2enum.KeyEscape && g.NPCMenu.IsOpen() {
		g.NPCMenu.Choose(len(g.NPCMenu.Rows()) - 1)
		return true
	}

	if g.NPCMenu.IsOpen() && g.NPCMenu.OnKeyDown(event) {
		return true
	}

	if g.skipSpeechOnKey(event.Key()) {
		return true
	}

	if event.Key() == d2enum.KeyEscape && g.PTrade.IsOpen() {
		g.PTrade.Cancel()
		return true
	}

	if event.Key() == d2enum.KeyEscape && g.Identify.IsOpen() {
		g.Identify.Close()
		return true
	}

	if event.Key() == d2enum.KeyEscape && g.Trade.IsOpen() {
		g.Trade.Close()
		g.Waypoints.Close()
		return true
	}

	if event.Key() == d2enum.KeyEscape {
		g.onEscKey()
		return true
	}

	if event.Key() == d2enum.KeyO && event.KeyMod() == 0 {
		// the mercenary screen ("O" in the original's default keys)
		g.ToggleMercPanel()

		return true
	}

	if g.gridKeyNav(event) {
		return true
	}

	gameEvent := g.keyMap.getGameEvent(event.Key())

	switch gameEvent {
	case d2enum.ClearScreen:
		g.clearScreen()
		g.updateLayout()
	case d2enum.ToggleInventoryPanel:
		g.toggleInventoryPanel()
	case d2enum.TogglePartyPanel:
		if !g.isSinglePlayer {
			g.togglePartyPanel()
		}
	case d2enum.ToggleSkillTreePanel:
		g.toggleSkilltreePanel()
	case d2enum.ToggleCharacterPanel:
		g.toggleHeroStatsPanel()
	case d2enum.ToggleQuestLog:
		g.toggleQuestLog()
	case d2enum.ToggleRunWalk:
		g.hud.onToggleRunButton(false)
	case d2enum.HoldRun:
		g.hud.onToggleRunButton(true)
	case d2enum.ToggleHelpScreen:
		g.toggleHelpOverlay()
	case d2enum.SwapWeapons:
		g.SwapWeapons()
	case d2enum.ToggleBelts:
		g.belt.Toggle()
	case d2enum.ToggleAutomap:
		g.automap.Toggle()
	case d2enum.CenterAutomap:
		g.automap.ToggleOption("center")
	case d2enum.FadeAutomap:
		g.automap.ToggleOption("fade")
	case d2enum.TogglePartyOnAutomap:
		g.automap.ToggleOption("party")
	case d2enum.ToggleNamesOnAutomap:
		g.automap.ToggleOption("names")
	case d2enum.HoldShowGroundItems:
		g.hud.showItems = true
	case d2enum.UseNextPotion:
		g.useNextPotion()
	case d2enum.UseTownPortal:
		g.useTownPortalKey()
	case d2enum.SelectNextSkill:
		g.onWheelSkill(1, event.KeyMod())
	case d2enum.SelectPreviousSkill:
		g.onWheelSkill(-1, event.KeyMod())
	case d2enum.UseBeltSlot1, d2enum.UseBeltSlot2, d2enum.UseBeltSlot3, d2enum.UseBeltSlot4:
		g.UseBeltColumn(int(gameEvent - d2enum.UseBeltSlot1))
	case d2enum.UseSkill1, d2enum.UseSkill2, d2enum.UseSkill3, d2enum.UseSkill4,
		d2enum.UseSkill5, d2enum.UseSkill6, d2enum.UseSkill7, d2enum.UseSkill8,
		d2enum.UseSkill9, d2enum.UseSkill10, d2enum.UseSkill11, d2enum.UseSkill12,
		d2enum.UseSkill13, d2enum.UseSkill14, d2enum.UseSkill15, d2enum.UseSkill16:
		g.onSkillKey(int(gameEvent - firstHotkeyEvent))
	default:
		return false
	}

	return false
}

// OnKeyUp handles key release
func (g *GameControls) OnKeyUp(event d2interface.KeyEvent) bool {
	gameEvent := g.keyMap.getGameEvent(event.Key())

	if gameEvent == d2enum.HoldRun {
		g.hud.onToggleRunButton(true)
	}

	if gameEvent == d2enum.HoldShowGroundItems {
		g.hud.showItems = false
	}

	return false
}

// When escape is pressed:
// 1. If there was some overlay or panel open, close it
// 2. Otherwise, if the Escape Menu was open, let the Escape Menu handle it
// 3. If nothing was open, open the Escape Menu
func (g *GameControls) onEscKey() {
	escHandled := false

	escHandled = g.hasOpenPanels() || g.HelpOverlay.IsOpen() || g.hud.skillSelectMenu.IsOpen()
	g.clearScreen()

	if escHandled {
		g.updateLayout()
		return
	}

	if g.escapeMenu.IsOpen() {
		g.escapeMenu.OnEscKey()
	} else {
		g.openEscMenu()
	}
}

func truncateFloat64(n float64) float64 {
	const ten = 10.0
	return float64(int(n*ten)) / ten
}

// OnMouseButtonRepeat handles repeated mouse clicks
func (g *GameControls) OnMouseButtonRepeat(event d2interface.MouseEvent) bool {
	const subtilesPerTile = 5

	halfScreenWidth, halfScreenHeight := d2display.W()/2, d2display.H()/2

	px, py := g.mapRenderer.ScreenToWorld(d2display.ToScreen(event.X(), event.Y()))
	px = truncateFloat64(px)
	py = truncateFloat64(py)

	now := d2util.Now()
	button := g.effectiveButton(event)
	isLeft := button == d2enum.MouseButtonLeft
	isRight := button == d2enum.MouseButtonRight
	lastLeft := now - g.lastLeftBtnActionTime
	lastRight := now - g.lastRightBtnActionTime
	inRect := !g.isInActiveMenusRect(event.X(), event.Y())
	shouldDoRight := lastRight >= mouseBtnActionsThreshold

	if isLeft && !LeftHoldRepeats(g.heldLeftWalk, g.inventory.CursorItem() != nil, lastLeft) {
		return true
	}

	// a held button repeating is not a fresh click: while a cast runs it waits (a held Inferno must not restart
	// its run on every repeat); only a new button press reaches the skill engine's interrupt rule
	if isLeft && inRect && !g.hero.IsCasting() {
		g.lastLeftBtnActionTime = now

		g.worldClick(button, event.KeyMod(), px, py)

		if g.FreeCam {
			camVect := g.mapRenderer.Camera.GetPosition().Vector

			x := float64(halfScreenWidth) / subtilesPerTile
			y := float64(halfScreenHeight) / subtilesPerTile

			targetPosition := d2vector.NewPositionTile(x, y)
			targetPosition.Add(&camVect)

			g.mapRenderer.SetCameraTarget(&targetPosition)
		}

		return true
	}

	if isRight && shouldDoRight && inRect && !g.hero.IsCasting() {
		g.lastRightBtnActionTime = now

		g.worldClick(button, event.KeyMod(), px, py)

		return true
	}

	return true
}

// effectiveButton is the button a click counts as (Control+click is the right
// button on macOS) - except while an item is on the cursor, when Control+click
// still drops it.
func (g *GameControls) effectiveButton(event d2interface.MouseEvent) d2enum.MouseButton {
	if g.inventory.CursorItem() != nil {
		return event.Button()
	}

	return EffectiveButton(event.Button(), event.KeyMod(), runtime.GOOS)
}

// worldClickInput describes a click on the world for ResolveWorldClick.
func (g *GameControls) worldClickInput(button d2enum.MouseButton, mod d2enum.KeyMod) WorldClickInput {
	in := WorldClickInput{Button: button, Mod: mod, OverMonster: g.hoveredMonster() != nil}
	if g.hero.LeftSkill != nil {
		in.LeftSkillID = g.hero.LeftSkill.ID
		in.LeftSkillInTown = g.hero.LeftSkill.SkillRecord != nil && g.hero.LeftSkill.SkillRecord.InTown
	}

	in.InTown = g.hero.IsInTown()

	return in
}

// ignoredWhileCasting says whether a click is dropped because the hero is busy with a cast or swing. The exe
// ignores such orders, it does not queue them: the walk, run and interact packets reach PLRMODE_ServerStartMode
// 0x57e9b0, which asks PLRMODE_CanInterruptCurrentMode 0x57ceb0, and a walk mode is never in the list of modes that
// may break a running cast (VERIFIED: attack, cast, throw and the skill modes 7, 8, 10, 0xb, 0xd, 0x12, only near the
// action frame). A skill click is passed on: the skill engine applies that rule itself (d2skills mayInterrupt) and
// refuses it when the cast cannot be broken (still not queued).
func (g *GameControls) ignoredWhileCasting(button d2enum.MouseButton, mod d2enum.KeyMod) bool {
	if !g.hero.IsCasting() {
		return false
	}

	act := ResolveWorldClick(g.worldClickInput(button, mod))
	if !IgnoredWhileCasting(act) {
		return false
	}

	g.Infof("INPUT click ignored while casting: button=%d action=%s (the exe drops it, it does not queue it)", button, act)

	return true
}

// worldClick performs a click on the game world (see ResolveWorldClick for the rules).
func (g *GameControls) worldClick(button d2enum.MouseButton, mod d2enum.KeyMod, px, py float64) {
	in := g.worldClickInput(button, mod)
	act := ResolveWorldClick(in)

	hsx, hsy := g.mapRenderer.WorldToScreen(g.hero.GetPositionF())
	hw := g.hero.Position.World()
	g.Infof("INPUT world-pick world=(%.2f,%.2f) hero=(%.2f,%.2f) hero_screen=(%d,%d)", px, py, hw.X(), hw.Y(), hsx, hsy)
	if button == d2enum.MouseButtonLeft && act == WorldCastLeft && d2gamepad.Default().Walking() {
		act = WorldMove // the left stick walks, it does not cast the left skill
	}

	g.Infof("INPUT world-click button=%d mod=%d action=%s left_skill=%d", button, mod, act, in.LeftSkillID)

	g.watchLogf("WATCH world-click action=%s", act)

	switch act {
	case WorldAttack:
		g.inputListener.OnPlayerAttack(g.hoveredMonster())
	case WorldMove:
		g.inputListener.OnPlayerMove(px, py)
	case WorldCastLeft, WorldStandStill:
		g.UseActiveSkill(true, px, py)
	case WorldCastRight:
		g.UseActiveSkill(false, px, py)
	case WorldNone:
	}
}

// OnMouseMove handles mouse movement events
func (g *GameControls) OnMouseMove(event d2interface.MouseMoveEvent) bool {
	mx, my := event.X(), event.Y()
	g.lastMouseX = mx
	g.lastMouseY = my
	g.inventory.lastMouseX = mx
	g.inventory.lastMouseY = my

	for i := range g.actionableRegions {
		// Mouse over a game control element
		if g.actionableRegions[i].rect.IsInRect(mx, my) {
			g.onHoverActionable(g.actionableRegions[i].actionableTypeID)
		}
	}

	g.NPCMenu.OnMouseMove(event)
	g.HireList.OnMouseMove(event)
	g.Waypoints.OnMouseMove(event)
	g.Trade.OnMouseMove(event)
	g.PTrade.OnMouseMove(event)
	g.Identify.OnMouseMove(event)
	g.stash.OnMouseMove(mx, my)
	g.cube.OnMouseMove(mx, my)
	g.belt.OnMouseMove(mx, my)
	g.hud.OnMouseMove(event)
	g.skilltree.OnMouseMove(mx, my)

	if g.PartyPanel != nil {
		g.PartyPanel.OnMouseMove(event)
	}

	return false
}

// OnMouseButtonUp handles mouse button presses
func (g *GameControls) OnMouseButtonUp(event d2interface.MouseEvent) bool {
	if event.Button() == d2enum.MouseButtonLeft {
		g.heldLeftWalk = false
	}

	return false
}

// hoveredNPC returns the named NPC under the cursor, if any.
func (g *GameControls) hoveredNPC() d2interface.MapEntity {
	if g.hud == nil || g.hud.hoveredEntity == nil {
		return nil
	}

	if _, ok := g.hud.hoveredEntity.(*d2mapentity.NPC); !ok {
		return nil
	}

	return g.hud.hoveredEntity
}

// hoveredWorldThing returns the ground item or object under the cursor, if any.
func (g *GameControls) hoveredWorldThing() d2interface.MapEntity {
	if g.hud == nil || g.hud.hoveredEntity == nil {
		return nil
	}

	switch g.hud.hoveredEntity.(type) {
	case *d2mapentity.Item, *d2mapentity.Object:
		return g.hud.hoveredEntity
	}

	return nil
}

// hoveredMonster returns the living monster under the cursor, if any.
func (g *GameControls) hoveredMonster() *d2mapentity.Monster {
	if g.hud == nil || g.hud.hoveredEntity == nil {
		return nil
	}

	if m, ok := g.hud.hoveredEntity.(*d2mapentity.Monster); ok && m.Alive() && (g.isFriendly == nil || !g.isFriendly(m)) {
		return m
	}

	return nil
}

// CursorItem returns the item the hero holds on the cursor, or nil.
func (g *GameControls) CursorItem() InventoryItem { return g.inventory.CursorItem() }

// SetCursorItem puts an item on the cursor.
func (g *GameControls) SetCursorItem(item InventoryItem) { g.inventory.SetCursorItem(item) }

// AddGold adds gold to the hero.
func (g *GameControls) AddGold(amount int) {
	g.hero.Gold += amount
	g.inventory.AddGold(amount)
}

// PickUpGold adds picked-up gold to the purse up to the carry cap of the
// hero's level (d2inventory.InventoryGoldLimit, PLAYER_GetMaxGoldCarry
// 0x623050) and returns the overflow, which the caller leaves on the ground
// (0x558e40 drops it as piles; VERIFIED). Gameplay change: a pickup can no
// longer take the purse past level*10000.
func (g *GameControls) PickUpGold(amount int) (overflow int) {
	level := 1
	if g.hero.Stats != nil {
		level = g.hero.Stats.Level
	}

	total, over := d2inventory.AddGold(g.hero.Gold, amount, d2inventory.InventoryGoldLimit(level))
	g.AddGold(total - g.hero.Gold)

	return over
}

// InventoryItemCount returns how many items are in the inventory grid.
func (g *GameControls) InventoryItemCount() int { return len(g.inventory.grid.items) }

// OnMouseButtonDown handles mouse button presses
func (g *GameControls) OnMouseButtonDown(event d2interface.MouseEvent) bool {
	mx, my := event.X(), event.Y()

	g.heroStatsPanel.SetClickMod(event.KeyMod())

	if event.Button() == d2enum.MouseButtonLeft {
		g.heldLeftWalk = false
	}

	if g.Waypoints.OnMouseButtonDown(event) {
		return true
	}

	if g.HireList.OnMouseButtonDown(event) {
		return true
	}

	if g.NPCMenu.OnMouseButtonDown(event) {
		return true
	}

	if g.PTrade.OnMouseButtonDown(event) {
		return true
	}

	if g.Identify.OnMouseButtonDown(event) {
		return true
	}

	if g.Trade.OnMouseButtonDown(event) {
		return true
	}

	for i := range g.actionableRegions {
		// If click is on a game control element
		if g.actionableRegions[i].rect.IsInRect(mx, my) {
			g.onClickActionable(g.actionableRegions[i].actionableTypeID)
			return false
		}
	}

	if g.hud.skillSelectMenu.IsOpen() && event.Button() == d2enum.MouseButtonLeft {
		g.lastLeftBtnActionTime = d2util.Now()
		g.hud.skillSelectMenu.HandleClick(mx, my)
		g.hud.skillSelectMenu.ClosePanels()

		return false
	}

	if g.skilltree.IsOpen() && g.skillTreeClick(event) {
		return true
	}

	px, py := g.mapRenderer.ScreenToWorld(d2display.ToScreen(mx, my))
	px = truncateFloat64(px)
	py = truncateFloat64(py)

	if (event.Button() == d2enum.MouseButtonLeft || event.Button() == d2enum.MouseButtonRight) &&
		g.mercPortraitClick(event.Button() == d2enum.MouseButtonLeft, mx, my) {
		g.lastLeftBtnActionTime = d2util.Now()

		return true
	}

	if event.Button() == d2enum.MouseButtonLeft && g.mercPanelClick(mx, my) {
		g.lastLeftBtnActionTime = d2util.Now()

		return true
	}

	if event.Button() == d2enum.MouseButtonLeft && g.shiftClickToBelt(event.KeyMod(), mx, my) {
		g.lastLeftBtnActionTime = d2util.Now()

		return true
	}

	if event.Button() == d2enum.MouseButtonLeft && g.handleContainerClick(mx, my, event.KeyMod() == d2enum.KeyModControl) {
		g.lastLeftBtnActionTime = d2util.Now()

		return true
	}

	if event.Button() == d2enum.MouseButtonRight && g.handleContainerRightClick(mx, my) {
		g.lastRightBtnActionTime = d2util.Now()

		return true
	}

	button := g.effectiveButton(event)
	standStill := event.KeyMod()&d2enum.KeyModShift != 0

	if button == d2enum.MouseButtonLeft && !g.isInActiveMenusRect(mx, my) && g.inventory.CursorItem() != nil {
		// clicking the world with an item on the cursor drops it (packet 0x17)
		g.lastLeftBtnActionTime = d2util.Now()

		// an NPC that is owed a reward on an item (Larzuk, Anya, Charsi) takes it
		if g.dropOnNPC(g.hoveredNPC()) {
			return true
		}

		item := g.inventory.CursorItem()
		g.inventory.SetCursorItem(nil)
		g.inputListener.OnPlayerDropItem(item)

		return true
	}

	if button == d2enum.MouseButtonLeft && !g.isInActiveMenusRect(mx, my) && !g.ignoredWhileCasting(button, event.KeyMod()) {
		g.lastLeftBtnActionTime = d2util.Now()
		g.heldLeftWalk = false

		// talking to an NPC or using an object walks there: ignored while a cast runs, like any walk order
		if npc := g.hoveredNPC(); npc != nil && !standStill {
			if !g.hero.IsCasting() {
				g.inputListener.OnPlayerInteract(npc)
			}

			return true
		}

		if thing := g.hoveredWorldThing(); thing != nil && !standStill {
			if !g.hero.IsCasting() {
				g.inputListener.OnPlayerInteract(thing)
			}

			return true
		}

		g.heldLeftWalk = true

		g.worldClick(button, event.KeyMod(), px, py)

		return true
	}

	if button == d2enum.MouseButtonRight && !g.isInActiveMenusRect(mx, my) && !g.ignoredWhileCasting(button, event.KeyMod()) {
		g.lastRightBtnActionTime = d2util.Now()

		g.worldClick(button, event.KeyMod(), px, py)

		return true
	}

	return false
}

func (g *GameControls) clearLeftScreenSide() {
	g.heroStatsPanel.Close()
	g.mercPanel.Close()

	if g.PartyPanel != nil {
		g.PartyPanel.Close()
	}

	g.questLog.Close()
	g.stash.Close()
	g.cube.Close()
	g.hud.skillSelectMenu.ClosePanels()
	g.updateLayout()
}

func (g *GameControls) clearRightScreenSide() {
	g.inventory.Close()
	g.skilltree.Close()
	g.hud.skillSelectMenu.ClosePanels()
	g.updateLayout()
}

func (g *GameControls) clearScreen() {
	g.clearRightScreenSide()
	g.clearLeftScreenSide()
	g.hud.skillSelectMenu.ClosePanels()
	g.HelpOverlay.Close()
}

func (g *GameControls) openLeftPanel(panel Panel) {
	if !g.HelpOverlay.IsOpen() && !g.escapeMenu.IsOpen() {
		isOpen := panel.IsOpen()

		g.clearLeftScreenSide()

		if !isOpen {
			panel.Open()
			g.updateLayout()
		}
	}
}

func (g *GameControls) openRightPanel(panel Panel) {
	if !g.HelpOverlay.IsOpen() && !g.escapeMenu.IsOpen() {
		isOpen := panel.IsOpen()

		g.clearRightScreenSide()

		if !isOpen {
			panel.Open()
			g.updateLayout()
		}
	}
}

func (g *GameControls) toggleHeroStatsPanel() {
	g.openLeftPanel(g.heroStatsPanel)
}

func (g *GameControls) togglePartyPanel() {
	g.openLeftPanel(g.PartyPanel)
}

func (g *GameControls) onCloseHeroStatsPanel() {
	g.updateLayout()
}

func (g *GameControls) toggleLeftSkillPanel() {
	if !g.HelpOverlay.IsOpen() {
		// clearScreen closes the popups too: remember whether this one was open, or the click that should close it reopens it
		wasOpen := g.hud.skillSelectMenu.LeftPanel.IsOpen()

		g.clearScreen()

		if !wasOpen {
			g.hud.skillSelectMenu.OpenLeftPanel()
		}
	}
}

func (g *GameControls) toggleRightSkillPanel() {
	if !g.HelpOverlay.IsOpen() {
		// clearScreen closes the popups too: remember whether this one was open, or the click that should close it reopens it
		wasOpen := g.hud.skillSelectMenu.RightPanel.IsOpen()

		g.clearScreen()

		if !wasOpen {
			g.hud.skillSelectMenu.OpenRightPanel()
		}
	}
}

func (g *GameControls) toggleQuestLog() {
	g.openLeftPanel(g.questLog)
}

func (g *GameControls) onCloseQuestLog() {
	g.updateLayout()
}

func (g *GameControls) toggleHelpOverlay() {
	if !g.isRightPanelOpen() || g.isLeftPanelOpen() {
		g.HelpOverlay.updateKeyMap(g.keyMap)
		g.hud.skillSelectMenu.ClosePanels()
		g.hud.miniPanel.openDisabled()
		g.HelpOverlay.Toggle()
		g.updateLayout()
	}
}

// AutoPanel opens (or closes) a panel by name for OD2_AUTOSCRIPT: inventory,
// character, skills, quest or close. Opening an already-open panel is a no-op.
func (g *GameControls) AutoPanel(name string) error {
	var panel Panel

	switch name {
	case "close":
		g.clearScreen()
		g.NPCMenu.Close()
		g.HireList.Close()
		g.Trade.Close()

		if g.inventory.moveGoldPanel.IsOpen() {
			g.inventory.moveGoldPanel.Close()
		}

		g.updateLayout()

		return nil
	case "trade":
		v, ok := d2vendor.ByName("Charsi")
		if !ok {
			return fmt.Errorf("no vendor Charsi")
		}

		g.OpenTrade(v, 1)

		return nil
	case "npcmenu":
		// Akara's menu (Talk, Trade) for the layout run, anchored where an NPC at (400, 300) puts it
		rows, _ := NPCMenuFor(148)
		g.NPCMenu.Open("Akara", rows, 400, 300-0x96, nil)

		return nil
	case "golddlg":
		g.inventory.moveGoldPanel.Open()

		return nil
	case "inventory":
		panel = g.inventory
	case "party":
		if g.PartyPanel == nil {
			return fmt.Errorf("no party panel in a single player game")
		}

		panel = g.PartyPanel
	case "stash":
		g.OpenStash()

		return nil
	case "cube":
		g.OpenCube()

		return nil
	case "belt":
		g.belt.SetExpanded(true)

		return nil
	case "character":
		panel = g.heroStatsPanel
	case "merc":
		panel = g.mercPanel
	case "skills":
		panel = g.skilltree
	case "quest":
		panel = g.questLog
	default:
		return fmt.Errorf("unknown panel %q", name)
	}

	if !panel.IsOpen() {
		if name == "inventory" || name == "skills" {
			g.openRightPanel(panel)
		} else {
			g.openLeftPanel(panel)
		}
	}

	if !panel.IsOpen() {
		// openLeftPanel/openRightPanel do nothing while the help overlay or the escape menu is showing: say which
		return fmt.Errorf("panel %q did not open (help=%t escape_menu=%t)", name, g.HelpOverlay.IsOpen(),
			g.escapeMenu.IsOpen())
	}

	if name == "character" {
		// the values the panel shows, for the autotests (scripts/verify.d/89-hero-stats.sh)
		g.heroStatsPanel.setDerivedValues()
		g.Infof("PANEL character: %s sim=%.2f", d2hero.StatsSummary(g.hero.Stats), d2util.SimSeconds())
	}

	g.logPanel(name)

	return nil
}

// logPanel writes the values an open panel shows as a "PANEL <name>: ..." log
// line, so OD2_AUTOSCRIPT runs can be checked without a screenshot.
func (g *GameControls) logPanel(name string) {
	switch name {
	case "character":
		g.Infof("PANEL character: %s", g.heroStatsPanel.Summary())
	case "skills":
		g.Infof("PANEL skills: %s", g.skilltree.Summary())
		g.Infof("PANEL skills active: left=%s right=%s", skillLabel(g.hero.LeftSkill), skillLabel(g.hero.RightSkill))
	case "inventory":
		g.Infof("PANEL inventory: gold=%d items=%d worn=[%s] equipment=[%s]", g.inventory.Gold(),
			len(g.inventory.grid.items), g.inventory.EquippedSummary(), g.equipmentSummary())
	}
}

// handSkillInfo describes the left and right skills for the character sheet boxes.
func (g *GameControls) handSkillInfo() [2]HandSkillInfo {
	var out [2]HandSkillInfo

	for i, sk := range []*d2hero.HeroSkill{g.hero.LeftSkill, g.hero.RightSkill} {
		if sk == nil || sk.SkillRecord == nil {
			continue
		}

		if sk.SkillDescriptionRecord != nil {
			out[i].Name = g.asset.TranslateString(sk.SkillDescriptionRecord.NameKey)
		}

		if out[i].Name == "" {
			out[i].Name = sk.Skill
		}

		switch sk.Anim {
		case d2enum.PlayerAnimationModeAttack1, d2enum.PlayerAnimationModeAttack2, d2enum.PlayerAnimationModeThrow:
			out[i].Weapon = true
		}

		out[i].Thrown = sk.Anim == d2enum.PlayerAnimationModeThrow
		out[i].SrcDam = sk.SkillRecord.SrcDam

		// SKILL_GetToHitBonus 0x645da0: ToHit + (level-1) * LevToHit (the ToHitCalc column is not evaluated here)
		if lvl := sk.SkillPoints; lvl > 0 && sk.SkillRecord.ToHitCalc == nil {
			out[i].ToHitBonus = sk.SkillRecord.ToHit + (lvl-1)*sk.SkillRecord.LevToHit
		}
	}

	return out
}

func skillLabel(s *d2hero.HeroSkill) string {
	if s == nil || s.SkillRecord == nil {
		return "none"
	}

	return fmt.Sprintf("%s(id=%d,lvl=%d)", s.Skill, s.ID, s.SkillPoints)
}

func (g *GameControls) equipmentSummary() string { return g.hero.Equipment.Describe() }

func (g *GameControls) toggleInventoryPanel() {
	g.openRightPanel(g.inventory)
}

func (g *GameControls) onCloseInventory() {
	g.updateLayout()
}

func (g *GameControls) toggleSkilltreePanel() {
	g.openRightPanel(g.skilltree)
}

func (g *GameControls) onCloseSkilltree() {
	g.updateLayout()
}

func (g *GameControls) openEscMenu() {
	g.clearScreen()
	g.hud.miniPanel.closeDisabled()
	g.escapeMenu.open()
	g.updateLayout()
}

// Load the resources required for the GameControls
func (g *GameControls) Load() {
	g.hud.Load()
	g.inventory.Load()
	g.Trade.Load()
	g.stash.Load()
	g.cube.Load()
	g.belt.Load()
	// The interface widgets (the bar picture with the four potion boxes) are drawn after the screen, so the potions
	// must be a widget of the same group to be on top of the boxes (drawn in the screen pass they were hidden under them).
	g.hud.panelGroup.AddWidget(g.ui.NewCustomWidget(g.belt.Render, screenWidth, screenHeight))
	g.skilltree.load()
	g.heroStatsPanel.Load()
	g.mercPanel.Load()
	g.loadContainers()

	if g.PartyPanel != nil {
		g.PartyPanel.Load()
	}

	g.questLog.Load()
	g.HelpOverlay.Load()

	g.loadAddButtons()
	g.setAddButtons()

	miniPanelActions := &miniPanelActions{
		characterToggle: g.toggleHeroStatsPanel,
		partyToggle:     g.togglePartyPanel,
		inventoryToggle: g.toggleInventoryPanel,
		skilltreeToggle: g.toggleSkilltreePanel,
		menuToggle:      g.openEscMenu,
		questToggle:     g.toggleQuestLog,
	}
	g.hud.miniPanel.load(miniPanelActions)
	g.logDisplay()
}

// logDisplay logs the display model once the interface is loaded (docs/DISPLAY.md): the logical screen, the
// origin of the 800x600 interface column and the span of the bottom bar, for the display scenarios.
func (g *GameControls) logDisplay() {
	sz := d2display.Get()
	ox, oy := d2display.ColumnOrigin()
	hx, hy := g.mapRenderer.WorldToScreen(g.hero.GetPositionF())

	g.Infof("DISPLAY size=%dx%d scale=%d column=(%d,%d) hud_x=%d..%d hud_bottom=%d hero_screen=(%d,%d)",
		sz.W, sz.H, d2display.Scale(), ox, oy, ox, ox+d2display.BaseW, oy+d2display.BaseH, hx, hy)
}

// Advance advances the state of the GameControls
func (g *GameControls) Advance(elapsed float64) error {
	g.advanceHeroWatch(elapsed)

	g.mapRenderer.Advance(elapsed)
	g.hud.Advance(elapsed)
	g.inventory.Advance(elapsed)
	g.advanceStashCoin()
	g.advancePotions(elapsed)
	g.advanceNaturalRegen(elapsed)
	g.advanceAmmo(elapsed)
	g.advanceRepair(elapsed)
	g.automap.Advance(elapsed)
	g.questLog.Advance(elapsed)
	g.mercPanel.Advance(elapsed)
	g.Speech.Advance(elapsed)

	if g.PartyPanel != nil {
		g.PartyPanel.Advance(elapsed)
	}

	if err := g.escapeMenu.Advance(elapsed); err != nil {
		return err
	}

	if g.heroStatsPanel.IsOpen() || g.skilltree.IsOpen() {
		g.setAddButtons()
	}

	if g.skilltree.IsOpen() {
		g.skilltree.refresh()
	}

	return nil
}

func (g *GameControls) updateLayout() {
	// the strip follows the exe's panel groups (the stash, cube and trade panels count for both sides)
	g.hud.miniPanel.Place(MiniPanelPlacement(g.panelGroups()))

	leftOpen := g.isLeftPanelOpen()
	rightOpen := g.isRightPanelOpen()

	switch {
	case leftOpen == rightOpen:
		g.mapRenderer.ViewportDefault()
	case leftOpen:
		g.mapRenderer.ViewportToLeft()
	case rightOpen:
		g.mapRenderer.ViewportToRight()
	}
}

func (g *GameControls) isLeftPanelOpen() bool {
	var partyPanel bool

	if g.PartyPanel != nil {
		partyPanel = g.PartyPanel.IsOpen()
	} else {
		partyPanel = false
	}

	return g.heroStatsPanel.IsOpen() || g.mercPanel.IsOpen() || partyPanel || g.questLog.IsOpen() || g.inventory.moveGoldPanel.IsOpen() || g.Trade.IsOpen() || g.Identify.IsOpen() ||
		g.stash.IsOpen() || g.cube.IsOpen()
}

func (g *GameControls) isRightPanelOpen() bool {
	return g.inventory.IsOpen() || g.skilltree.IsOpen()
}

func (g *GameControls) hasOpenPanels() bool {
	return g.isRightPanelOpen() || g.isLeftPanelOpen() || g.hud.skillSelectMenu.IsOpen()
}

func (g *GameControls) isInActiveMenusRect(px, py int) bool {
	if g.bottomMenuRect.IsInRect(px, py) {
		return true
	}

	if g.isLeftPanelOpen() && g.leftMenuRect.IsInRect(px, py) {
		return true
	}

	if g.isRightPanelOpen() && g.rightMenuRect.IsInRect(px, py) {
		return true
	}

	if g.hud.miniPanel.IsOpen() && g.hud.miniPanel.IsInRect(px, py) {
		return true
	}

	if g.escapeMenu.IsOpen() {
		return true
	}

	if g.HelpOverlay.IsOpen() && g.HelpOverlay.IsInRect(px, py) {
		return true
	}

	if g.hud.skillSelectMenu.IsOpen() {
		return true
	}

	return false
}

// Render draws the GameControls onto the target
func (g *GameControls) Render(target d2interface.Surface) error {
	// the automap covers the whole screen, not the interface column
	ox, oy := d2display.ColumnOrigin()
	target.PushTranslation(-ox, -oy)
	g.automap.Render(target) // before the interface, as in the original
	target.Pop()

	if err := g.hud.Render(target); err != nil {
		return err
	}

	g.updateCompareTooltip()

	if err := g.renderPanels(target); err != nil {
		return err
	}

	g.renderMercPortrait(target)

	g.Trade.Render(target)
	g.PTrade.Render(target)
	g.Identify.Render(target)
	g.stash.Render(target)
	g.cube.Render(target)
	g.NPCMenu.Render(target)
	g.HireList.Render(target)
	g.Speech.Render(target)
	g.Waypoints.Render(target)

	if err := g.escapeMenu.Render(target); err != nil {
		return err
	}

	g.renderKbdCursor(target)
	g.inventory.RenderCursorItem(target, g.lastMouseX, g.lastMouseY)

	return nil
}

func (g *GameControls) renderPanels(target d2interface.Surface) error {
	g.inventory.Render(target)

	return nil
}

// QuestLog returns the quest log panel, so the quest system can feed it the
// real quest states.
func (g *GameControls) QuestLog() *QuestLog { return g.questLog }

// SetZoneChangeText sets the zoneChangeText
func (g *GameControls) SetZoneChangeText(text string) {
	g.hud.zoneChangeText.SetText(text)
}

// ShowZoneChangeText shows the zoneChangeText
func (g *GameControls) ShowZoneChangeText() {
	g.hud.isZoneTextShown = true
}

// HideZoneChangeTextAfter hides the zoneChangeText after the given amount of seconds
func (g *GameControls) HideZoneChangeTextAfter(delay float64) {
	time.AfterFunc(time.Duration(delay)*time.Second, func() {
		g.hud.isZoneTextShown = false
	})
}

// HpStatsIsVisible returns true if the hp and mana stats are visible to the player
func (g *GameControls) HpStatsIsVisible() bool {
	return g.hud.hpStatsIsVisible
}

// ManaStatsIsVisible returns true if the hp and mana stats are visible to the player
func (g *GameControls) ManaStatsIsVisible() bool {
	return g.hud.manaStatsIsVisible
}

// ToggleHpStats toggles the visibility of the hp and mana stats placed above their respective globe and load only if they do not match
func (g *GameControls) ToggleHpStats() {
	g.hud.hpStatsIsVisible = !g.hud.hpStatsIsVisible
}

// ToggleManaStats toggles the visibility of the hp and mana stats placed above their respective globe
func (g *GameControls) ToggleManaStats() {
	g.hud.manaStatsIsVisible = !g.hud.manaStatsIsVisible
}

// Handles what to do when an actionable is hovered
func (g *GameControls) onHoverActionable(item actionableType) {
	hoverMap := map[actionableType]func(){
		leftSkill:  func() {},
		xp:         func() {},
		stamina:    func() {},
		rightSkill: func() {},
		hpGlobe:    func() {},
		manaGlobe:  func() {},
	}

	onHover, found := hoverMap[item]
	if !found {
		g.Errorf("Unrecognized actionableType(%d) being hovered", item)
		return
	}

	onHover()
}

// Handles what to do when an actionable is clicked
func (g *GameControls) onClickActionable(item actionableType) {
	actionMap := map[actionableType]func(){
		leftSkill: func() {
			g.toggleLeftSkillPanel()
		},

		xp: func() {
			g.Info("XP Action Pressed")
		},

		stamina: func() {
			g.Info("Stamina Action Pressed")
		},

		rightSkill: func() {
			g.toggleRightSkillPanel()
		},

		hpGlobe: func() {
			g.ToggleHpStats()
			g.Info("HP Globe Pressed")
		},

		manaGlobe: func() {
			g.ToggleManaStats()
			g.Info("Mana Globe Pressed")
		},
	}

	action, found := actionMap[item]
	if !found {
		// Warning, because some action types are still todo, and could return this error
		g.Warningf("Unrecognized actionableType(%d) being clicked", item)
		return
	}

	action()
}

func (g *GameControls) bindTerminalCommands(term d2interface.Terminal) error {
	if err := term.Bind("freecam", "toggle free camera movement", nil, g.commandFreeCam); err != nil {
		return err
	}

	if err := term.Bind("bindkey", "bind a key to a game event and save it, e.g. bindkey ToggleInventoryPanel X",
		[]string{"event", "key"}, g.commandBindKey(term)); err != nil {
		return err
	}

	// test-only: puts the keyboard cell cursor on the first item of its grid (scenarios cannot know where items lie)
	if err := term.Bind("kbdfind", "move the keyboard grid cursor to the first item of its grid, for scenarios", nil,
		func([]string) error { g.kbdFind(); return nil }); err != nil {
		return err
	}

	// test-only: logs the hero position so OD2_AUTOSCRIPT scenarios can assert that a click walked (or did not)
	if err := term.Bind("heropos", "log the hero's world position (HERO pos=...), for scenarios", nil, func([]string) error {
		p := g.hero.Position.World()
		g.Infof("HERO pos=(%.2f,%.2f) town=%t", p.X(), p.Y(), g.hero.IsInTown())

		return nil
	}); err != nil {
		return err
	}

	// test-only: arm a one-shot watch that logs "HERO moved"/"HERO still" (see hero_watch.go)
	for name, kind := range map[string]heroWatchKind{"heromoved": heroWatchMoved, "herostill": heroWatchStill} {
		kind := kind

		if err := term.Bind(name, "arm a one-shot hero motion log line, for scenarios", nil, func([]string) error {
			p := g.hero.Position.World()
			g.heroWatch.arm(kind, p.X(), p.Y())

			return nil
		}); err != nil {
			return err
		}
	}

	if err := term.Bind("setleftskill", "set skill to fire on left click", []string{"id"}, g.commandSetLeftSkill(term)); err != nil {
		return err
	}

	if err := term.Bind("setrightskill", "set skill to fire on right click", []string{"id"}, g.commandSetRightSkill(term)); err != nil {
		return err
	}

	if err := term.Bind("learnskills", "learn all skills for the a given class", []string{"token"}, g.commandLearnSkills(term)); err != nil {
		return err
	}

	if err := term.Bind("learnskillid", "learn a skill by a given ID", []string{"id"}, g.commandLearnSkillID(term)); err != nil {
		return err
	}

	if err := term.Bind("levelup", "give the hero level-ups (skill and stat points)", []string{"levels"}, g.commandLevelUp(term)); err != nil {
		return err
	}

	return nil
}

// UnbindTerminalCommands unbinds commands from the terminal
func (g *GameControls) UnbindTerminalCommands(term d2interface.Terminal) error {
	return term.Unbind("freecam", "setleftskill", "setrightskill", "learnskills", "learnskillid", "levelup")
}

func (g *GameControls) setAddButtons() {
	g.hud.addStatsButton.SetEnabled(g.hero.Stats.StatsPoints > 0)
	g.hud.addSkillButton.SetEnabled(g.hero.Stats.SkillPoints > 0)
}

func (g *GameControls) loadAddButtons() {
	g.hud.addStatsButton.OnActivated(func() { g.toggleHeroStatsPanel() })
	g.hud.addSkillButton.OnActivated(func() { g.toggleSkilltreePanel() })
}

func (g *GameControls) commandFreeCam([]string) error {
	g.FreeCam = !g.FreeCam

	return nil
}

func (g *GameControls) commandSetLeftSkill(term d2interface.Terminal) func(args []string) error {
	return func(args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			term.Errorf("invalid argument")
			return nil
		}

		skill, err := g.heroSkillByID(id)
		if err != nil {
			term.Errorf(err.Error())
			return nil
		}

		g.hero.LeftSkill = skill

		return nil
	}
}

func (g *GameControls) commandSetRightSkill(term d2interface.Terminal) func(args []string) error {
	return func(args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			term.Errorf("invalid argument")
			return nil
		}

		skill, err := g.heroSkillByID(id)
		if err != nil {
			term.Errorf(err.Error())
			return nil
		}

		g.hero.RightSkill = skill

		return nil
	}
}

func (g *GameControls) commandLearnSkillID(term d2interface.Terminal) func(args []string) error {
	return func(args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			term.Errorf("invalid argument")
			return nil
		}

		skill, err := g.heroSkillByID(id)
		if err != nil {
			term.Errorf(err.Error())
			return nil
		}

		g.hero.Skills[skill.ID] = skill
		g.hud.skillSelectMenu.RegenerateImageCache()
		g.Infof("Learned skill: %s", skill.Skill)

		return nil
	}
}

func (g *GameControls) heroSkillByID(id int) (*d2hero.HeroSkill, error) {
	skillRecord := g.asset.Records.Skill.Details[id]
	if skillRecord == nil {
		return nil, fmt.Errorf("cannot find a skill record for ID: %d", id)
	}

	skill, err := g.heroState.CreateHeroSkill(1, skillRecord.Skill)
	if err != nil {
		return nil, fmt.Errorf("cannot create skill with ID of %d", id)
	}

	return skill, nil
}

func (g *GameControls) commandLearnSkills(term d2interface.Terminal) func(args []string) error {
	const classTokenLength = 3

	return func(args []string) error {
		token := args[0]
		if len(token) < classTokenLength {
			term.Errorf("The given class token should be at least 3 characters")
			return nil
		}

		validPrefixes := []string{"ama", "ass", "nec", "bar", "sor", "dru", "pal"}
		classToken := strings.ToLower(token)
		tokenPrefix := classToken[0:3]
		isValidToken := false

		for idx := range validPrefixes {
			if strings.Compare(tokenPrefix, validPrefixes[idx]) == 0 {
				isValidToken = true
			}
		}

		if !isValidToken {
			fmtInvalid := "Invalid class, must be a value starting with(case insensitive): %s"
			term.Errorf(fmtInvalid, strings.Join(validPrefixes, ", "))

			return nil
		}

		var err error

		learnedSkillsCount := 0

		for _, skillDetailRecord := range g.asset.Records.Skill.Details {
			if skillDetailRecord.Charclass != classToken {
				continue
			}

			if skill, ok := g.hero.Skills[skillDetailRecord.ID]; ok {
				skill.SkillPoints++
				learnedSkillsCount++
			} else {
				skill, skillErr := g.heroState.CreateHeroSkill(1, skillDetailRecord.Skill)
				if skill == nil {
					continue
				}

				learnedSkillsCount++

				g.hero.Skills[skill.ID] = skill

				if skillErr != nil {
					err = skillErr
					break
				}
			}
		}

		g.hud.skillSelectMenu.RegenerateImageCache()
		g.Infof("Learned %d skills", learnedSkillsCount)

		if err != nil {
			term.Errorf("cannot learn skill for class, error: %s", err)
			return nil
		}

		return nil
	}
}

// OpenTrade opens the vendor window and the inventory beside it.
func (g *GameControls) OpenTrade(v d2vendor.Vendor, seed uint32) {
	if g.PTrade.IsOpen() { // an NPC buy is refused while a player trade is open (inventory-trade.md, w24)
		return
	}

	g.NPCMenu.Close()
	g.clearScreen()
	g.inventory.Open()
	g.Trade.Open(v, seed, nil) // quest overrides: the quest record is not available here, a gap of this port
	g.updateLayout()
}

// OpenGamble opens the gamble window of a vendor and the inventory beside it.
func (g *GameControls) OpenGamble(v d2vendor.Vendor, seed uint32) error {
	if g.PTrade.IsOpen() {
		return fmt.Errorf("a player trade is open")
	}

	g.NPCMenu.Close()
	g.clearScreen()
	g.inventory.Open()

	if err := g.Trade.OpenGamble(v, seed, nil); err != nil {
		g.updateLayout()
		return err
	}

	g.updateLayout()

	return nil
}

// OpenIdentify opens Cain's identify window and the inventory beside it.
func (g *GameControls) OpenIdentify(quest *d2s.QuestRecord) {
	g.NPCMenu.Close()
	g.clearScreen()
	g.inventory.Open()
	g.Identify.Open(quest) // quest bits (4,0) / (4,1) make the fee 0 (UI_BuildNpcMenu 0x4b1050)
	g.updateLayout()
}

func (g *GameControls) onCloseTrade() {
	if g.inventory.IsOpen() {
		g.inventory.Close()
	}

	g.updateLayout()
}

// saveHero persists the hero after a transaction (the server copies the gold
// into the HeroState and writes it, see d2server SavePlayer).
func (g *GameControls) saveHero() {
	g.SyncContainers()

	if err := g.inputListener.OnPlayerSave(); err != nil {
		g.Errorf("saving the hero: %v", err)
	}
}

// SetRelationSource tells the automap and the party panel how the hero sees
// the other players (the roster of the game client).
func (g *GameControls) SetRelationSource(rel func(p *d2mapentity.Player) d2enum.PlayersRelationships) {
	g.relation = rel
}

// isPartyMember reports whether another player is in the hero's party.
func (g *GameControls) isPartyMember(p *d2mapentity.Player) bool {
	return g.relation != nil && g.relation(p) == d2enum.PlayerRelationFriend
}
