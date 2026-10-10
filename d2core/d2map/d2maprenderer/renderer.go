package d2maprenderer

import (
	"errors"
	"fmt"
	"image/color"
	"math"
	"strconv"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2ds1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
)

const (
	logPrefix = "Map Renderer"
)

const (
	screenMiddleX = 400
	two           = 2

	dbgOffsetXY   = 40
	dbgBoxWidth   = 220
	dbgBoxHeight  = 60
	dbgBoxPadding = 10

	dbgCollisionSize    = 5
	dbgCollisionOffsetX = -3
	dbgCollisionOffsetY = 4

	whiteHalfOpacity        = 0xffffff7f
	blackQuarterOpacity     = 0x00000040
	lightGreenFullOpacity   = 0x40ff00ff
	magentaFullOpacity      = 0xff00ffff
	yellowFullOpacity       = 0xffff00ff
	lightBlueQuarterOpacity = 0x5050ff32
	whiteQuarterOpacity     = 0xffffff64
	redQuarterOpacity       = 0x74000064

	subtilesPerTile    = 5
	orthoSubTileWidth  = 16
	orthoSubTileHeight = 8
	orthoTileWidth     = subtilesPerTile * orthoSubTileWidth
	orthoTileHeight    = subtilesPerTile * orthoSubTileHeight
)

// MapRenderer manages the game viewport and Camera. It requests tile and entity data from MapEngine and renders it.
type MapRenderer struct {
	asset               *d2asset.AssetManager
	renderer            d2interface.Renderer   // Used for drawing operations
	mapEngine           *d2mapengine.MapEngine // The map engine that is being rendered
	palette             d2interface.Palette    // The palette used for this map
	viewport            *Viewport              // Used for rendering offsets
	Camera              Camera                 // Used to determine where on the map we are rendering
	imageCacheRecords   map[uint32]d2interface.Surface
	blankWalls          map[uint32]bool // wall tiles without graphics (marker DT1 files, collision placeholders)
	blankShadows        map[uint32]bool // shadow tiles without graphics (never cached, never drawn)
	mapDebugVisLevel    int             // Map debug visibility index (0=none, 1=tiles, 2=sub-tiles)
	entityDebugVisLevel int             // Entity Debug visibility index (0=none, 1=vectors)
	lastFrameTime       float64         // The last time the map was rendered
	currentFrame        int             // Current render frame (for animations)
	light               *lighting
	entBuckets          map[[2]int]*entityBucket // per-frame entity index, see indexEntities
	entFree             []*entityBucket
	shadeVals           []color.RGBA // scratch for renderShadedImage
	stats               DrawStats
	roofCover           map[[2]int]bool // roof tiles fading out because the hero is under them

	*d2util.Logger
}

// CreateMapRenderer creates a new MapRenderer, sets the required fields and returns a pointer to it.
func CreateMapRenderer(asset *d2asset.AssetManager, renderer d2interface.Renderer,
	mapEngine *d2mapengine.MapEngine,
	term d2interface.Terminal, l d2util.LogLevel, startX, startY float64) *MapRenderer {
	result := &MapRenderer{
		asset:     asset,
		renderer:  renderer,
		mapEngine: mapEngine,
		viewport:  NewViewport(0, 0, 800, 600),
		light:     newLighting(),
	}

	result.Logger = d2util.NewLogger()
	result.Logger.SetPrefix(logPrefix)
	result.Logger.SetLevel(l)

	result.Camera = Camera{}
	rx, ry := result.WorldToOrtho(startX, startY)
	startPosition := d2vector.NewPosition(rx, ry)
	result.Camera.position = &startPosition
	result.viewport.SetCamera(&result.Camera)

	var name, desc, level string

	name, desc = "mapdebugvis", "set map debug visualization level"
	level = "level"

	if err := term.Bind(name, desc, []string{level}, result.commandMapDebugVis); err != nil {
		result.Errorf("could not bind the mapdebugvis action, err: %v", err)
	}

	name, desc = "entitydebugvis", "set entity debug visualization level"

	if err := term.Bind(name, desc, []string{level}, result.commandEntityDebugVis); err != nil {
		result.Errorf("could not bind the entitydebugvis action, err: %v", err)
	}

	if mapEngine.LevelType().ID != 0 {
		result.generateTileCache()
	}

	return result
}

