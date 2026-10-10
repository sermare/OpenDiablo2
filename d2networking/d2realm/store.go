package d2realm

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Store errors.
var (
	ErrNotFound       = errors.New("d2realm: character not found")
	ErrOwnedElsewhere = errors.New("d2realm: character name belongs to another account")
	ErrBadAccount     = errors.New("d2realm: invalid account name")
)

// Store persists characters per account. Implementations must be safe for
// concurrent use.
type Store interface {
	Save(account, char string, data []byte) error
	Load(account, char string) ([]byte, error)
	List(account string) ([]string, error)
}

// ValidAccount reports whether name is usable as an account name: 2 to 15
// letters, digits, '-' or '_' (also keeps it safe as a path element).
func ValidAccount(name string) bool {
	if len(name) < 2 || len(name) > MaxAccountName {
		return false
	}

	for i := 0; i < len(name); i++ {
		c := name[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}

	return true
}

// MemStore keeps characters in memory.
type MemStore struct {
	mu   sync.Mutex
	data map[string]map[string][]byte // lower(account) -> lower(char) -> file
	own  map[string]string            // lower(char) -> lower(account)
}

// NewMemStore returns an empty in-memory store.
func NewMemStore() *MemStore {
	return &MemStore{data: map[string]map[string][]byte{}, own: map[string]string{}}
}

// Save stores a copy of data.
func (m *MemStore) Save(account, char string, data []byte) error {
	a, c := strings.ToLower(account), strings.ToLower(char)
	if !ValidAccount(account) || !ValidAccount(char) {
		return ErrBadAccount
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if o, ok := m.own[c]; ok && o != a {
		return ErrOwnedElsewhere
	}

	if m.data[a] == nil {
		m.data[a] = map[string][]byte{}
	}

	m.data[a][c] = append([]byte(nil), data...)
	m.own[c] = a

	return nil
}

// Load returns a copy of a stored character.
func (m *MemStore) Load(account, char string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	d, ok := m.data[strings.ToLower(account)][strings.ToLower(char)]
	if !ok {
		return nil, ErrNotFound
	}

	return append([]byte(nil), d...), nil
}

// List returns the lower case names of an account's characters, sorted.
func (m *MemStore) List(account string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var out []string
	for c := range m.data[strings.ToLower(account)] {
		out = append(out, c)
	}

	sort.Strings(out)

	return out, nil
}

// DirStore keeps <root>/<account>/<char>.d2s files (names lower case, so the
// store behaves the same on case-insensitive file systems).
type DirStore struct {
	root string
	mu   sync.Mutex
}

// NewDirStore creates the root folder if needed.
func NewDirStore(root string) (*DirStore, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}

	return &DirStore{root: root}, nil
}

func (d *DirStore) path(account, char string) (string, error) {
	if !ValidAccount(account) || !ValidAccount(char) {
		return "", ErrBadAccount
	}

	return filepath.Join(d.root, strings.ToLower(account), strings.ToLower(char)+".d2s"), nil
}

// Save writes the file atomically (temp file + rename) and refuses a
// character name that another account already owns.
func (d *DirStore) Save(account, char string, data []byte) error {
	p, err := d.path(account, char)
	if err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	other, err := filepath.Glob(filepath.Join(d.root, "*", strings.ToLower(char)+".d2s"))
	if err != nil {
		return err
	}

	for _, o := range other {
		if o != p {
			return ErrOwnedElsewhere
		}
	}

	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}

	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, p)
}

// Load reads a stored character.
func (d *DirStore) Load(account, char string) ([]byte, error) {
	p, err := d.path(account, char)
	if err != nil {
		return nil, err
	}

	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}

	return b, err
}

// List returns the lower case names of an account's characters, sorted.
func (d *DirStore) List(account string) ([]string, error) {
	if !ValidAccount(account) {
		return nil, ErrBadAccount
	}

	ents, err := os.ReadDir(filepath.Join(d.root, strings.ToLower(account)))
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	var out []string

	for _, e := range ents {
		if n := e.Name(); strings.HasSuffix(n, ".d2s") {
			out = append(out, strings.TrimSuffix(n, ".d2s"))
		}
	}

	return out, nil
}
