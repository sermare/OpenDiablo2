package d2gamescreen

import (
	"errors"
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2gui"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2audio"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2maprenderer"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2screen"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2skills"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2vendor"
	"github.com/OpenDiablo2/OpenDiablo2/d2game/d2player"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2client"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2netpacket"
)

const hideZoneTextAfterSeconds = 2.0

const (
	moveErrStr         = "failed to send MovePlayer packet to the server, playerId: %s, x: %g, x: %g\n"
	bindControlsErrStr = "failed to add gameControls as input handler for player: %s\n"
	castErrStr         = "failed to send CastSkill packet to the server, playerId: %s, skillId: %d, x: %g, x: %g\n"
	spawnItemErrStr    = "failed to send SpawnItem packet to the server: (%d, %d) %+v"
)

const (
	black50alpha = 0x0000007f // rgba
)

// CreateGame creates the Gameplay screen and returns a pointer to it
func CreateGame(
	navigator d2interface.Navigator,
	asset *d2asset.AssetManager,
	ui *d2ui.UIManager,
	renderer d2interface.Renderer,
	inputManager d2interface.InputManager,
	audioProvider d2interface.AudioProvider,
	gameClient *d2client.GameClient,
	term d2interface.Terminal,
	l d2util.LogLevel,
	guiManager *d2gui.GuiManager,
) (*Game, error) {
	// find the local player and its initial location
	var startX, startY float64

	for _, player := range gameClient.Players {
		if player.ID() != gameClient.PlayerID {
			continue
		}

		worldPosition := player.Position.World()
		startX, startY = worldPosition.X(), worldPosition.Y()

		break
	}

	keyMap := d2player.GetDefaultKeyMap(asset)
	keyMap.LoadSavedBindings()

	game := &Game{
		asset:                asset,
		gameClient:           gameClient,
		gameControls:         nil,
		localPlayer:          nil,
		ticksSinceLevelCheck: 0,
		mapRenderer: d2maprenderer.CreateMapRenderer(asset, renderer,
			gameClient.MapEngine, term, l, startX, startY),
		escapeMenu:    d2player.NewEscapeMenu(navigator, renderer, audioProvider, ui, guiManager, asset, l, keyMap),
		inputManager:  inputManager,
		audioProvider: audioProvider,
		renderer:      renderer,
		terminal:      term,
		soundEngine:   d2audio.NewSoundEngine(audioProvider, asset, l, term),
		uiManager:     ui,
		guiManager:    guiManager,
		keyMap:        keyMap,
		logLevel:      l,
	}
	game.Logger = d2util.NewLogger()
	game.Logger.SetLevel(l)
	game.Logger.SetPrefix(logPrefix)
	game.initAutoScript()
	game.hookNetwork()
	setActiveGame(game)

	game.soundEnv = d2audio.NewSoundEnvironment(game.soundEngine)

	game.escapeMenu.OnLoad()

	if err := inputManager.BindHandler(game.escapeMenu); err != nil {
		return nil, errors.New("failed to add gameplay screen as event handler")
	}

	return game, nil
}

// Game represents the Gameplay screen
const (
	npcInteractDistance  = 3.0  // tiles
	npcMenuLeaveDistance = 5.0  // tiles; the menu closes when the hero is farther
	npcBubbleLift        = 30   // pixels above the NPC's head
	npcMenuLift          = 0x96 // the NPC dialog's anchor is this far above the NPC's position (0x4ae400)
	npcMenuTopMin        = 0x14
	noonHour             = 12
	autoTestDelaySeconds = 6.0
	eveningHour          = 18
)

type Game struct {
	// pvpDefend carries the fraction of a life point elemental resist cuts off small PvP ticks
	pvpDefend d2combat.PvPDefendCarry

	*d2mapentity.MapEntityFactory
	asset                *d2asset.AssetManager
	gameClient           *d2client.GameClient
	mapRenderer          *d2maprenderer.MapRenderer
	uiManager            *d2ui.UIManager
	gameControls         *d2player.GameControls
	localPlayer          *d2mapentity.Player
	lastZoneLevel        int       // Levels.txt id last announced; 0 = none yet
	vendorSeed           uint32    // per game session base of every vendor stock seed (set on first use)
	lightLogLevel        int       // level whose base light was last logged
	statsLevel           int       // level of the last DRAWSTATS line
	statsAt              time.Time // time of the last DRAWSTATS line
	travel               travelState
	ticksSinceLevelCheck float64
	escapeMenu           *d2player.EscapeMenu
	soundEngine          *d2audio.SoundEngine
	soundEnv             d2audio.SoundEnvironment
	guiManager           *d2gui.GuiManager
	keyMap               *d2player.KeyMap
	npcTarget            d2interface.MapEntity
	tradeActive          bool        // a vendor window opened from the NPC menu is open
	rewardDrop           *rewardDrop // an item reward the hero walks to claim with the cursor item
	greetingLast         map[string]string
	dayClock             *dayClock
	greetingRecent       map[string]string
	returnGreet          returnGreetings
	autosaveElapsed      float64
	autoTestElapsed      float64
	autoTestDone         bool
	autoScript           *autoScriptState
	levels               levelState
	portal               portalState
	autoSoundElapsed     float64
	autoSoundDone        bool
	ground               groundState
	levelStore           levelStore // state of the levels the hero has left (level_persist.go)
	populated            int        // levels.changes+1 of the level that was populated with monsters
	prisonDoors          int        // levels.changes+1 of the level whose cages got their prison doors
	objects              objectState
	autoObject           autoObject
	autoGround           autoGround
	monsters             *d2monsters.Director
	rankLeader           *d2mapentity.Monster // leader of the last spawnrank pack
	realm                *realmState          // the monsters of a game played through the realm (realm_sync.go)
	monsterTest          *monsterTest
	summonCheckAcc       float64 // seconds since the last SUMMONCHECK line (game_monsters.go)
	aiTest               *aiAutoTest
	bossTest             *bossAutoTest
	uber                 *uberRuntime
	chaos                *chaosRuntime
	act3                 act3State
	uberTest             *uberAutoTest
	merc                 mercGame
	skills               *d2skills.Engine
	skillStatSig         string // last sum of the hero's skill stats, to know when to recalculate
	castTestState        *castTest
	attackTarget         *d2mapentity.Monster
	attackRepathAcc      float64
	soundTraceSet        bool
	heroStepAcc          float64
	speech               *d2audio.Sound // the NPC voice line playing, if any
	ambientTest          *ambientTest
	regionEnvs           map[int]int
	autoPanel            autoPanelState
	autoCube             autoCubeState
	autoEquip            autoEquipState
	levelStatusAcc       float64
	questRT              *questRuntime
	death                deathState
	social               socialState

	renderer      d2interface.Renderer
	inputManager  d2interface.InputManager
	audioProvider d2interface.AudioProvider
	terminal      d2interface.Terminal

	*d2util.Logger
	logLevel d2util.LogLevel
}

