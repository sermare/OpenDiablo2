// Package d2app is the application shell of the macOS fork. App boots the engine
// (configuration, asset manager, MPQ loading, input, audio, screens), owns the main loop and
// the screen transitions (ToMainMenu, ToCreateGame), registers the console commands and runs
// first-run discovery through d2setup. It also holds the environment driven test entry points
// (OD2_AUTOGAME imports a .d2s and starts it directly; the autonewchar, autoperf, autoshot and
// autoclass hooks) that scripts/verify.sh scenarios rely on. It needs a display, so it is not
// unit tested as a whole (autoperf and autoshot have small tests); its behaviour is checked by
// the in-game scenarios in scripts/verify.d. Much of it is upstream OpenDiablo2 code adapted
// for macOS; nothing here is compared against the original game.
package d2app