// UnbindTerminalCommands unbinds commands from the terminal
func (mr *MapRenderer) UnbindTerminalCommands(term d2interface.Terminal) error {
	return term.Unbind("mapdebugvis", "entitydebugvis")
}

func (mr *MapRenderer) commandMapDebugVis(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("invalid argument supplied")
	}

	level, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid argument supplied")
	}

	mr.mapDebugVisLevel = level

	return nil
}

func (mr *MapRenderer) commandEntityDebugVis(args []string) error {
	level, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid argument supplied")
	}

	mr.entityDebugVisLevel = level

	return nil
}

// RegenerateTileCache calls MapRenderer.generateTileCache().
func (mr *MapRenderer) RegenerateTileCache() {
	// the cache key is (style, sequence, type, index) without the DT1 file or the palette of the
	// level, so images of the previous level would be drawn for the new one
	mr.InvalidateImageCache()
	mr.generateTileCache()
}

// SetMapEngine sets the MapEngine this renderer is rendering.
func (mr *MapRenderer) SetMapEngine(mapEngine *d2mapengine.MapEngine) {
	mr.mapEngine = mapEngine
	mr.InvalidateImageCache()
	mr.generateTileCache()
}

// Render determines the width and height of map tiles that should be rendered. The following render passes are
// made in succession (the real game's order, renderer.md b1: all floors before any wall):
//
// Pass 1: Floor tiles of the whole view, then lower wall tiles of the whole view.
//
// Pass 2: Per tile: shadows, then entities below walls.
//
// Pass 3: Upper wall tiles and entities above walls.
//
// Pass 4: Roof tiles.
func (mr *MapRenderer) Render(target d2interface.Surface) {
	// https://github.com/OpenDiablo2/OpenDiablo2/issues/789
	// Prevents concurrent map read & write exceptions that otherwise occur when we join a TCP game
	// as a remote client, due to rendering before we have handled the GenerateMapPacket.
	if mr.mapEngine.IsLoading {
		return
	}

	mapSize := mr.mapEngine.Size()

	stxf, styf := mr.viewport.ScreenToWorld(screenMiddleX, -200)
	etxf, etyf := mr.viewport.ScreenToWorld(screenMiddleX, 1050)

	startX := int(math.Max(0, math.Floor(stxf)))
	startY := int(math.Max(0, math.Floor(styf)))

	endX := int(math.Min(float64(mapSize.Width), math.Ceil(etxf)))
	endY := int(math.Min(float64(mapSize.Height), math.Ceil(etyf)))

	mr.stats = DrawStats{Lit: mr.light.active()}
	if mr.light.active() {
		mr.stats.LightSources = 1 + len(mr.light.input.Extra)
	}

	mr.indexEntities()
	mr.renderPass1(target, startX, startY, endX, endY)
	mr.renderPass2(target, startX, startY, endX, endY)

	if mr.mapDebugVisLevel > 0 {
		mr.renderMapDebug(mr.mapDebugVisLevel, target, startX, startY, endX, endY)
	}

	mr.renderPass3(target, startX, startY, endX, endY)
	mr.renderPass4(target, startX, startY, endX, endY)

	if mr.entityDebugVisLevel > 0 {
		mr.renderEntityDebug(target)
	}
}

// MoveCameraTo sets the position of the Camera to the given x and y coordinates.
func (mr *MapRenderer) MoveCameraTo(position *d2vector.Position) {
	mr.Camera.MoveTo(position)
}

// MoveCameraBy adds the given vector to the current position of the Camera.
func (mr *MapRenderer) MoveCameraBy(vector *d2vector.Vector) {
	mr.Camera.MoveBy(vector)
}

// MoveCameraTargetBy adds the given vector to the current position of the Camera.
func (mr *MapRenderer) MoveCameraTargetBy(vector *d2vector.Vector) {
	mr.Camera.MoveTargetBy(vector)
}

// ScreenToWorld returns the world position for the given screen (pixel) position.
func (mr *MapRenderer) ScreenToWorld(x, y int) (worldX, worldY float64) {
	return mr.viewport.ScreenToWorld(x, y)
}

// ScreenToOrtho returns the orthogonal position, without accounting for the isometric angle, for the given screen
// (pixel) position.
func (mr *MapRenderer) ScreenToOrtho(x, y int) (orthoX, orthoY float64) {
	return mr.viewport.ScreenToOrtho(x, y)
}

