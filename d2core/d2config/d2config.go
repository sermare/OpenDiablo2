package d2config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Configuration defines the configuration for the engine, loaded from config.json
type Configuration struct {
	MpqLoadOrder    []string
	MpqPath         string
	TicksPerSecond  int
	FpsCap          int
	SfxVolume       float64
	BgmVolume       float64
	FullScreen      bool
	RunInBackground bool
	VsyncEnabled    bool
	Backend         string
	// WindowScale multiplies the 800x600 start window size (0 or 1 = normal).
	WindowScale int
	// D2SDir is an optional folder of real Diablo II .d2s characters to import
	// (read only). When empty, well-known locations are searched.
	D2SDir string
	// Language selects the game language by tag (enUS, deDE, esES, frFR, itIT, jaJP, koKR,
	// plPL, ptBR, ruRU, zhCN, zhTW). Empty or "auto" uses the language of the install.
	// The -lang flag and the OD2_LANGUAGE environment variable override it.
	Language string `json:",omitempty"`
	// Options holds the choices of the in-game options menu (see options.go);
	// sound and music live in SfxVolume and BgmVolume.
	Options map[string]int `json:",omitempty"`
	path    string
}

// Save saves the configuration object to disk
func (c *Configuration) Save() error {
	configDir := filepath.Dir(c.path)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return err
	}

	configFile, err := os.Create(c.path)
	if err != nil {
		return err
	}

	buf, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	if _, err := configFile.Write(buf); err != nil {
		return err
	}

	return configFile.Close()
}

// Dir returns the directory component of the path
func (c *Configuration) Dir() string {
	return filepath.Dir(c.path)
}

// Base returns the base component of the path
func (c *Configuration) Base() string {
	return filepath.Base(c.path)
}

// Path returns the config file path
func (c *Configuration) Path() string {
	return c.path
}

// SetPath sets where the config file is saved to (a full path)
func (c *Configuration) SetPath(p string) {
	c.path = p
}