// OnLoad loads the resources for the Gameplay screen
func (v *Game) OnLoad(_ d2screen.LoadingState) {
	v.audioProvider.PlayBGM("")

	commands := []struct {
		name string
		desc string
		args []string
		fn   func([]string) error
	}{
		{"spawnitem", "spawns an item at the local player position",
			[]string{"code1", "code2", "code3", "code4", "code5"}, v.commandSpawnItem},
		{"spawnitemat", "spawns an item at the x,y coordinates",
			[]string{"x", "y", "code1", "code2", "code3", "code4", "code5"}, v.commandSpawnItemAt},
		{"spawnmon", "spawn monster at the local player position", []string{"name"}, v.commandSpawnMon},
		{"forcestate", "puts a forced AI state (fear, blind, taunt, confuse, attract, charm) on a monster for n frames",
			[]string{"monster id", "state", "frames"}, v.commandForceState},
		{"setgold", "sets the hero's gold (saved to the .d2s on the next save)", []string{"amount"}, v.commandSetGold},
		{"spawnchest", "spawns chests/barrels (objects.txt ids, default 7 1 5) next to the hero",
			[]string{"id1", "id2", "id3"}, v.commandSpawnChest},
		{"spawnportal", "spawns a town portal object to the given level next to the hero",
			[]string{"level"}, v.commandSpawnPortal},
		{"setwaypoint", "activates (1) or clears (0) the waypoint of a level for the hero",
			[]string{"level", "0|1"}, v.commandSetWaypoint},
		{"completequest", "marks quest <act> <quest> done for the hero (debug)",
			[]string{"act", "quest"}, v.commandCompleteQuest},
		{"restorevitals", "fills the hero's life and mana (debug)", nil, v.commandRestoreVitals},

		{"useitem", "uses the inventory item with this base code like a right click (debug)", []string{"code"}, v.commandUseItem},
		{"lootquest", "picks up the quest items near the hero (debug)", []string{"tiles", "seconds"}, v.commandLootQuest},
		{"clearinv", "empties the inventory grid (debug)", nil, v.commandClearInv},
		{"questpanel", "opens the quest log on a quest and logs its title and page text", []string{"act", "quest"}, v.commandQuestPanel},
		{"setmana", "sets the hero's mana, at most the maximum (debug)", []string{"n"}, v.commandSetMana},
		{"resetquests", "clears the hero's quest record in memory (debug)", nil, v.commandResetQuests},
		{"travelfree", "1 lets act travel skip the quest and NPC rules (debug), 0 restores them",
			[]string{"0|1"}, v.commandTravelFree},
		{"travel", "travels to the town of an act through the act travel rules",
			[]string{"act"}, v.commandTravel},
		{"walkprobe", "logs how many lava/water tiles of the level the hero can walk to and whether a walk order onto lava ends on it (debug)",
			nil, v.commandWalkProbe},
		{"players", "logs the players of the game with their positions", []string{}, v.commandPlayers},
		{"chat", "sends a chat line to all players (_ for a space)", []string{"text"}, v.commandChat},
		{"mpkill", "realm games: fight the nearest monster of the realm", nil, v.commandMPKill},
		{"mpworld", "realm games: logs the simulation as this client sees it (digest, monsters)", nil, v.commandMPWorld},
		{"party", "party invite|accept|decline|leave|list <name or ->", []string{"op", "name"}, v.commandParty},
		{"hostile", "declares (1) or withdraws (0) hostility toward a player", []string{"name", "0|1"}, v.commandHostile},
		{"roster", "logs the roster and the party panel", []string{}, v.commandRoster},
		{"trade", "trade request|yes|no|add|remove|gold|accept|cancel <name, item code, amount or ->",
			[]string{"op", "arg"}, v.commandTrade},
		{"pvp", "swings at another player (melee, needs hostility)", []string{"name"}, v.commandPvP},
		{"giveitem", "puts a new item into the inventory", []string{"code"}, v.commandGiveItem},
		{"dropinv", "removes the first inventory item with this base code (debug)", []string{"code"}, v.commandDropInv},
		{"autobuy", "opens a vendor's trade window and buys the cheapest affordable item (OD2_AUTOTRADE_KEEP=1 keeps it)",
			[]string{"vendor"}, v.commandAutoBuy},
		{"spawnrank", "spawns a champion pack, a unique pack or a super unique next to the hero (drop tests)",
			[]string{"champion|unique|super", "monster or super unique"}, v.commandSpawnRank},
		{"killleader", "kills the leader of the last spawnrank pack as the hero", []string{}, v.commandKillLeader},
		{"pvpcast", "casts a skill (name, _ for a space) at another player's position",
			[]string{"skill", "name"}, v.commandPvPCast},
		{"pvpwalk", "walks the hero by dx dy tiles", []string{"dx", "dy"}, v.commandPvPWalk},
		{"sethp", "sets the hero's life points (scenarios)", []string{"hp"}, v.commandSetHP},
		{"townportal", "casts a town portal (scroll or tome charge; \"free\" skips the charge)", []string{"free"}, v.commandTownPortal},
		{"closeportal", "closes the hero's town portal pair", []string{}, v.commandClosePortal},
		{"portals", "logs the open town portal pairs", []string{}, v.commandPortals},
		{"useportal", "uses the nearest town portal object without walking to it (scenarios)", []string{}, v.commandUsePortal},
		{"killnear", "kills the nearest monster as the hero (party experience tests)", []string{}, v.commandKillNear},
		{"rewarditem", "spends a pending Larzuk (socket), Anya (personalize) or Charsi (imbue) quest reward on an item",
			[]string{"socket|personalize|imbue"}, v.commandRewardItem},
		{"questpending", "puts quest <act> <quest> into the state of a finished quest whose reward waits (debug)",
			[]string{"act", "quest"}, v.commandQuestPending},
		{"pickitem", "takes the first inventory item with this base code (or any) onto the cursor",
			[]string{"code|any"}, v.commandPickItem},
		{"giveitemq", "puts a new item of a quality (1 low .. 4 magic, 6 rare) into the inventory",
			[]string{"code", "quality"}, v.commandGiveItemQ},
		{"freeinv", "removes up to n items from the inventory to make room (debug)", []string{"n"}, v.commandFreeInv},
		{"putitem", "puts the cursor item back into the inventory", nil, v.commandPutItem},
		{"transmute", "transmutes the quest recipes in the Horadric Cube (Staff, Khalim's Will, Pandemonium portals)",
			nil, v.commandTransmute},
		{"pickground", "walks to the nearest ground item with this base code and picks it up (scripts)",
			[]string{"code"}, v.commandPickGround},
		{"cubeput", "moves inventory items (by base code) into the Horadric Cube; then use transmute (debug)",
			[]string{"code1", "code2", "code3", "code4"}, v.commandCubePut},
		{"setexp", "raises the hero's experience to at least <amount>; the level follows (debug)", []string{"amount"}, v.commandSetExp},
	}

	for _, cmd := range commands {
		if err := v.terminal.Bind(cmd.name, cmd.desc, cmd.args, cmd.fn); err != nil {
			v.Errorf(err.Error())
		}
	}

	if err := v.asset.BindTerminalCommands(v.terminal); err != nil {
		v.Errorf(err.Error())
	}
}

