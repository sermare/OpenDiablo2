package d2config

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	od2ConfigDirName  = "OpenDiablo2"
	od2ConfigFileName = "config.json"
)

// ConfigDirEnv names an environment variable that overrides the directory that
// holds config.json and the Saves folder. It exists for tests; normal use keeps
// the per-user default (~/Library/Application Support/OpenDiablo2 on macOS).
const ConfigDirEnv = "OD2_CONFIG_DIR"

// ConfigDir returns the directory holding config.json and Saves, or "" when the
// user config directory is unknown.
func ConfigDir() string {
	if dir := os.Getenv(ConfigDirEnv); dir != "" {
		return dir
	}

	if configDir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(configDir, od2ConfigDirName)
	}

	if runtime.GOOS == "windows" {
		if base := windowsConfigBase(os.Getenv); base != "" {
			return base + `\` + od2ConfigDirName
		}
	}

	return ""
}

// DefaultConfigPath returns the absolute path for the default config file location
func DefaultConfigPath() string {
	if dir := ConfigDir(); dir != "" {
		return filepath.Join(dir, od2ConfigFileName)
	}

	return LocalConfigPath()
}

// LocalConfigPath returns the absolute path to the directory of the OpenDiablo2 executable
func LocalConfigPath() string {
	return filepath.Join(filepath.Dir(os.Args[0]), od2ConfigFileName)
}

// windowsConfigBase gives %APPDATA%, else %USERPROFILE%\AppData\Roaming, else "". It is the fallback for when
// os.UserConfigDir fails on Windows and is pure so that tests can run it anywhere.
func windowsConfigBase(getenv func(string) string) string {
	if v := getenv("APPDATA"); v != "" {
		return strings.TrimRight(v, `\/`)
	}

	if v := getenv("USERPROFILE"); v != "" {
		return strings.TrimRight(v, `\/`) + `\AppData\Roaming`
	}

	return ""
}

// DecodeConfig reads config JSON into cfg. Windows editors (Notepad, PowerShell's Set-Content) often save a UTF-8 byte
// order mark, which encoding/json rejects; it is skipped here. CRLF line endings are plain JSON whitespace.
func DecodeConfig(r io.Reader, cfg *Configuration) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	return json.Unmarshal(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")), cfg)
}