// WorldToOrtho returns the orthogonal position for the given isometric world position.
func (mr *MapRenderer) WorldToOrtho(x, y float64) (orthoX, orthoY float64) {
	return mr.viewport.WorldToOrtho(x, y)
}

// Floors of the whole view first, then lower walls (a wall must never be
// overpainted by the floor of a later tile).
func (mr *MapRenderer) renderPass1(target d2interface.Surface, startX, startY, endX, endY int) {
	for tileY := startY; tileY < endY; tileY++ {
		for tileX := startX; tileX < endX; tileX++ {
			tile := mr.mapEngine.TileAt(tileX, tileY)
			mr.viewport.PushTranslationWorld(float64(tileX), float64(tileY))

			for _, floor := range tile.Components.Floors {
				if !floor.Hidden() && floor.Prop1 != 0 {
					mr.renderFloor(floor, target, tileX, tileY)
				}
			}

			mr.viewport.PopTranslation()
		}
	}

	for tileY := startY; tileY < endY; tileY++ {
		for tileX := startX; tileX < endX; tileX++ {
			tile := mr.mapEngine.TileAt(tileX, tileY)
			mr.viewport.PushTranslationWorld(float64(tileX), float64(tileY))

			for idx, wall := range tile.Components.Walls {
				if !wall.Hidden() && wall.Prop1 != 0 && wall.Type.LowerWall() {
					mr.renderWall(wall, mr.viewport, target, tileX, tileY, idx, false)
				}
			}

			mr.viewport.PopTranslation()
		}
	}
}

// Entities below walls.
func (mr *MapRenderer) renderPass2(target d2interface.Surface, startX, startY, endX, endY int) {
	for tileY := startY; tileY < endY; tileY++ {
		for tileX := startX; tileX < endX; tileX++ {
			mr.viewport.PushTranslationWorld(float64(tileX), float64(tileY))

			tile := mr.mapEngine.TileAt(tileX, tileY)
			for _, shadow := range tile.Components.Shadows {
				if !shadow.Hidden() && shadow.Prop1 != 0 {
					mr.renderShadow(shadow, target, tileX, tileY)
				}
			}

			tileEnt := mr.getEntitiesBelowWalls(tileX, tileY)

			for subY := 0; subY < 5; subY++ {
				for subX := 0; subX < 5; subX++ {
					for _, mapEntity := range tileEnt {
						pos := mapEntity.GetPosition()
						if (int(pos.SubTileOffset().X()) != subX) || (int(pos.SubTileOffset().Y()) != subY) {
							continue
						}

						target.PushTranslation(mr.viewport.GetTranslationScreen())
						mr.renderLitEntity(target, mapEntity)
						target.Pop()
					}
				}
			}

			mr.viewport.PopTranslation()
		}
	}
}

// entityBucket holds the entities standing on one tile, split by layer.
type entityBucket struct {
	below []d2interface.MapEntity // layer 1: drawn below the walls
	above []d2interface.MapEntity // every other layer: drawn above the walls
}

// indexEntities sorts all entities into per-tile buckets once per frame. The passes used to scan
// every entity for every visible tile (tiles x entities map iterations per frame). The buckets
// and their slices are reused between frames. Entities are visited in map order, as before, so
// the order of entities that share a tile and a subtile is as unspecified as it always was.
func (mr *MapRenderer) indexEntities() {
	if mr.entBuckets == nil {
		mr.entBuckets = make(map[[2]int]*entityBucket)
	}

	for key, b := range mr.entBuckets {
		b.below, b.above = b.below[:0], b.above[:0]

		mr.entFree = append(mr.entFree, b)

		delete(mr.entBuckets, key)
	}

	for _, mapEntity := range mr.mapEngine.Entities() {
		pos := mapEntity.GetPosition()
		vec := pos.World()
		key := [2]int{int(vec.X()), int(vec.Y())}

		b := mr.entBuckets[key]
		if b == nil {
			if n := len(mr.entFree); n > 0 {
				b, mr.entFree = mr.entFree[n-1], mr.entFree[:n-1]
			} else {
				b = &entityBucket{}
			}

			mr.entBuckets[key] = b
		}

		if mapEntity.GetLayer() == 1 {
			b.below = append(b.below, mapEntity)
		} else {
			b.above = append(b.above, mapEntity)
		}
	}
}