// OnUnload releases the resources of Gameplay screen
func (v *Game) OnUnload() error {
	v.gameControls.UnbindGamepad()

	if err := v.gameControls.UnbindTerminalCommands(v.terminal); err != nil {
		return err
	}

	// https://github.com/OpenDiablo2/OpenDiablo2/issues/792
	if err := v.inputManager.UnbindHandler(v.gameControls); err != nil {
		return err
	}

	// https://github.com/OpenDiablo2/OpenDiablo2/issues/792
	if err := v.inputManager.UnbindHandler(v.escapeMenu); err != nil {
		return err
	}

	if err := v.terminal.Unbind("spawnitemat", "spawnitem", "spawnmon", "spawnchest", "setgold", "spawnportal", "setwaypoint", "players", "chat",
		"party", "hostile", "roster", "trade", "pvp", "giveitem", "dropinv", "autobuy", "spawnrank", "killleader", "killnear", "rewarditem", "transmute", "setexp", "cubeput", "pickground",
		"questpending", "pickitem", "putitem", "giveitemq", "freeinv",
		"townportal", "closeportal", "portals", "useportal", "pvpcast", "pvpwalk", "sethp"); err != nil {
		return err
	}

	if err := v.OnPlayerSave(); err != nil {
		return err
	}

	clearActiveGame(v)

	if err := v.gameClient.Close(); err != nil {
		return err
	}

	if err := v.asset.UnbindTerminalCommands(v.terminal); err != nil {
		return err
	}

	if err := v.mapRenderer.UnbindTerminalCommands(v.terminal); err != nil {
		return err
	}

	if err := v.soundEngine.UnbindTerminalCommands(v.terminal); err != nil {
		return err
	}

	v.soundEngine.Reset()

	return nil
}

// Render renders the Gameplay screen
func (v *Game) Render(screen d2interface.Surface) {
	if v.gameClient.RegenMap {
		v.gameClient.RegenMap = false
		v.mapRenderer.RegenerateTileCache()
		v.gameClient.MapEngine.IsLoading = false
	}

	screen.Clear(color.Black)
	v.mapRenderer.Render(screen)
	v.logDrawStats()

	if v.gameControls != nil {
		if v.gameControls.HelpOverlay != nil && v.gameControls.HelpOverlay.IsOpen() {
			screen.DrawRect(screenWidth, screenHeight, d2util.Color(black50alpha))
		}

		if err := v.gameControls.Render(screen); err != nil {
			return
		}
	}

	v.renderFade(screen)
}

// Advance runs the update logic on the Gameplay screen
// nolint:gocyclo // not need to change
func (v *Game) Advance(elapsed float64) error {
	// OD2_AUTOSPEED: a fast scripted run takes several update steps per frame. Every step is at most one
	// 25 Hz tick long (the simulations clamp a call to 0.25 s and the hero's walking and collision were
	// made for short steps), so no simulation time is lost however fast the clock runs. The harness
	// (script, autotests, screenshots) runs once per rendered frame with the whole span.
	total := elapsed * autoTimeScale()
	steps := autoSubsteps(total)
	per := total / float64(steps)

	for i := 0; i < steps; i++ {
		if err := v.advanceStep(per, total, i == steps-1); err != nil {
			return err
		}
	}

	return nil
}

