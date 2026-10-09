// Package d2setup holds the first-run logic of the macOS app: finding the
// player's Diablo II game files and real character saves, validating them and
// asking the user (through native dialogs) when nothing usable is found.
// The discovery and validation parts are pure and unit tested; the dialogs are
// behind the UI interface.
package d2setup