func (mr *MapRenderer) getEntitiesBelowWalls(tileX, tileY int) []d2interface.MapEntity {
	if b := mr.entBuckets[[2]int{tileX, tileY}]; b != nil {
		return b.below
	}

	return nil
}

// Upper wall tiles and entities above walls.
func (mr *MapRenderer) renderPass3(target d2interface.Surface, startX, startY, endX, endY int) {
	for tileY := startY; tileY < endY; tileY++ {
		for tileX := startX; tileX < endX; tileX++ {
			tile := mr.mapEngine.TileAt(tileX, tileY)
			mr.viewport.PushTranslationWorld(float64(tileX), float64(tileY))
			mr.renderTilePass2(tile, target, tileX, tileY)

			entities := mr.getEntitiesAboveWalls(tileX, tileY)

			for subY := 0; subY < 5; subY++ {
				for subX := 0; subX < 5; subX++ {
					for _, entity := range entities {
						pos := entity.GetPosition()
						if (int(pos.SubTileOffset().X()) != subX) || (int(pos.SubTileOffset().Y()) != subY) {
							continue
						}

						target.PushTranslation(mr.viewport.GetTranslationScreen())
						mr.renderLitEntity(target, entity)
						target.Pop()
					}
				}
			}

			mr.viewport.PopTranslation()
		}
	}
}

func (mr *MapRenderer) getEntitiesAboveWalls(tileX, tileY int) []d2interface.MapEntity {
	if b := mr.entBuckets[[2]int{tileX, tileY}]; b != nil {
		return b.above
	}

	return nil
}

// Roof tiles.
func (mr *MapRenderer) renderPass4(target d2interface.Surface, startX, startY, endX, endY int) {
	mr.updateRoofCover()

	for tileY := startY; tileY < endY; tileY++ {
		for tileX := startX; tileX < endX; tileX++ {
			tile := mr.mapEngine.TileAt(tileX, tileY)
			mr.viewport.PushTranslationWorld(float64(tileX), float64(tileY))
			mr.renderTilePass3(tile, target, tileX, tileY)
			mr.viewport.PopTranslation()
		}
	}
}

func (mr *MapRenderer) renderTilePass2(tile *d2mapengine.MapTile, target d2interface.Surface, tileX, tileY int) {
	for idx, wall := range tile.Components.Walls {
		if !wall.Hidden() && wall.Type.UpperWall() {
			mr.renderWall(wall, mr.viewport, target, tileX, tileY, idx, true)
		}
	}
}

func (mr *MapRenderer) renderTilePass3(tile *d2mapengine.MapTile, target d2interface.Surface, tileX, tileY int) {
	for idx, wall := range tile.Components.Walls {
		if wall.Type == d2enum.TileRoof {
			mr.renderRoof(wall, mr.viewport, target, tileX, tileY, idx)
		}
	}
}

func (mr *MapRenderer) renderFloor(tile d2ds1.Tile, target d2interface.Surface, tileX, tileY int) {
	var img d2interface.Surface
	if !tile.Animated {
		img = mr.getImageCacheRecord(tile.Style, tile.Sequence, 0, tile.RandomIndex)
	} else {
		img = mr.getImageCacheRecord(tile.Style, tile.Sequence, 0, byte(mr.currentFrame))
	}

	if img == nil {
		mr.Warningf("Render called on uncached floor {%v,%v}", tile.Style, tile.Sequence)
		return
	}

	mr.stats.Floors++

	mr.viewport.PushTranslationOrtho(-80, float64(tile.YAdjust))
	defer mr.viewport.PopTranslation()

	target.PushTranslation(mr.viewport.GetTranslationScreen())
	defer target.Pop()

	if mr.light.active() {
		mr.renderShadedImage(target, img, floorShadeCols, floorShadeRows, 1, shadeFloor, tileX, tileY)

		return
	}

	target.Render(img)
}