// advanceStep is one update step of the Gameplay screen; the harness (autoscript, autotests) only runs
// when last is set, with the span total of the whole frame.
// nolint:gocyclo // not need to change
func (v *Game) advanceStep(elapsed, total float64, last bool) error {
	d2util.PerfMark("game-playable")

	v.gameClient.Drain()

	v.soundEngine.Advance(elapsed)
	v.advanceDayClock(elapsed)
	v.advanceLighting()

	v.advanceNPCInteraction(elapsed)
	if last {
		v.advanceAutoSound(total)
	}
	if last {
		v.advanceAutoTest(total)
	}
	if last {
		v.advanceFlow(total)
	}
	if last {
		v.advanceAutoScript(total)
	}
	v.advanceQuests(elapsed)
	v.advanceAutosave(elapsed)
	v.advanceGroundInteraction(elapsed)
	v.advanceObjects(elapsed)
	if last {
		v.advanceAutoObject(total)
	}
	v.advanceLevels(elapsed)
	v.advanceSavedAct()
	if last {
		v.advanceAutoGround(total)
	}
	v.advanceSound(elapsed)
	v.advanceAutoAmbient(elapsed)
	v.advanceAutoPanel(elapsed)
	v.advanceAutoCube(elapsed)
	v.advanceAutoEquip(elapsed)
	v.advanceSocial(elapsed)

	if (v.escapeMenu != nil && !v.escapeMenu.IsOpen()) || len(v.gameClient.Players) != 1 {
		v.gameClient.MapEngine.Advance(elapsed)
		v.advanceMonsters(elapsed)
		v.advanceSkills(elapsed)
		v.advanceHeroLevel()
		v.advanceDeath(elapsed)
	}

	if v.gameControls != nil {
		if err := v.gameControls.Advance(elapsed); err != nil {
			return err
		}
	}

	v.ticksSinceLevelCheck += elapsed
	if v.ticksSinceLevelCheck > 1 {
		v.ticksSinceLevelCheck = 0
		if v.localPlayer != nil {
			tilePosition := v.localPlayer.Position.Tile()
			tile := v.gameClient.MapEngine.TileAt(int(tilePosition.X()), int(tilePosition.Y()))

			if tile != nil {
				// tile.RegionType is the LevelType, not a Levels.txt id: index
				// Details by the id of the level the hero is actually in.
				levelID := v.currentLevel()
				levelDetails := v.asset.Records.Level.Details[levelID]

				fallbackEnv := 0
				if levelDetails != nil {
					fallbackEnv = levelDetails.SoundEnvironmentID
				}

				if v.ambientTest == nil { // OD2_AUTOAMBIENT picks the environment itself
					v.soundEnv.SetEnv(v.soundEnvForRegion(tile.RegionType, fallbackEnv))
				}

				// skipped the first time we enter the world
				if text, ok := zoneChangeText(v.lastZoneLevel, levelID, levelDetails); ok {
					v.gameControls.SetZoneChangeText(text)
					v.gameControls.ShowZoneChangeText()
					v.gameControls.HideZoneChangeTextAfter(hideZoneTextAfterSeconds)
				}

				v.lastZoneLevel = levelID
			}
		}
	}

	// Bind the game controls to the player once it exists
	if v.gameControls == nil {
		if err := v.bindGameControls(); err != nil {
			return err
		}
	}

	// Update the camera to focus on the player
	if v.localPlayer != nil && !v.gameControls.FreeCam {
		worldPosition := v.localPlayer.Position.World()
		rx, ry := v.mapRenderer.WorldToOrtho(worldPosition.X(), worldPosition.Y())
		position := d2vector.NewPosition(rx, ry)
		v.mapRenderer.SetCameraTarget(&position)
	}

	v.soundEnv.Advance(elapsed * v.ambientSpeed())

	if v.gameControls != nil {
		if v.gameControls.PartyPanel != nil {
			v.gameControls.PartyPanel.UpdatePlayersList(v.gameClient.Players)
		}
	}

	return nil
}

func (v *Game) bindGameControls() error {
	d2player.SetItemSoundHook(v.onItemSound)

	for _, player := range v.gameClient.Players {
		if player.ID() != v.gameClient.PlayerID {
			continue
		}

		v.localPlayer = player

		var err error
		v.gameControls, err = d2player.NewGameControls(v.asset, v.renderer, player, v.gameClient.MapEngine,
			v.escapeMenu, v.mapRenderer, v, v.terminal, v.uiManager, v.keyMap, v.audioProvider, v.logLevel,
			v.gameClient.IsSinglePlayer(), v.gameClient.Players)

		if err != nil {
			return err
		}

		v.gameControls.Load()
		v.gameControls.SetCubePortalHandler(v.cubePortal)
		v.gameControls.BindGamepad()
		v.gameControls.Automap().SetLevelSource(v.currentLevel, v.levelName)
		v.gameControls.SetEquipSound(v.playHeroUISound)
		v.gameControls.SetQuestItemUse(v.useQuestItem)

		if err := v.inputManager.BindHandler(v.gameControls); err != nil {
			v.Error(bindControlsErrStr + player.ID())
		}

		break
	}

	return nil
}

// OnPlayerMove is a move order (a click or a script step). It cancels a walk
// to an object and targets a warp tile if the order lands on one.
func (v *Game) OnPlayerMove(targetX, targetY float64) {
	if v.localPlayer.IsDead() {
		return // the dead do not walk
	}

	v.levels.use = nil
	v.targetWarpAt(targetX, targetY)
	v.movePlayerTo(targetX, targetY)
}

