package d2gamescreen

import (
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2daynight"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2lightmap"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2maprenderer"
)

const (
	act5Index        = 4
	act5MaxIntensity = 0xaa // ENVIRON_UpdateAmbientIntensity caps Act 5 at 0xaa
	levelFixedLight  = 0x78 // level 0x78 has a fixed intensity 200
	levelFixedValue  = 200
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

// advanceLighting feeds the map renderer the hero position and base light.
func (v *Game) advanceLighting() {
	if !v.mapRenderer.LightingEnabled() || v.localPlayer == nil {
		return
	}

	pos := v.localPlayer.Position.World()
	tilePos := v.localPlayer.Position.Tile()

	var cell d2lightmap.Cell

	if tile := v.gameClient.MapEngine.TileAt(int(tilePos.X()), int(tilePos.Y())); tile != nil {
		id := int(tile.RegionType)

		if lv, ok := v.asset.Records.Level.Details[id]; ok {
			cell = baseLight(lv.LightIntensity, lv.Red, lv.Green, lv.Blue, lv.Act, id, v.currentDayClock())
		} else {
			cell = baseLight(0, 0, 0, 0, 0, id, v.currentDayClock())
		}
	} else {
		cell = baseLight(0, 0, 0, 0, 0, 0, v.currentDayClock())
	}

	v.mapRenderer.SetLightInput(d2maprenderer.LightInput{HeroX: pos.X(), HeroY: pos.Y(), Base: cell})
}

func (v *Game) currentDayClock() *dayClock {
	if v.dayClock == nil {
		v.dayClock = newDayClock()
	}

	return v.dayClock
}