func (mr *MapRenderer) renderWall(tile d2ds1.Tile, viewport *Viewport, target d2interface.Surface,
	tileX, tileY, idx int, upper bool) {
	img := mr.getImageCacheRecord(tile.Style, tile.Sequence, tile.Type, tile.RandomIndex)
	if img == nil {
		if !mr.blankWalls[blankWallKey(&tile)] {
			mr.Warningf("Render called on uncached wall {%v,%v,%v}", tile.Style, tile.Sequence, tile.Type)
		}

		return
	}

	if upper {
		mr.stats.UpperWalls++
	} else {
		mr.stats.Walls++
	}

	viewport.PushTranslationOrtho(-80, float64(tile.YAdjust))
	defer viewport.PopTranslation()

	target.PushTranslation(viewport.GetTranslationScreen())
	defer target.Pop()

	if !mr.light.active() {
		target.Render(img)
		return
	}

	alpha := 1.0
	if upper {
		sx, sy := viewport.GetTranslationScreen()
		alpha = mr.light.fadeFor(wallKey{tileX, tileY, idx}, mr.heroBehind(tileX, tileY, sx, sy, img))
		if alpha < 1 {
			mr.stats.WallsFading++
		}
	}

	mr.renderShadedImage(target, img, wallShadeCols, wallShadeRows, alpha, shadeWall, tileX, tileY)
}

// renderRoof draws a roof tile with the flat frame ambient (roofs sample no
// light map in the real game).
func (mr *MapRenderer) renderRoof(tile d2ds1.Tile, viewport *Viewport, target d2interface.Surface,
	tileX, tileY, idx int) {
	img := mr.getImageCacheRecord(tile.Style, tile.Sequence, tile.Type, tile.RandomIndex)
	if img == nil {
		mr.Warningf("Render called on uncached wall {%v,%v,%v}", tile.Style, tile.Sequence, tile.Type)
		return
	}

	mr.stats.Roofs++

	viewport.PushTranslationOrtho(-80, float64(tile.YAdjust))
	defer viewport.PopTranslation()

	target.PushTranslation(viewport.GetTranslationScreen())
	defer target.Pop()

	if mr.light.active() {
		alpha := mr.light.roofAlpha(wallKey{tileX, tileY, idx}, mr.roofCover[[2]int{tileX, tileY}])
		if alpha < 1 {
			mr.stats.RoofsFading++
		}

		if alpha <= 0 {
			return
		}

		target.PushColor(withAlpha(mr.light.ambientTint(), alpha))
		defer target.Pop()
	}

	target.Render(img)
}

// updateRoofCover recomputes which roof tiles the hero is under (see roof_fade.go).
func (mr *MapRenderer) updateRoofCover() {
	mr.roofCover = nil
	if !mr.light.active() {
		return
	}

	hx, hy := int(mr.light.input.HeroX), int(mr.light.input.HeroY)
	size := mr.mapEngine.Size()

	mr.roofCover = roofRegion(func(x, y int) bool {
		if x < 0 || y < 0 || x >= size.Width || y >= size.Height {
			return false
		}

		for _, w := range mr.mapEngine.TileAt(x, y).Components.Walls {
			if w.Type == d2enum.TileRoof && !w.Hidden() && w.Prop1 != 0 {
				return true
			}
		}

		return false
	}, hx, hy)
}

// heroBehind reports whether the hero is hidden by the wall tile whose image is
// drawn at screen position (sx, sy): the wall is nearer to the camera than the
// hero and the hero's screen position lies inside the wall image. This is a
// heuristic; the real game flags the covering tiles in the room code, which
// the notes did not locate (U).
func (mr *MapRenderer) heroBehind(tileX, tileY, sx, sy int, img d2interface.Surface) bool {
	in := mr.light.input
	if float64(tileX+tileY) <= in.HeroX+in.HeroY-0.5 {
		return false
	}

	hx, hy := mr.viewport.WorldToScreenF(in.HeroX, in.HeroY)
	w, h := img.GetSize()

	// a wall only hides what is behind it: hero feet and body inside the image
	return hx > float64(sx)+8 && hx < float64(sx+w)-8 && hy-30 > float64(sy) && hy < float64(sy+h)
}

// renderLitEntity draws an entity tinted by one light map sample at its feet.
func (mr *MapRenderer) renderLitEntity(target d2interface.Surface, e d2interface.MapEntity) {
	mr.stats.Entities++

	if !mr.light.active() {
		e.Render(target)
		return
	}

	x, y := e.GetPositionF()
	target.PushColor(mr.light.tintAt(x*subtilesPerTile, y*subtilesPerTile))
	e.Render(target)
	target.Pop()
}