// movePlayerTo sends the player move action to the server
func (v *Game) movePlayerTo(targetX, targetY float64) {
	worldPosition := v.localPlayer.Position.World()

	playerID, worldX, worldY := v.gameClient.PlayerID, worldPosition.X(), worldPosition.Y()

	createMovePlayerPacket, err := d2netpacket.CreateMovePlayerPacket(playerID, worldX, worldY, targetX, targetY)
	if err != nil {
		v.Errorf("MovePlayerPacket: %v", err)
	}

	err = v.gameClient.SendPacketToServer(createMovePlayerPacket)

	if err != nil {
		v.Errorf(moveErrStr, v.gameClient.PlayerID, targetX, targetY)
	}
}

// OnPlayerInteract walks the player up to the given entity (e.g. an NPC)
func (v *Game) OnPlayerInteract(entity d2interface.MapEntity) {
	switch e := entity.(type) {
	case *d2mapentity.Item:
		v.walkToItem(e)
		return
	case *d2mapentity.Object:
		v.walkToObject(e)

		return
	}

	if v.tryRewardDrop(entity) { // an NPC that is owed an item reward and the hero holds an item
		return
	}

	targetX, targetY := entity.GetPositionF()

	v.Infof("interacting with %q", entity.Label())

	if v.gameControls != nil {
		v.gameControls.NPCMenu.Close()
	}

	v.npcTarget = entity

	v.OnPlayerMove(targetX, targetY)
	v.levels.warpTarget = nil // an NPC that wanders near an exit is not a click on the exit
}

// npcClassID returns the monstats class id (hcIdx) of an NPC entity, or -1.
// stockSeed is the seed of a vendor's stock in this game session: fixed per
// session (so the stock of a vendor does not depend on when the window is
// opened) and different for every vendor and for gamble versus normal stock.
func (v *Game) stockSeed(npc d2interface.MapEntity, gamble bool) uint32 {
	if v.vendorSeed == 0 {
		v.vendorSeed = uint32(time.Now().UnixNano()) | 1
	}

	return d2vendor.StockSeed(v.vendorSeed, v.npcClassID(npc), 0, gamble)
}

func (v *Game) npcClassID(entity d2interface.MapEntity) int {
	if npc, ok := entity.(interface{ MonstatID() int }); ok {
		return npc.MonstatID()
	}

	return -1
}

// advanceNPCInteraction opens the NPC menu once the player has walked up to
// the NPC they clicked, keeps it attached to the NPC and closes it again
// when the player walks away.
func (v *Game) advanceNPCInteraction(_ float64) {
	if v.npcTarget == nil || v.localPlayer == nil || v.gameControls == nil {
		return
	}

	menu := v.gameControls.NPCMenu

	px, py := v.localPlayer.GetPositionF()
	nx, ny := v.npcTarget.GetPositionF()
	dist := math.Hypot(px-nx, py-ny)

	if v.tradeActive {
		// the trade window replaces the menu; it closes when the hero walks
		// away (not in the autotest, where the hero stays where it is)
		switch {
		case !v.gameControls.Trade.IsOpen() && !v.gameControls.Identify.IsOpen():
			v.tradeActive = false
			v.npcTarget = nil
		case dist > npcMenuLeaveDistance && os.Getenv("OD2_AUTOTRADE") == "" && os.Getenv("OD2_AUTOGAMBLE") == "" &&
			os.Getenv("OD2_AUTOIDENTIFY") == "":
			v.Infof("trade window closed: walked away from %q", v.npcTarget.Label())
			v.gameControls.Trade.Close()
			v.gameControls.Identify.Close()
		}

		return
	}

	if menu.IsOpen() {
		if dist > npcMenuLeaveDistance {
			v.Infof("NPC menu closed: walked away from %q", v.npcTarget.Label())
			menu.Close()
			v.questClose(v.npcTarget)

			v.npcTarget = nil

			return
		}

		v.anchorNPCMenu(menu, v.npcTarget)

		return
	}

	if dist > npcInteractDistance {
		return
	}

	if v.rewardDrop != nil && v.rewardDrop.npc == v.npcTarget {
		v.finishRewardDrop()

		return
	}

	v.openNPCMenu(menu, v.npcTarget)
	v.playNPCGreeting(v.npcTarget.Label())
}

// endConversationUnlessMenuOpen forgets the NPC the hero walked up to once its
// menu is gone: advanceNPCInteraction would otherwise open the menu again at
// once (and play another greeting); like in the original the player clicks the
// NPC again to talk again.
func (v *Game) endConversationUnlessMenuOpen() {
	if !v.gameControls.NPCMenu.IsOpen() {
		v.npcTarget = nil
	}
}

func (v *Game) anchorNPCMenu(menu *d2player.NPCMenu, npc d2interface.MapEntity) {
	sx, sy := v.mapRenderer.WorldToScreenF(npc.GetPositionF())
	_, h := npc.GetSize()

	// the original (0x4ae400) anchors the dialog at the NPC's screen position lifted by 0x96, at least 0x14 from the top
	_ = h

	ay := int(sy) - npcMenuLift
	if ay < npcMenuTopMin {
		ay = npcMenuTopMin
	}

	menu.SetAnchor(int(sx), ay)
}

