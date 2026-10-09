// Command OpenDiablo2 is the entry point of the macOS fork: main.go wires up
// d2app (boot, first-run discovery through d2setup, exit-code reporting) and
// bundle_darwin.go / bundle_other.go hold the macOS .app bundle supervisor (
// when launched from OpenDiablo2.app it changes to Contents/Resources, redirects the log to
// ~/Library/Logs/OpenDiablo2 and starts a crash watcher driven by OD2_WATCH_PID and OD2_WATCH_LOG).
// GitBranch and GitCommit are filled in at build time. There is nothing to verify
// against the original game here; the whole application is exercised by the
// scenarios in scripts/verify.d, which need a GUI session and a Diablo II install.
// See docs/ARCHITECTURE.md for the package map.
package main