// shadowAlpha is the opacity of DT1 shadow tiles (the original uses a PL2 blend table; 160/255 is
// the approximation chosen in renderer.md section c, item 4).
const shadowAlpha = 160.0 / 255.0

func (mr *MapRenderer) renderShadow(tile d2ds1.Tile, target d2interface.Surface, tileX, tileY int) {
	img := mr.getImageCacheRecord(tile.Style, tile.Sequence, 13, tile.RandomIndex)
	if img == nil {
		// shadow tiles without graphics (width or height 0) are never cached;
		// the exact room tiles of the outdoor generator contain such markers
		key := uint32(tile.Style)<<16 | uint32(tile.Sequence)<<8 | uint32(tile.RandomIndex)

		if mr.blankShadows == nil {
			mr.blankShadows = map[uint32]bool{}
		}

		blank, known := mr.blankShadows[key]
		if !known {
			opts := mr.mapEngine.GetTiles(int(tile.Style), int(tile.Sequence), d2enum.TileShadow)
			blank = opts != nil && int(tile.RandomIndex) < len(opts) &&
				(opts[tile.RandomIndex].Width == 0 || opts[tile.RandomIndex].Height == 0)
			mr.blankShadows[key] = blank
		}

		if blank {
			return
		}

		mr.Warningf("Render called on uncached shadow {%v,%v}", tile.Style, tile.Sequence)

		return
	}

	mr.stats.Shadows++

	defer mr.viewport.PushTranslationOrtho(-80, float64(tile.YAdjust)).PopTranslation()

	target.PushTranslation(mr.viewport.GetTranslationScreen())
	defer target.Pop()

	if mr.light.active() {
		// lit like the floor underneath
		mr.renderShadedImage(target, img, floorShadeCols, floorShadeRows, shadowAlpha, shadeFloor, tileX, tileY)

		return
	}

	target.PushColor(color.RGBA{R: 255, G: 255, B: 255, A: 160}) //nolint:gomnd // Not a magic number...
	defer target.Pop()

	target.Render(img)
}

func (mr *MapRenderer) renderMapDebug(mapDebugVisLevel int, target d2interface.Surface, startX, startY, endX, endY int) {
	for tileY := startY; tileY < endY; tileY++ {
		for tileX := startX; tileX < endX; tileX++ {
			mr.viewport.PushTranslationWorld(float64(tileX), float64(tileY))
			mr.renderTileDebug(tileX, tileY, mapDebugVisLevel, target)
			mr.viewport.PopTranslation()
		}
	}
}

//nolint:funlen // doesn't make sense to split this function
func (mr *MapRenderer) renderEntityDebug(target d2interface.Surface) {
	entities := mr.mapEngine.Entities()

	for idx := range entities {
		e := entities[idx]
		pos := e.GetPosition()
		world := pos
		x, y := world.X()/subtilesPerTile, world.Y()/subtilesPerTile
		velocity := e.GetVelocity()
		velocity = *velocity.Clone()
		vx, vy := mr.viewport.WorldToOrtho(velocity.X(), velocity.Y())
		screenX, screenY := mr.viewport.WorldToScreen(x, y)

		offX, offY := dbgOffsetXY, -dbgOffsetXY

		entScreenXf, entScreenYf := mr.WorldToScreenF(e.GetPositionF())
		entScreenX := int(math.Floor(entScreenXf))
		entScreenY := int(math.Floor(entScreenYf))
		entityWidth, entityHeight := e.GetSize()
		halfWidth, halfHeight := entityWidth/two, entityHeight/two
		l, r := entScreenX-halfWidth, entScreenX+halfWidth
		t, b := entScreenY-halfHeight, entScreenY+halfHeight
		mx, my := mr.renderer.GetCursorPos()
		xWithin := (l <= mx) && (r >= mx)
		yWithin := (t <= my) && (b >= my)
		within := xWithin && yWithin

		boxLineColor := d2util.Color(magentaFullOpacity)
		boxHoverColor := d2util.Color(yellowFullOpacity)

		boxColor := boxLineColor

		if within {
			boxColor = boxHoverColor
		}

		stack := 0
		// box
		mr.viewport.PushTranslationWorld(x, y)

		target.PushTranslation(screenX, screenY)
		stack++

		target.PushTranslation(-halfWidth, -halfHeight)
		stack++

		target.DrawLine(0, entityHeight, boxColor)
		target.DrawLine(entityWidth, 0, boxColor)

		target.PushTranslation(entityWidth, entityHeight)
		stack++

		target.DrawLine(-entityWidth, 0, boxColor)
		target.DrawLine(0, -entityHeight, boxColor)
		target.PopN(stack)
		mr.viewport.PopTranslation()

		// hover
		if within {
			mr.viewport.PushTranslationWorld(x, y)
			target.PushTranslation(screenX, screenY)
			target.DrawLine(offX, offY, d2util.Color(whiteHalfOpacity))
			target.PushTranslation(offX+dbgBoxPadding, offY-dbgBoxPadding*two)
			target.PushTranslation(-dbgOffsetXY, -dbgOffsetXY)
			target.DrawRect(dbgBoxWidth, dbgBoxHeight, d2util.Color(blackQuarterOpacity))
			target.Pop()
			target.DrawTextf("World (%.2f, %.2f)\nVelocity (%.2f, %.2f)", x, y, vx, vy)
			target.Pop()
			target.DrawLine(int(vx), int(vy), d2util.Color(lightGreenFullOpacity))
			target.Pop()
			mr.viewport.PopTranslation()
		}
	}
}

