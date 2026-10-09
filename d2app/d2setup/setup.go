package d2setup

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2config"
)

const dialogTitle = "OpenDiablo2"

// ErrNoGameFiles is returned when the user did not provide Diablo II files.
var ErrNoGameFiles = errors.New("no Diablo II game files were selected")

func apply(cfg *d2config.Configuration, v Validation) error {
	cfg.MpqPath = v.Dir
	cfg.MpqLoadOrder = v.LoadOrder

	return cfg.Save()
}

// EnsureGameFiles makes sure cfg points at a folder holding the Diablo II
// archives. A valid configuration is left untouched. Otherwise it offers
// a folder found in the usual places, then asks the user to pick one, and
// writes the result to the config file.
func EnsureGameFiles(cfg *d2config.Configuration, ui UI, home string) error {
	if Complete(cfg.MpqPath, cfg.MpqLoadOrder) {
		return nil
	}

	log.Printf("game files not found at %q, starting first-run setup", cfg.MpqPath)

	if found := FindInstalls(home, cfg.MpqPath); len(found) > 0 {
		v := found[0]
		msg := fmt.Sprintf("OpenDiablo2 found Diablo II files in:\n\n%s\n\nUse this folder?", v.Dir)

		if len(v.MissingExpansion) > 0 {
			msg += "\n\nNote: Lord of Destruction files are missing here (" +
				strings.Join(v.MissingExpansion, ", ") + ")."
		}

		if ui.Ask(dialogTitle, msg, "Choose Another Folder", "Use This Folder") {
			return apply(cfg, v)
		}
	}

	start := home

	for {
		dir, err := ui.ChooseFolder(
			"Select your Diablo II folder (the one containing d2data.mpq, d2char.mpq, patch_d2.mpq ...)", start)
		if err != nil {
			return ErrNoGameFiles
		}

		v := Resolve(dir)
		if v.OK() {
			return apply(cfg, v)
		}

		start = dir

		if !ui.Ask(dialogTitle, fmt.Sprintf(
			"That folder is missing these Diablo II files:\n\n%s\n\nPlease choose the folder that contains them.",
			strings.Join(v.MissingCore, "\n")), "Cancel", "Choose Again") {
			return ErrNoGameFiles
		}
	}
}

// ImportDirs returns the folders to import real characters from: the config
// field when set, else every well-known location that holds .d2s files.
func ImportDirs(cfg *d2config.Configuration, home string) []string {
	if cfg.D2SDir != "" {
		return []string{cfg.D2SDir}
	}

	return FindSaveDirs(home, cfg.MpqPath)
}
