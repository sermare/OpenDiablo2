package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2app/d2setup"
)

const (
	watchPidEnv = "OD2_WATCH_PID"
	watchLogEnv = "OD2_WATCH_LOG"
)

// crashMarkers are lines that the Go runtime or main.go write to the log when
// the engine dies unexpectedly.
//
//nolint:gochecknoglobals // constant table
var crashMarkers = []string{"panic:", "fatal error:", "OpenDiablo2 stopped:", "PlatformError", "SIGSEGV", "SIGABRT", "goroutine 1 ["}

// runBundleSupervisor prepares a Finder launch of OpenDiablo2.app.
//
// A double-clicked app starts with "/" as working directory and no terminal.
// The engine process (which must stay the one LaunchServices started, or
// Cocoa cannot reach the window server) therefore:
//   - changes to Contents/Resources,
//   - sends stdout and stderr to ~/Library/Logs/OpenDiablo2/OpenDiablo2.log,
//   - starts a tiny watcher (the same executable in watch mode, no GUI) that
//     waits for the engine to exit and shows a dialog with the log path if
//     the log contains a crash.
//
// Outside an .app bundle nothing happens (handled is false) so command line
// use is unchanged. In watch mode it runs the watcher and reports handled.
func runBundleSupervisor() (handled bool, code int) {
	if pid := os.Getenv(watchPidEnv); pid != "" {
		watch(pid, os.Getenv(watchLogEnv))
		return true, 0
	}

	exe, err := os.Executable()
	if err != nil || !strings.Contains(exe, ".app/Contents/MacOS/") {
		return false, 0
	}

	_ = os.Chdir(filepath.Join(filepath.Dir(exe), "..", "Resources"))

	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, "Library", "Logs", "OpenDiablo2")
	logPath := filepath.Join(logDir, "OpenDiablo2.log")

	_ = os.MkdirAll(logDir, 0o750)
	_ = os.Rename(logPath, filepath.Join(logDir, "OpenDiablo2.previous.log"))

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640) //nolint:gosec // user log
	if err != nil {
		return false, 0
	}

	_ = syscall.Dup2(int(logFile.Fd()), 1)
	_ = syscall.Dup2(int(logFile.Fd()), 2)

	fmt.Fprintf(os.Stderr, "OpenDiablo2 starting (%s)\n", exe)

	w := exec.Command(exe) //nolint:gosec // re-runs ourselves in watch mode
	w.Env = append(os.Environ(), watchPidEnv+"="+strconv.Itoa(os.Getpid()), watchLogEnv+"="+logPath)
	w.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	_ = w.Start()

	return false, 0
}

// watch waits for process pid to exit and then looks for a crash in the log.
func watch(pidText, logPath string) {
	pid, _ := strconv.Atoi(pidText)

	for pid > 0 && syscall.Kill(pid, 0) == nil {
		time.Sleep(time.Second)
	}

	data, _ := os.ReadFile(logPath)
	text := string(data)

	for _, m := range crashMarkers {
		if strings.Contains(text, m) {
			d2setup.AlertWithLog("OpenDiablo2 stopped unexpectedly",
				"OpenDiablo2 hit a problem and had to close. Your characters are saved after each change, "+
					"and your original Diablo II files were not touched.\n\nA log was written to:\n"+logPath,
				logPath)

			return
		}
	}
}