// openNPCMenu shows the Talk/Trade/... menu for an NPC.
func (v *Game) openNPCMenu(menu *d2player.NPCMenu, npc d2interface.MapEntity) []d2player.NPCMenuRow {
	classID := v.npcClassID(npc)
	rows, known := d2player.NPCMenuFor(classID)

	rows = v.withTravelRows(classID, rows)
	rows = v.withRewardRows(classID, rows)

	menu.Open(npc.Label(), rows, 0, 0, func(row d2player.NPCMenuRow) {
		v.onNPCMenuChoice(npc, row)
	})

	v.anchorNPCMenu(menu, npc)

	labels := make([]string, 0, len(menu.Rows()))
	for _, r := range menu.Rows() {
		labels = append(labels, menu.RowLabel(r))
	}

	v.Infof("NPC menu opened: npc=%q class=%d known=%v rows=%v", npc.Label(), classID, known, labels)

	return menu.Rows()
}

func (v *Game) onNPCMenuChoice(npc d2interface.MapEntity, row d2player.NPCMenuRow) {
	switch row.Action {
	case d2player.NPCActionCancel:
		v.Infof("NPC menu: Cancel")
		v.questClose(npc)

		v.npcTarget = nil
	case d2player.NPCActionTalk:
		// the Talk row ends the menu; questTalk may open the topic submenu in its place
		v.gameControls.NPCMenu.Close()

		if v.questTalk(npc) {
			v.Infof("NPC menu: Talk with %q (quest speech)", npc.Label())
			v.endConversationUnlessMenuOpen()

			return
		}

		path := v.playNPCGreeting(npc.Label())
		v.Infof("NPC menu: Talk with %q (voice %q)", npc.Label(), path)
		v.travelOnTalk(npc)
		v.endConversationUnlessMenuOpen()
	case d2player.NPCActionTopic:
		v.questTopic(npc, row.StringID)
		v.endConversationUnlessMenuOpen()
	case d2player.NPCActionTrade, d2player.NPCActionTradeRepair:
		v.openTrade(npc, v.stockSeed(npc, false))
	case d2player.NPCActionHire:
		v.openHire(npc)
	case d2player.NPCActionGamble:
		v.openGamble(npc, v.stockSeed(npc, true))
	case d2player.NPCActionIdentify:
		v.openIdentify(npc)
	case d2player.NPCActionTravelWest, d2player.NPCActionSailWest, d2player.NPCActionTravelEast, d2player.NPCActionSailEast:
		v.travelFromNPC(npc, row)
	case d2player.NPCActionReward, d2player.NPCActionRespec:
		v.onRewardRow(npc, row)
	default:
		v.Infof("NPC menu: %s (%s) not implemented yet", row.Action, row.Fallback)
	}
}

// openGamble opens the gamble window of a vendor that has one (Gheed).
func (v *Game) openGamble(npc d2interface.MapEntity, seed uint32) bool {
	vendor, ok := d2vendor.ByClassID(v.npcClassID(npc))
	if !ok || !vendor.Gambles {
		v.Infof("NPC menu: Gamble with %q not implemented yet (no gamble model for class %d)",
			npc.Label(), v.npcClassID(npc))

		return false
	}

	if err := v.gameControls.OpenGamble(vendor, seed); err != nil {
		v.Infof("NPC menu: Gamble with %q: %v", npc.Label(), err)

		return false
	}

	v.npcTarget, v.tradeActive = npc, true

	return true
}

// openIdentify opens Deckard Cain's identify window.
func (v *Game) openIdentify(npc d2interface.MapEntity) {
	v.gameControls.OpenIdentify()

	v.npcTarget, v.tradeActive = npc, true
}

// openTrade opens the vendor window for an Act 1 vendor. Other NPCs have a
// Trade row but no stock model yet.
func (v *Game) openTrade(npc d2interface.MapEntity, seed uint32) bool {
	vendor, ok := d2vendor.ByClassID(v.npcClassID(npc))
	if !ok {
		v.Infof("NPC menu: Trade with %q not implemented yet (no vendor model for class %d)",
			npc.Label(), v.npcClassID(npc))

		return false
	}

	v.gameControls.OpenTrade(vendor, seed)

	v.npcTarget, v.tradeActive = npc, true

	return true
}

func (v *Game) currentDayPhase() dayPhaseSource {
	if v.dayClock == nil {
		v.dayClock = newDayClock()
	}

	return v.dayClock
}

func (v *Game) advanceDayClock(elapsed float64) {
	if v.dayClock == nil {
		v.dayClock = newDayClock()
	}

	v.dayClock.Advance(elapsed)
}

// playNPCGreeting plays the NPC's spoken greeting, chosen the way the real
// game's picker does (see pickGreeting in npc_greeting.go).
func (v *Game) playNPCGreeting(name string) string {
	name = strings.ToLower(strings.TrimPrefix(name, "Deckard "))

	if v.greetingLast == nil {
		v.greetingLast = make(map[string]string)
		v.greetingRecent = make(map[string]string)
	}

	if v.returnGreet == nil {
		v.returnGreet = returnGreetings{}
	}

	v.armReturnGreeting(name)

	set := loadGreetingSet(v.asset.Records.Sound.Details, name)

	// nolint:gosec // not concerned with crypto-strong randomness
	handle := pickGreeting(set, v.returnGreet.Take(name), v.currentDayPhase().Phase(),
		v.greetingLast[name], v.greetingRecent, rand.Intn)
	if handle == "" {
		return ""
	}

	v.greetingLast[name] = handle
	record := v.asset.Records.Sound.Details[handle]
	path := "data/local/sfx/" + strings.ReplaceAll(record.FileName, "\\", "/")

	ok, _ := v.asset.FileExists(path)
	v.Debugf("greeting %s file %s exists=%v", handle, path, ok)

	if !ok {
		return ""
	}

	sfx, err := v.audioProvider.LoadSound(path, false, false)
	if err != nil {
		v.Warningf("could not load NPC greeting %s: %v", path, err)
		return ""
	}

	if os.Getenv("OD2_AUTOTEST_MUTE") == "" {
		sfx.Play()
	}

	v.Infof("NPC greeting: %s (%s)", handle, path)

	return path
}