// WorldToScreen returns the screen (pixel) position for the given isometric world position as two ints.
func (mr *MapRenderer) WorldToScreen(x, y float64) (screenX, screenY int) {
	return mr.viewport.WorldToScreen(x, y)
}

// WorldToScreenF returns the screen (pixel) position for the given isometric world position as two float64s.
func (mr *MapRenderer) WorldToScreenF(x, y float64) (screenX, screenY float64) {
	return mr.viewport.WorldToScreenF(x, y)
}

func (mr *MapRenderer) renderTileDebug(ax, ay, debugVisLevel int, target d2interface.Surface) {
	subTileColor := d2util.Color(lightBlueQuarterOpacity)
	tileColor := d2util.Color(whiteQuarterOpacity)
	tileCollisionColor := d2util.Color(redQuarterOpacity)

	screenX1, screenY1 := mr.viewport.WorldToScreen(float64(ax), float64(ay))
	screenX2, screenY2 := mr.viewport.WorldToScreen(float64(ax+1), float64(ay))
	screenX3, screenY3 := mr.viewport.WorldToScreen(float64(ax), float64(ay+1))

	target.PushTranslation(screenX1, screenY1)
	defer target.Pop()

	target.DrawLine(screenX2-screenX1, screenY2-screenY1, tileColor)
	target.DrawLine(screenX3-screenX1, screenY3-screenY1, tileColor)
	target.PushTranslation(-10, 10)
	target.DrawTextf("%v, %v", ax, ay)
	target.Pop()

	if debugVisLevel > 1 {
		for i := 1; i <= 4; i++ {
			x2 := i * orthoSubTileWidth
			y2 := i * orthoSubTileHeight

			target.PushTranslation(-x2, y2)
			target.DrawLine(orthoTileWidth, orthoTileHeight, subTileColor)
			target.Pop()

			target.PushTranslation(x2, y2)
			target.DrawLine(-orthoTileWidth, orthoTileHeight, subTileColor)
			target.Pop()
		}

		tile := mr.mapEngine.TileAt(ax, ay)

		for i, wall := range tile.Components.Walls {
			if wall.Type.Special() {
				target.PushTranslation(-20, 10+(i+1)*14) // nolint:gomnd // just for debug
				target.DrawTextf("s: %v-%v", wall.Style, wall.Sequence)
				target.Pop()
			}
		}

		for yy := 0; yy < 5; yy++ {
			for xx := 0; xx < 5; xx++ {
				isoX := (xx - yy) * orthoSubTileWidth
				isoY := (xx + yy) * orthoSubTileHeight

				blocked := tile.GetSubTileFlags(xx, yy).BlockWalk

				if blocked {
					target.PushTranslation(isoX+dbgCollisionOffsetX, isoY+dbgCollisionOffsetY)
					target.DrawRect(dbgCollisionSize, dbgCollisionSize, tileCollisionColor)
					target.Pop()
				}
			}
		}
	}
}

const (
	frameOverflow = 10
	frameLength   = 1.0 / frameOverflow
)

