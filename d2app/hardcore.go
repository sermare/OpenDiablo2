package d2app

import (
	"os"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// playRefusal returns why the saved character cannot be started ("" if it can):
// a hardcore character that died is permanently dead.
func (a *App) playRefusal(filePath string) string {
	if filePath == "" {
		return ""
	}

	factory, err := d2hero.NewHeroStateFactory(a.asset)
	if err != nil {
		return ""
	}

	hero := factory.LoadHeroState(filePath)
	if hero == nil {
		return ""
	}

	return hero.PlayRefusal()
}

// markAutoDead is the OD2_AUTOMARKDEAD=1 test aid: it turns the OD2_AUTOGAME hero
// into a dead hardcore character (as a hardcore death leaves the save) so the
// load refusal can be verified without playing a death first.
func (a *App) markAutoDead(save string) {
	if os.Getenv("OD2_AUTOMARKDEAD") == "" {
		return
	}

	factory, err := d2hero.NewHeroStateFactory(a.asset)
	if err != nil {
		return
	}

	hero := factory.LoadHeroState(save)
	if hero == nil {
		a.Errorf("OD2_AUTOMARKDEAD: cannot load %s", save)
		return
	}

	hero.Hardcore = true
	hero.Death = &d2hero.DeathState{Died: true, Deaths: 1}

	err = factory.Save(hero)
	a.Infof("HARDCORE marked %s dead (kind=%v) err=%v", hero.HeroName, hero.Kind(), err)
}