// advanceAutoSound plays the sounds named in OD2_AUTOSOUND=<handle|index>[,..]
// through the sound engine's voice bank and logs the resolved row, file,
// priority and decision (AUTOSOUND lines). OD2_AUTOTEST_MUTE keeps it silent;
// with OD2_AUTOEXIT it quits afterwards unless another autotest is running.
func (v *Game) advanceAutoSound(elapsed float64) {
	spec := os.Getenv("OD2_AUTOSOUND")
	if spec == "" || v.autoSoundDone {
		return
	}

	v.autoSoundElapsed += elapsed
	if v.autoSoundElapsed < autoTestDelaySeconds {
		return
	}

	v.autoSoundDone = true
	v.soundEngine.AutoSound(spec)

	if os.Getenv("OD2_AUTOTALK") == "" && os.Getenv("OD2_AUTOMENU") == "" {
		v.autoTestExit()
	}
}

// advanceAutoTest checks NPC behaviour without any clicking.
//   - OD2_AUTOTALK=<names>: logs and plays the greeting chosen for each NPC.
//   - OD2_AUTOMENU=<names>: opens each NPC's menu and logs its rows;
//     OD2_AUTOMENU_CHOOSE=<Talk|Trade|...|Cancel> then picks that row, and
//     OD2_AUTOMENU_HOLD=<seconds> keeps the last menu on screen that long.
//
// OD2_AUTOTEST_MUTE skips playback and OD2_AUTOEXIT quits when done.
func (v *Game) advanceAutoTest(elapsed float64) {
	talk, menus, trades := os.Getenv("OD2_AUTOTALK"), os.Getenv("OD2_AUTOMENU"), os.Getenv("OD2_AUTOTRADE")
	gamble, identify := os.Getenv("OD2_AUTOGAMBLE"), os.Getenv("OD2_AUTOIDENTIFY")
	if os.Getenv("OD2_AUTOOPTIONS") != "" && v.localPlayer != nil && v.gameControls != nil && !v.autoTestDone {
		v.autoTestElapsed += elapsed
		if v.autoTestElapsed >= autoTestDelaySeconds {
			v.autoTestDone = true
			v.gameControls.RunOptionsAutoTest()
			v.autoTestExit()
		}

		return
	}

	if (talk == "" && menus == "" && trades == "" && gamble == "" && identify == "") || v.localPlayer == nil || v.gameControls == nil {
		return
	}

	v.autoTestElapsed += elapsed

	if v.autoTestDone {
		v.autoTestHold(elapsed)
		return
	}

	if v.autoTestElapsed < autoTestDelaySeconds {
		return
	}

	v.autoTestDone = true

	byLabel := make(map[string]d2interface.MapEntity)

	for _, e := range v.gameClient.MapEngine.Entities() {
		if label := e.Label(); label != "" {
			byLabel[label] = e
		}
	}

	if talk != "" {
		for _, name := range strings.Split(talk, ",") {
			_, present := byLabel[name]
			path := v.playNPCGreeting(name)
			v.Infof("AUTOTEST greeting npc=%s in_town=%v file=%q", name, present, path)
		}
	}

	for _, name := range strings.Split(menus, ",") {
		npc, present := byLabel[name]
		if name == "" || !present {
			if name != "" {
				v.Infof("AUTOTEST menu npc=%s in_town=false", name)
			}

			continue
		}

		rows := v.openNPCMenu(v.gameControls.NPCMenu, npc)
		v.Infof("AUTOTEST menu npc=%s class=%d rows=%d", name, v.npcClassID(npc), len(rows))

		if choose := os.Getenv("OD2_AUTOMENU_CHOOSE"); choose != "" {
			for i, r := range rows {
				if strings.EqualFold(r.Action.String(), choose) {
					v.gameControls.NPCMenu.Choose(i)
				}
			}
		}
	}

	v.autoTestTrade(trades, byLabel)
	v.autoTestGamble(gamble, byLabel)
	v.autoTestIdentify(identify, byLabel)

	if os.Getenv("OD2_AUTOMENU_HOLD") == "" {
		v.autoTestExit()
	}
}

// autoTestTrade runs OD2_AUTOTRADE=<names>: for each vendor it opens the trade
// window, logs the stock with computed buy prices and runs one scripted buy
// and sell (and a repair for Charsi). OD2_AUTOTRADE_SEED fixes the stock.
func (v *Game) autoTestTrade(names string, byLabel map[string]d2interface.MapEntity) {
	if names == "" {
		return
	}

	seed := uint32(1)
	if n, err := strconv.ParseUint(os.Getenv("OD2_AUTOTRADE_SEED"), 10, 32); err == nil {
		seed = uint32(n)
	}

	if n, err := strconv.Atoi(os.Getenv("OD2_AUTOTRADE_LEVEL")); err == nil {
		v.gameControls.Trade.LevelOverride = n
	}

	for _, name := range strings.Split(names, ",") {
		npc, present := byLabel[name]
		if !present {
			v.Infof("AUTOTRADE npc=%s in_town=false", name)
			continue
		}

		if !v.openTrade(npc, seed) {
			continue
		}

		v.gameControls.Trade.RunAutoTest() // stays open for OD2_AUTOMENU_HOLD
	}
}

