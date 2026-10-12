package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2automap"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2config"
)

// Persistence of the revealed cells. The exe writes them to sidecar files next to the character
// (d2automap.Sidecar). It saves a level's cells when the hero leaves the level and reloads them when he
// comes back (AUTOMAP_FindOrCreateLevelLayer 0x4543e0); the fork also saves with the character (autosave, exit).

// automapOptionKeys maps the F9..F12 option names to the config.json option keys. The exe stores these four
// in the registry (AutoMapFade, AutoMap Centers, AutoMap Party, AutoMap Party Names); the fork uses config.json.
var automapOptionKeys = map[string]string{
	"center": d2config.OptAutomapCenter, "fade": d2config.OptAutomapFade,
	"party": d2config.OptAutomapShowParty, "names": d2config.OptAutomapNames,
}

// SetPersistence tells the automap where the character's sidecar files are. An invalid sidecar (no folder or
// no name: a remote hero) keeps the automap in memory only.
func (a *Automap) SetPersistence(sc d2automap.Sidecar) {
	a.sidecar = sc
	a.saved = map[int]int{}
	a.loadTried = map[int]bool{}
}

// loadLevel brings the stored cells of a level into the store the first time the level is used.
func (a *Automap) loadLevel(id int) {
	if !a.sidecar.Valid() || a.loadTried[id] || a.store.Has(id) {
		return
	}

	a.loadTried[id] = true

	m := a.sidecar.LoadLevel(id)
	if m == nil {
		return
	}

	a.store.Set(id, m)
	a.saved[id] = m.Count()
	a.Infof("AUTOMAP loaded level=%d cells=%d from=%s", id, m.Count(), a.sidecar.Name)
}

// Flush writes the levels whose cells changed since they were loaded or last written. Errors are logged, never
// fatal (the automap is a convenience).
func (a *Automap) Flush() {
	if !a.sidecar.Valid() {
		return
	}

	dirty := map[int]*d2automap.Model{}
	cells := 0

	for _, id := range a.store.IDs() {
		m := a.store.Level(id)
		if n, ok := a.saved[id]; (ok && n == m.Count()) || (!ok && m.Count() == 0) {
			continue
		}

		dirty[id] = m
		cells += m.Count()
	}

	if len(dirty) == 0 {
		return
	}

	if err := a.sidecar.SaveLevels(dirty); err != nil {
		a.Warningf("AUTOMAP save: %v", err)

		return
	}

	for id, m := range dirty {
		a.saved[id] = m.Count()
	}

	a.Infof("AUTOMAP saved levels=%d cells=%d to=%s.map", len(dirty), cells, a.sidecar.Name)
}

// ToggleOption flips one of the F9..F12 options (center, fade, party, names) and stores it in config.json.
// It returns the new value.
func (a *Automap) ToggleOption(key string) bool {
	on := !a.option(key)
	delete(a.override, key)

	if id := automapOptionKeys[key]; id != "" && optionsBackend != nil {
		i := 0 // the NO/YES rows: index 1 is on
		if on {
			i = 1
		}

		if err := optionsBackend.Change(id, i); err != nil {
			a.Warningf("AUTOMAP option %s: %v", key, err)
			a.override[key] = on
		}
	} else {
		a.override[key] = on
	}

	a.Infof("AUTOMAP option %s=%v (stored)", key, on)

	return on
}
