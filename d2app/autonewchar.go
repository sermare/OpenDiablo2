package d2app

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// runAutoNewChar implements OD2_AUTONEWCHAR=<class>[,hardcore][,classic][,ladder]:
// it creates a new character through the same path the hero creation screen
// uses (CreateNewHero), logs the parse summary of the .d2s it wrote and, when
// OD2_AUTONEWCHAR_REF names a real new-character file, a field by field diff
// against it. The .d2s goes to OD2_D2S_WRITEBACK (never over an existing file);
// the .od2 the path also writes is removed again so the test leaves no hero
// in the character list. With OD2_AUTOEXIT the process ends afterwards.
// It returns true when the scenario ran.
func (a *App) runAutoNewChar() bool {
	spec := os.Getenv("OD2_AUTONEWCHAR")
	if spec == "" {
		return false
	}

	parts := strings.Split(spec, ",")

	class, ok := parseClassName(parts[0])
	if !ok {
		a.Errorf("NEWCHAR unknown class %q", parts[0])
		return true
	}

	expansion, hardcore, ladder := true, false, false

	for _, flag := range parts[1:] {
		switch strings.ToLower(strings.TrimSpace(flag)) {
		case "hardcore":
			hardcore = true
		case "classic":
			expansion = false
		case "ladder":
			ladder = true
		}
	}

	name := os.Getenv("OD2_AUTONEWCHAR_NAME")
	if name == "" {
		name = "Auto" + class.String()
	}

	factory, err := d2hero.NewHeroStateFactory(a.asset)
	if err != nil {
		a.Errorf("NEWCHAR factory: %v", err)
		return true
	}

	hero, ok := d2hero.HeroOfD2SClass(class)
	if !ok {
		a.Errorf("NEWCHAR class %v has no hero", class)
		return true
	}

	state, res, err := factory.CreateNewHero(name, hero, expansion, hardcore, ladder)
	if err != nil {
		a.Errorf("NEWCHAR create failed: %v", err)
		return true
	}

	defer func() { _ = os.Remove(state.FilePath) }()

	a.Infof("NEWCHAR created name=%s class=%v expansion=%v hardcore=%v ladder=%v d2s=%s", name, class, expansion,
		hardcore, ladder, res.Path)

	for _, w := range res.Warnings {
		a.Warningf("NEWCHAR %s", w)
	}

	if len(state.D2SBase) == 0 {
		return true
	}

	a.logNewCharFile(res.Path, state.D2SBase)
	a.logNewCharReference(state.D2SBase)

	// the first save of the new hero turns the header-only file into a full save
	if saved, err := factory.SaveD2S(state); err != nil {
		a.Warningf("NEWCHAR first save: %v", err)
	} else {
		a.Infof("NEWCHAR first-save reparse: %s", saved.Summary)
	}

	return true
}

func (a *App) logNewCharFile(path string, made []byte) {
	onDisk, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		a.Errorf("NEWCHAR cannot read back %s: %v", path, err)
		return
	}

	a.Infof("NEWCHAR file bytes=%d on_disk_equals_created=%v", len(onDisk), string(onDisk) == string(made))
	a.Infof("NEWCHAR parse: %s", d2hero.SummarizeD2S(onDisk, nil))
}

// logNewCharReference compares the new file with OD2_AUTONEWCHAR_REF, a real
// new character written by the game, and checks the oracle: building the
// reference from its own name, class, flags and creation time must give its
// bytes exactly.
func (a *App) logNewCharReference(made []byte) {
	refPath := os.Getenv("OD2_AUTONEWCHAR_REF")
	if refPath == "" {
		return
	}

	ref, err := os.ReadFile(filepath.Clean(refPath))
	if err != nil {
		a.Warningf("NEWCHAR reference unreadable: %v", err)
		return
	}

	report, unexpected := d2s.DiffReport(made, ref)
	a.Infof("NEWCHAR diff vs %s: unexpected=%d %s", filepath.Base(refPath), unexpected, report)

	h, err := d2s.ParseHeader(ref)
	if err != nil || !h.IsNewCharacter() {
		a.Warningf("NEWCHAR reference is not a new character file: %v", err)
		return
	}

	created := time.Unix(int64(binary.LittleEndian.Uint32(ref[0x2C:])), 0)

	rebuilt, err := d2s.NewCharacter(h.Name, h.Class, d2s.NewCharacterFlags{
		Expansion: h.IsExpansion(), Hardcore: h.IsHardcore(), Ladder: h.IsLadder(), Created: created,
	}, d2s.DefaultAppearance(h.Class))
	if err != nil {
		a.Warningf("NEWCHAR oracle: %v", err)
		return
	}

	rep, n := d2s.DiffReport(rebuilt, ref)
	a.Infof("NEWCHAR oracle byte_identical=%v unexpected=%d %s", string(rebuilt) == string(ref), n, rep)
}

func parseClassName(s string) (d2s.Class, bool) {
	s = strings.ToLower(strings.TrimSpace(s))

	for c := d2s.Amazon; c <= d2s.Assassin; c++ {
		if strings.ToLower(c.String()) == s {
			return c, true
		}
	}

	return 0, false
}
