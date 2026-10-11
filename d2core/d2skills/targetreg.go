package d2skills

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2summon"

// Engine side of the target registry (GAME\Targets.cpp, d2summon.Targets): converted monsters sit in their
// caster's owner bucket, confused and attracted ones in the global buckets. The bucket is cleaned when the
// conversion or curse ends, when the monster dies and when the owner dies. UNVERIFIED: which of the two
// global buckets the exe uses for Attract and Confuse (8 for Attract and 9 for Confuse here).

const (
	attractBucket = 8
	confuseBucket = 9
)

// regNum is the registry handle of a unit id (the registry works with numbers).
func (e *Engine) regNum(id string) uint32 {
	if e.regIDs == nil {
		e.regIDs = map[string]uint32{}
		e.reg = d2summon.NewTargets()
	}

	n, ok := e.regIDs[id]
	if !ok {
		e.regNext++
		n = e.regNext
		e.regIDs[id] = n
	}

	return n
}

// Registry is the engine's target registry (for tests and the log).
func (e *Engine) Registry() *d2summon.Targets {
	e.regNum("")

	return e.reg
}

// regOwner makes sure the caster has an owner bucket and returns its slot.
func (e *Engine) regOwner(id string) (int, bool) {
	n := e.regNum(id)
	if s := e.reg.BucketOf(n); s != d2summon.NoBucket {
		return s, true
	}

	s, err := e.reg.AddPlayer(n)

	return s, err == nil
}

// regConverted files a converted monster under its caster.
func (e *Engine) regConverted(owner, monster string) {
	if slot, ok := e.regOwner(owner); ok {
		_ = e.reg.AddEntry(e.regNum(monster), "convert", slot)
	}
}

// regForced files a confused or attracted monster in a global bucket until the curse ends.
func (e *Engine) regForced(monster string, attract bool) {
	e.regNum(monster)

	slot := confuseBucket
	if attract {
		slot = attractBucket
	}

	_ = e.reg.AddGlobal(e.regNum(monster), "curse", slot)
}

// regRelease takes a monster out of the registry (conversion revert, curse end or death).
func (e *Engine) regRelease(monster string) {
	if e.reg == nil {
		return
	}

	if n, ok := e.regIDs[monster]; ok {
		e.reg.Remove(n)
	}
}

// regOwnerDied releases everything the dead hero holds.
func (e *Engine) regOwnerDied(owner string) {
	if e.reg == nil {
		return
	}

	if n, ok := e.regIDs[owner]; ok {
		e.reg.OwnerDied(n)
	}
}
