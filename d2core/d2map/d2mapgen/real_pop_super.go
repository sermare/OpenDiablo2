package d2mapgen

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgpop"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monreg"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// fallbackSuper makes the super unique of a preset node as a DS1 super unique placement (see createSuper) and
// reports whether it did.
func (p *popLevel) fallbackSuper(game *d2monreg.Game, key string, rec d2monreg.SuperRec, st *d2records.MonStatRecord,
	rq drlgpop.Request) bool {
	if !superFallbackAllowed(game.Difficulty, game.SuperMade(rec.HcIdx), rec.Stacks) {
		return false
	}

	npc, err := p.g.safeNPC(rq.X, rq.Y, st)
	if err != nil {
		p.g.Warningf("real population: could not place super unique %s: %v", key, err)

		return false
	}

	npc.SetSuperUnique(key)
	p.g.engine.AddEntity(npc)
	p.g.Infof("real population: super unique %s found no free spot by the placement rules; placed on the DS1 spot (%d,%d)",
		key, rq.X, rq.Y)

	p.stats.super++

	return true
}

// superFallbackAllowed says whether a refused super unique may still be placed: not above Hell and not for a hcIdx
// that the game made already (unless it stacks), the two refusals of the original.
func superFallbackAllowed(difficulty int, made, stacks bool) bool {
	return difficulty <= 2 && (stacks || !made)
}
