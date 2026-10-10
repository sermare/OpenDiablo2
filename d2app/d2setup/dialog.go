package d2setup

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ErrCancelled is returned when the user dismisses a dialog.
var ErrCancelled = errors.New("cancelled by user")

// UI is what the setup flow needs from the desktop.
type UI interface {
	// ChooseFolder shows a folder picker and returns the chosen path.
	ChooseFolder(prompt, startDir string) (string, error)
	// Ask shows message with two buttons and reports whether yes was clicked.
	Ask(title, message, no, yes string) bool
	// Alert shows an error message with an OK button.
	Alert(title, message string)
}

// asQuote escapes s as an AppleScript string literal.
func asQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)

	return `"` + s + `"`
}

// OsaUI implements UI with osascript, so it works from a Finder launch.
type OsaUI struct{}

func osascript(script string) (string, error) {
	out, err := exec.Command("/usr/bin/osascript", "-e", script).Output()

	return strings.TrimSpace(string(out)), err
}

// ChooseFolder implements UI.
func (OsaUI) ChooseFolder(prompt, startDir string) (string, error) {
	script := "POSIX path of (choose folder with prompt " + asQuote(prompt)

	if st, err := os.Stat(startDir); err == nil && st.IsDir() {
		script += " default location (POSIX file " + asQuote(startDir) + ")"
	}

	out, err := osascript(script + ")")
	if err != nil || out == "" {
		return "", ErrCancelled
	}

	return strings.TrimRight(out, "/"), nil
}

// Ask implements UI.
func (OsaUI) Ask(title, message, no, yes string) bool {
	script := fmt.Sprintf("button returned of (display dialog %s with title %s buttons {%s, %s} default button %s)",
		asQuote(message), asQuote(title), asQuote(no), asQuote(yes), asQuote(yes))
	out, err := osascript(script)

	return err == nil && out == yes
}

// Alert implements UI.
func (OsaUI) Alert(title, message string) {
	_, _ = osascript(fmt.Sprintf(`display dialog %s with title %s with icon stop buttons {"OK"} default button "OK"`,
		asQuote(message), asQuote(title)))
}

// AlertWithLog shows an error and offers to reveal logPath in Finder.
func AlertWithLog(title, message, logPath string) {
	script := fmt.Sprintf(`button returned of (display dialog %s with title %s with icon stop buttons {"Show Log", "OK"} default button "OK")`,
		asQuote(message), asQuote(title))
	if out, _ := osascript(script); out == "Show Log" {
		_ = exec.Command("/usr/bin/open", "-R", logPath).Run()
	}
}

// ScriptedUI is a UI for tests: ChooseFolder returns Pick, Ask answers yes.
// It is selected by the OD2_SETUP_AUTOPICK environment variable.
type ScriptedUI struct {
	Pick string
	used bool // the pick is offered once; asking again cancels (no endless loop on a bad folder)
}

// ChooseFolder implements UI.
func (s *ScriptedUI) ChooseFolder(string, string) (string, error) {
	if s.Pick == "" || s.used {
		return "", ErrCancelled
	}

	s.used = true

	return s.Pick, nil
}

// Ask implements UI.
func (*ScriptedUI) Ask(string, string, string, string) bool { return true }

// Alert implements UI.
func (*ScriptedUI) Alert(title, message string) {
	fmt.Fprintf(os.Stderr, "ALERT %s: %s\n", title, message)
}

// DefaultUI returns the scripted UI when OD2_SETUP_AUTOPICK is set (tests
// only) and the native osascript dialogs otherwise.
func DefaultUI() UI {
	if p := os.Getenv("OD2_SETUP_AUTOPICK"); p != "" {
		return &ScriptedUI{Pick: p}
	}

	return OsaUI{}
}