// autoTestGamble runs OD2_AUTOGAMBLE=<names>: opens each vendor's gamble
// window and logs the stock, prices and a scripted purchase (AUTOGAMBLE lines,
// no clicking). OD2_AUTOTRADE_SEED and OD2_AUTOTRADE_LEVEL apply too.
func (v *Game) autoTestGamble(names string, byLabel map[string]d2interface.MapEntity) {
	if names == "" {
		return
	}

	seed := uint32(1)
	if n, err := strconv.ParseUint(os.Getenv("OD2_AUTOTRADE_SEED"), 10, 32); err == nil {
		seed = uint32(n)
	}

	if n, err := strconv.Atoi(os.Getenv("OD2_AUTOTRADE_LEVEL")); err == nil {
		v.gameControls.Trade.LevelOverride = n
	}

	for _, name := range strings.Split(names, ",") {
		npc, present := byLabel[name]
		if !present {
			v.Infof("AUTOGAMBLE npc=%s in_town=false", name)
			continue
		}

		if !v.openGamble(npc, seed) {
			continue
		}

		v.gameControls.Trade.RunGambleAutoTest()
	}
}

// autoTestIdentify runs OD2_AUTOIDENTIFY=1: it opens Cain's window, logs the
// unidentified items and fees and identifies them (AUTOIDENTIFY lines).
func (v *Game) autoTestIdentify(spec string, byLabel map[string]d2interface.MapEntity) {
	if spec == "" {
		return
	}

	var cain d2interface.MapEntity

	for name, e := range byLabel {
		if strings.Contains(name, "Cain") {
			cain = e
		}
	}

	if cain == nil {
		// Cain only stands in the camp after his rescue; the window does not
		// need him, so the test opens it directly
		v.Infof("AUTOIDENTIFY npc=Cain in_town=false (opening the window directly)")
		v.gameControls.OpenIdentify()
	} else {
		v.openIdentify(cain)
	}

	v.gameControls.Identify.RunAutoTest()
}

func (v *Game) autoTestHold(_ float64) {
	hold, err := strconv.ParseFloat(os.Getenv("OD2_AUTOMENU_HOLD"), 64)
	if err == nil && v.autoTestElapsed >= autoTestDelaySeconds+hold {
		v.autoTestExit()
	}
}

func (v *Game) autoTestExit() {
	if os.Getenv("OD2_AUTOEXIT") != "" {
		v.saveBeforeExit()
		os.Exit(0)
	}
}

// OnPlayerSave instructs the server to save our player data
func (v *Game) OnPlayerSave() error {
	playerState := v.gameClient.Players[v.gameClient.PlayerID]

	if v.gameControls != nil {
		v.gameControls.SyncContainers()
	}

	sp, err := d2netpacket.CreateSavePlayerPacket(playerState, v.gameClient.Difficulty)
	if err != nil {
		return fmt.Errorf("SavePlayerPacket: %v", err)
	}

	err = v.gameClient.SendPacketToServer(sp)

	if err != nil {
		return err
	}

	return nil
}

// OnPlayerCast sends the casting skill action to the server
func (v *Game) OnPlayerCast(skillID int, targetX, targetY float64) {
	if v.localPlayer != nil && v.localPlayer.IsDead() {
		return
	}

	// skills the skill pipeline implements run locally with real missiles; the
	// rest keep the old path (a CastSkill packet that plays the client effects)
	if v.localPlayer != nil && v.castWithPipeline(skillID, targetX, targetY) {
		v.announceCast(skillID, targetX, targetY)
		return
	}

	cp, err := d2netpacket.CreateCastPacket(v.gameClient.PlayerID, skillID, targetX, targetY)
	if err != nil {
		v.Errorf("CastPacket: %v", err)
	}

	err = v.gameClient.SendPacketToServer(cp)
	if err != nil {
		v.Errorf(castErrStr, v.gameClient.PlayerID, skillID, targetX, targetY)
	}
}

func (v *Game) debugSpawnItemAtPlayer(codes ...string) {
	if v.localPlayer == nil {
		return
	}

	pos := v.localPlayer.GetPosition()
	tile := pos.Tile()
	x, y := int(tile.X()), int(tile.Y())

	v.debugSpawnItemAtLocation(x, y, codes...)
}

func (v *Game) debugSpawnItemAtLocation(x, y int, codes ...string) {
	packet, err := d2netpacket.CreateSpawnItemPacket(x, y, codes...)
	if err != nil {
		v.Errorf("SpawnItemPacket: %v", err)
	}

	err = v.gameClient.SendPacketToServer(packet)
	if err != nil {
		v.Errorf(spawnItemErrStr, x, y, codes)
	}
}

func (v *Game) commandSpawnItem(args []string) error {
	v.debugSpawnItemAtPlayer(args...)

	return nil
}

func (v *Game) commandSpawnItemAt(args []string) error {
	x, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid argument")
	}

	y, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid argument")
	}

	v.debugSpawnItemAtLocation(x, y, args[2:]...)

	return nil
}

func (v *Game) commandSpawnMon(args []string) error {
	name := args[0]
	x := int(v.localPlayer.Position.X())
	y := int(v.localPlayer.Position.Y())

	monstat := v.asset.Records.Monster.Stats[name]
	if monstat == nil {
		v.terminal.Errorf("no monstat entry for \"%s\"", name)
		return nil
	}

	// Hostile classes get the real monster AI; everything else stays a
	// passive NPC as before.
	if d := v.monsterDirector(); d != nil && d2monsters.IsHostile(monstat) {
		if _, err := d.SpawnNear(monstat, x+monsterSpawnOffset, y, 2); err != nil {
			v.terminal.Errorf("error generating monster \"%s\": %v", name, err)
		}

		return nil
	}

	monster, npcErr := v.gameClient.MapEngine.NewNPC(x, y, monstat, 0)
	if npcErr != nil {
		v.terminal.Errorf("error generating monster \"%s\": %v", name, npcErr)
		return nil
	}

	v.gameClient.MapEngine.AddEntity(monster)

	return nil
}
