package main

import (
	"errors"
	"log"
	"os"

	"github.com/OpenDiablo2/OpenDiablo2/d2app"
	"github.com/OpenDiablo2/OpenDiablo2/d2app/d2setup"
)

// GitBranch is set by the CI build process to the name of the branch
//nolint:gochecknoglobals // This is filled in by the build system
var GitBranch = "local"

// GitCommit is set by the CI build process to the commit hash
//nolint:gochecknoglobals // This is filled in by the build system
var GitCommit = "build"

// exitReported is the exit code of a run that already told the user what went
// wrong in a dialog, so the app supervisor does not show a second one.
const exitReported = 3

func main() {
	// inside OpenDiablo2.app: supervise a child run, with logs and crash dialog
	if handled, code := runBundleSupervisor(); handled {
		os.Exit(code)
	}

	log.SetFlags(log.Lshortfile)

	instance := d2app.Create(GitBranch, GitCommit)

	if err := instance.Run(); err != nil {
		if errors.Is(err, d2setup.ErrReported) {
			log.Printf("OpenDiablo2 exited: %v", err)
			os.Exit(exitReported)
		}

		log.Printf("OpenDiablo2 stopped: %v", err)

		os.Exit(1)
	}
}
