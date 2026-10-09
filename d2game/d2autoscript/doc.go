// Package d2autoscript implements the OD2_AUTOSCRIPT scenario runner: a small,
// display-free state machine that drives the hero through a list of steps
// (walk, cast, open panels, console commands, log expectations) so a change can
// be tested without a mouse. The game supplies a Host that performs the actions.
//
// Parse turns the semicolon separated script into steps and a Runner executes them against a Host.
// Verified by the package's unit tests (TestParse, TestRunnerPass, TestRunnerFailures,
// TestParseLevelSteps and the automap and skill step tests) with a fake Host; the real Host lives in
// d2game/d2gamescreen and is exercised by the scripts/verify.d scenarios (for example 20-script-walk.sh).
// The script grammar is this fork's own invention, not something the original game has.
package d2autoscript
