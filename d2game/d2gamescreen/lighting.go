package d2gamescreen

import (
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2daynight"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2lightmap"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2maprenderer"
)

const (
	act5Index        = 4
	act5MaxIntensity = 0xaa // ENVIRON_UpdateAmbientIntensity caps Act 5 at 0xaa
	levelFixedLight  = 0x78 // level 0x78 has a fixed intensity 200
	levelFixedValue  = 200

	// warm flame colour of object lights (U: not in the notes)
	objectLightR, objectLightG, objectLightB = 255, 200, 130
)

// parseAutoTime parses OD2_AUTOTIME: a phase number 0..5 or a name (night,
// dawn, day, noon, dusk, evening), optionally followed by "@<degree>" to start
// that many degrees into the day (e.g. "dusk@185"). It returns the phase and
// the degree (-1 = start of the phase).
func parseAutoTime(spec string) (phase, degree int, ok bool) {
	spec = strings.ToLower(strings.TrimSpace(spec))
	if spec == "" {
		return 0, 0, false
	}

	degree = -1

	if i := strings.Index(spec, "@"); i >= 0 {
		d, err := strconv.Atoi(spec[i+1:])
		if err != nil {
			return 0, 0, false
		}

		degree, spec = d, spec[:i]
	}

	names := map[string]int{
		"night": d2daynight.PhaseNight0, "dawn": d2daynight.PhaseDawn, "morning": d2daynight.PhaseDawn,
		"day": d2daynight.PhaseDay, "noon": d2daynight.PhaseDay, "afternoon": d2daynight.PhaseDusk,
		"dusk": d2daynight.PhaseNight4, "evening": d2daynight.PhaseNight4, "midnight": d2daynight.PhaseNight5,
	}

	if p, found := names[spec]; found {
		return p, degree, true
	}

	p, err := strconv.Atoi(spec)
	if err != nil || p < 0 || p >= d2daynight.PhaseCount {
		return 0, 0, false
	}

	return p, degree, true
}

// applyAutoTime forces the day/night phase from OD2_AUTOTIME and freezes the
// clock there (for screenshots).
func (c *dayClock) applyAutoTime(spec string) {
	phase, degree, ok := parseAutoTime(spec)
	if !ok {
		return
	}

	c.env.SetPhase(phase)

	if degree >= 0 {
		c.env.SetPhaseAndTick(phase, degree*d2daynight.TicksPerDegreeReal)
	}

	c.frozen = true
}

// Intensity is the ambient intensity of the outdoor cycle.
func (c *dayClock) Intensity() int { return c.env.Intensity() }

// Ambient is the ambient colour of the outdoor cycle.
func (c *dayClock) Ambient() d2daynight.RGB { return c.env.Ambient() }

// baseLight is the light of the hero's level: Levels.txt Intensity/R/G/B when
// the level defines a colour, else the day/night ambient (renderer.md b4.1).
func baseLight(intensity, r, g, b, act, levelID int, env *dayClock) d2lightmap.Cell {
	if levelID == levelFixedLight {
		return d2lightmap.Cell{Intensity: levelFixedValue, R: 0xf5, G: 0xf0, B: 0xff}
	}

	if r != 0 || g != 0 || b != 0 {
		return d2lightmap.Cell{Intensity: clampByte(intensity), R: clampByte(r), G: clampByte(g), B: clampByte(b)}
	}

	i := env.Intensity()
	if act == act5Index && i > act5MaxIntensity {
		i = act5MaxIntensity
	}

	a := env.Ambient()

	return d2lightmap.Cell{Intensity: uint8(i), R: a.R, G: a.G, B: a.B}
}

func clampByte(v int) uint8 {
	if v < 0 {
		return 0
	}

	if v > 255 { //nolint:gomnd // byte
		return 255
	}

	return uint8(v)
}

// advanceLighting feeds the map renderer the hero position and base light of the hero's LEVEL.
// Bug fixed here: the level used to be looked up with tile.RegionType, which is the level TYPE
// (Levels.txt LevelType); e.g. the Act 2 desert (type 16) read the row of level 16 (Pit Level 2,
// intensity 0 + white RGB), so the whole Act 2 outdoors rendered pitch dark.
func (v *Game) advanceLighting() {
	if !v.mapRenderer.LightingEnabled() || v.localPlayer == nil {
		return
	}

	pos := v.localPlayer.Position.World()
	id := v.currentLevel()

	var cell d2lightmap.Cell

	if lv, ok := v.asset.Records.Level.Details[id]; ok {
		cell = baseLight(lv.LightIntensity, lv.Red, lv.Green, lv.Blue, lv.Act, id, v.currentDayClock())
	} else {
		cell = baseLight(0, 0, 0, 0, d2level.ActOfLevel(id)-1, id, v.currentDayClock())
	}

	if id != v.lightLogLevel {
		v.lightLogLevel = id
		v.Infof("LIGHT level=%d base intensity=%d rgb=%d,%d,%d", id, cell.Intensity, cell.R, cell.G, cell.B)
	}

	v.mapRenderer.SetLightInput(d2maprenderer.LightInput{HeroX: pos.X(), HeroY: pos.Y(), Base: cell, Extra: v.objectLights(pos.X(), pos.Y())})
}

// objectLights collects the lights of objects (torches, fires, shrines...) within the light map window of the hero.
func (v *Game) objectLights(hx, hy float64) []d2maprenderer.LightPoint {
	if v.gameClient == nil || v.gameClient.MapEngine == nil {
		return nil
	}

	var out []d2maprenderer.LightPoint

	const window = 14.0 // tiles; the light map covers 48 subtiles, objects farther out still reach into it

	for _, e := range v.gameClient.MapEngine.Entities() {
		ob, ok := e.(*d2mapentity.Object)
		if !ok {
			continue
		}

		r := ob.LightRadius()
		if r <= 0 {
			continue
		}

		x, y := ob.GetPositionF()
		if x < hx-window || x > hx+window || y < hy-window || y > hy+window {
			continue
		}

		out = append(out, d2maprenderer.LightPoint{X: x, Y: y, Radius: r, R: objectLightR, G: objectLightG, B: objectLightB})
	}

	return out
}

func (v *Game) currentDayClock() *dayClock {
	if v.dayClock == nil {
		v.dayClock = newDayClock()
	}

	return v.dayClock
}