// Advance is called once per frame and maintains the MapRenderer's previous
// render timestamp and current frame.
func (mr *MapRenderer) Advance(elapsed float64) {
	mr.lastFrameTime += elapsed
	framesAdvanced := int(mr.lastFrameTime / frameLength)
	mr.lastFrameTime -= float64(framesAdvanced) * frameLength

	mr.currentFrame += framesAdvanced
	if mr.currentFrame >= frameOverflow {
		mr.currentFrame = 0
	}

	mr.Camera.Advance(elapsed)
	mr.advanceLighting(elapsed)
}

func (mr *MapRenderer) loadPaletteForAct(levelType d2enum.RegionIdType) (d2interface.Palette,
	error) {
	var palettePath string

	switch levelType {
	case d2enum.RegionAct1Town, d2enum.RegionAct1Wilderness, d2enum.RegionAct1Cave, d2enum.RegionAct1Crypt,
		d2enum.RegionAct1Monestary, d2enum.RegionAct1Courtyard, d2enum.RegionAct1Barracks,
		d2enum.RegionAct1Jail, d2enum.RegionAct1Cathedral, d2enum.RegionAct1Catacombs, d2enum.RegionAct1Tristram:
		palettePath = d2resource.PaletteAct1
	case d2enum.RegionAct2Town, d2enum.RegionAct2Sewer, d2enum.RegionAct2Harem, d2enum.RegionAct2Basement,
		d2enum.RegionAct2Desert, d2enum.RegionAct2Tomb, d2enum.RegionAct2Lair, d2enum.RegionAct2Arcane:
		palettePath = d2resource.PaletteAct2
	case d2enum.RegionAct3Town, d2enum.RegionAct3Jungle, d2enum.RegionAct3Kurast, d2enum.RegionAct3Spider,
		d2enum.RegionAct3Dungeon, d2enum.RegionAct3Sewer:
		palettePath = d2resource.PaletteAct3
	case d2enum.RegionAct4Town, d2enum.RegionAct4Mesa, d2enum.RegionAct4Lava, d2enum.RegionAct5Lava:
		palettePath = d2resource.PaletteAct4
	case d2enum.RegonAct5Town, d2enum.RegionAct5Siege, d2enum.RegionAct5Barricade, d2enum.RegionAct5Temple,
		d2enum.RegionAct5IceCaves, d2enum.RegionAct5Baal:
		palettePath = d2resource.PaletteAct5
	default:
		return nil, errors.New("failed to find palette for region")
	}

	mr.loadShadeTable(palettePath)

	return mr.asset.LoadPalette(palettePath)
}

// ViewportToLeft moves the viewport to the left.
func (mr *MapRenderer) ViewportToLeft() {
	mr.viewport.toLeft()
}

// ViewportToRight moves the viewport to the right.
func (mr *MapRenderer) ViewportToRight() {
	mr.viewport.toRight()
}

// ViewportDefault resets the viewport to it's default position.
func (mr *MapRenderer) ViewportDefault() {
	mr.viewport.resetAlign()
}

// SetCameraTarget sets the Camera target
func (mr *MapRenderer) SetCameraTarget(position *d2vector.Position) {
	mr.Camera.SetTarget(position)
}

// SetCameraPosition sets the Camera position
func (mr *MapRenderer) SetCameraPosition(position *d2vector.Position) {
	mr.Camera.MoveTo(position)
}

// InvalidateImageCache the global region image cache. Call this when you are changing regions.
func (mr *MapRenderer) InvalidateImageCache() {
	mr.imageCacheRecords = nil
}

func (mr *MapRenderer) getImageCacheRecord(style, sequence byte, tileType d2enum.TileType, randomIndex byte) d2interface.Surface {
	lookupIndex := uint32(style)<<24 | uint32(sequence)<<16 | uint32(tileType)<<8 | uint32(randomIndex)
	return mr.imageCacheRecords[lookupIndex]
}

func (mr *MapRenderer) setImageCacheRecord(style, sequence byte, tileType d2enum.TileType, randomIndex byte, image d2interface.Surface) {
	lookupIndex := uint32(style)<<24 | uint32(sequence)<<16 | uint32(tileType)<<8 | uint32(randomIndex)

	if mr.imageCacheRecords == nil {
		mr.imageCacheRecords = make(map[uint32]d2interface.Surface)
	}

	mr.imageCacheRecords[lookupIndex] = image
}
