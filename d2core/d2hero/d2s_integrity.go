package d2hero

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// Save integrity: the owner must never lose a character. Every write of a .d2s
// goes through CommitD2S (atomic, validated, backed up, last-writer checked,
// under a lock); every read of a .d2s from disk goes through the loader in this
// file (validated, with an offer to recover from a backup).

// BackupCount is the number of rotating backups kept next to a save:
// "Name.d2s.bak1" is the newest, "Name.d2s.bak3" the oldest. "Name.d2s.bak"
// (BackupSuffix) stays the untouched first original.
const BackupCount = 3

// ConflictSuffix is appended to a .d2s name for the copy that was not written
// because another process changed the file in the meantime.
const ConflictSuffix = ".conflict"

// CorruptSuffix is appended (with a time stamp) to the copy of a damaged save
// that is kept when a backup replaces it.
const CorruptSuffix = ".corrupt-"

// Errors of the save layer.
var (
	// ErrIncompleteLoad: the hero did not load completely, so a save would drop data.
	ErrIncompleteLoad = errors.New("d2s: the hero was not loaded completely, not saved")
	// ErrConcurrentWrite: the file was changed by someone else since this process read or wrote it.
	ErrConcurrentWrite = errors.New("d2s: the file changed on disk since it was loaded (another game window?), not overwritten")
	// ErrSaveLocked: another process holds the save lock for too long.
	ErrSaveLocked = errors.New("d2s: the save is locked by another process")
	// ErrCorruptNoBackup: the file is damaged and no valid backup exists.
	ErrCorruptNoBackup = errors.New("d2s: the save is damaged and has no valid backup")
)

// rotateMinAge is how old the newest rotating backup must be before the older
// ones are shifted, so that a burst of saves (level changes) does not push all
// good versions out. Tests set it to 0.
//
//nolint:gochecknoglobals // tuning knob of the package, changed only by tests
var rotateMinAge = 2 * time.Minute

// ValidateD2S says whether data is a complete .d2s: magic, size field,
// checksum, version and class are those the game accepts (the same checks as
// d2s.ParseHeader, in the order of the original). Nothing is repaired.
func ValidateD2S(data []byte) error {
	_, err := d2s.ParseHeader(data)

	return err
}

// SumD2S is the identity of a file's content used by the last-writer check.
func SumD2S(data []byte) string {
	s := sha256.Sum256(data)

	return hex.EncodeToString(s[:])
}

// BackupPath is the path of rotating backup n (1 = newest).
func BackupPath(path string, n int) string { return path + BackupSuffix + strconv.Itoa(n) }

// BackupCandidates lists the backups of a save, newest first: bak1..bak3 and
// the first original (.bak).
func BackupCandidates(path string) []string {
	out := make([]string, 0, BackupCount+1)
	for n := 1; n <= BackupCount; n++ {
		out = append(out, BackupPath(path, n))
	}

	return append(out, path+BackupSuffix)
}

// FindValidBackup returns the newest backup of path that is a valid .d2s, and
// its content; "" when there is none.
func FindValidBackup(path string) (string, []byte) {
	for _, c := range BackupCandidates(path) {
		data, err := os.ReadFile(filepath.Clean(c))
		if err == nil && ValidateD2S(data) == nil {
			return c, data
		}
	}

	return "", nil
}

// rotateBackups makes room for cur (a valid save about to be replaced) as the
// newest backup. The older ones shift by one when the newest is old enough.
func rotateBackups(path string, cur []byte) error {
	b1 := BackupPath(path, 1)

	if st, err := os.Stat(b1); err == nil && time.Since(st.ModTime()) >= rotateMinAge {
		for n := BackupCount; n >= 2; n-- {
			if _, err := os.Stat(BackupPath(path, n-1)); err == nil {
				if err := os.Rename(BackupPath(path, n-1), BackupPath(path, n)); err != nil {
					return err
				}
			}
		}
	}

	return writeFileAtomic(b1, cur)
}

// CommitD2S is the one writer of a .d2s. It refuses data that is not a valid
// save, takes the save lock, refuses to overwrite a file that changed since
// the caller last saw it (known = SumD2S of that content, "" when unknown; the
// new content is then kept as path+ConflictSuffix), keeps the previous version
// as rotating backups (never rotating a damaged file over good backups: it is
// kept as .corrupt-<time> instead) and replaces the file atomically.
// It returns the SumD2S of what is on disk afterwards.
func CommitD2S(path string, data []byte, known string) (string, error) {
	if err := ValidateD2S(data); err != nil {
		return "", fmt.Errorf("refusing to write an invalid save: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), mkdirPermission); err != nil {
		return "", err
	}

	unlock, err := lockD2S(path)
	if err != nil {
		return "", err
	}
	defer unlock()

	removeStaleTemps(path)

	cur, err := os.ReadFile(filepath.Clean(path))
	exists := err == nil

	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	if exists {
		if known != "" && SumD2S(cur) != known {
			if werr := writeFileAtomic(path+ConflictSuffix, data); werr != nil {
				return "", fmt.Errorf("%w (and the side copy failed: %v)", ErrConcurrentWrite, werr)
			}

			return SumD2S(cur), fmt.Errorf("%w; the new version is in %s", ErrConcurrentWrite, filepath.Base(path+ConflictSuffix))
		}

		if bytes.Equal(cur, data) {
			return SumD2S(cur), nil
		}

		if ValidateD2S(cur) == nil {
			if err := backupOriginal(path); err != nil {
				return "", fmt.Errorf("backup of %s: %w", path, err)
			}

			if err := rotateBackups(path, cur); err != nil {
				return "", fmt.Errorf("rotating the backups of %s: %w", path, err)
			}
		} else if len(cur) > 0 {
			keep := fmt.Sprintf("%s%s%d", path, CorruptSuffix, time.Now().UnixNano())
			if err := writeFileAtomic(keep, cur); err != nil {
				return "", fmt.Errorf("keeping the damaged %s: %w", path, err)
			}
		}
	}

	if err := writeFileAtomic(path, data); err != nil {
		return "", err
	}

	return SumD2S(data), nil
}

// removeStaleTemps deletes the temp files that a writer of this save left behind when it was killed between creating
// the temp file and renaming it (kill -9, power loss). Every writer of the save holds the save lock, and the caller
// holds it now, so no temp file of this save belongs to a live writer.
func removeStaleTemps(path string) {
	dir, prefix := filepath.Dir(filepath.Clean(path)), "."+filepath.Base(path)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, e := range entries {
		if n := e.Name(); strings.HasPrefix(n, prefix) && strings.Contains(n, ".tmp") {
			_ = os.Remove(filepath.Join(dir, n))
		}
	}
}

// NoteD2SOnDisk records that path holds data, for the last-writer check of the next save.
func (s *HeroState) NoteD2SOnDisk(path string, data []byte) {
	s.d2sDiskPath, s.d2sDiskSum = path, SumD2S(data)
}

// knownSum is the content sum the next write to path expects to find, or "".
func (s *HeroState) knownSum(path string) string {
	if s.d2sDiskPath == path {
		return s.d2sDiskSum
	}

	return ""
}

// AskUser asks the player a yes/no question about a save (a recovery, a
// repair). It is a variable so that tests and the automation can answer.
// The default is OD2_D2S_PROMPT=yes|no when set; "no" while a scripted run is
// in progress (OD2_AUTOGAME, OD2_AUTOFLOW, OD2_AUTOMENU); the native macOS
// dialog otherwise (a dialog that is not answered in two minutes is "no").
//
//nolint:gochecknoglobals // replaceable prompt hook
var AskUser = defaultAskUser

func defaultAskUser(title, message, no, yes string) bool {
	switch strings.ToLower(os.Getenv("OD2_D2S_PROMPT")) {
	case "yes", "y", "1":
		return true
	case "no", "n", "0":
		return false
	}

	for _, v := range []string{"OD2_AUTOGAME", "OD2_AUTOFLOW", "OD2_AUTOMENU"} {
		if os.Getenv(v) != "" {
			return false
		}
	}

	if runtime.GOOS != "darwin" {
		return false
	}

	quote := func(s string) string {
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `"`, `\"`)

		return `"` + s + `"`
	}

	script := fmt.Sprintf("button returned of (display dialog %s with title %s buttons {%s, %s} default button %s giving up after 120)",
		quote(message), quote(title), quote(no), quote(yes), quote(no))
	out, err := exec.Command("/usr/bin/osascript", "-e", script).Output()

	return err == nil && strings.TrimSpace(string(out)) == yes
}

// ReadD2SFile reads a .d2s for import and makes sure it is a valid save. A
// damaged file is never deleted, reset or overwritten here: when a valid backup
// exists the player is asked whether to restore it (the damaged file is kept as
// .corrupt-<time>); otherwise the error is returned and the file stays as it is.
func (f *HeroStateFactory) ReadD2SFile(path string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	if verr := ValidateD2S(data); verr != nil {
		return f.recoverD2S(path, data, verr)
	}

	return f.offerPlaceholderRepair(path, data), nil
}

func (f *HeroStateFactory) recoverD2S(path string, bad []byte, verr error) ([]byte, error) {
	name := filepath.Base(path)
	bak, good := FindValidBackup(path)

	if bak == "" {
		fmt.Printf("SAVE DAMAGED %s: %v; no valid backup; the file is left untouched\n", name, verr)

		return nil, fmt.Errorf("%w: %s: %v", ErrCorruptNoBackup, name, verr)
	}

	msg := fmt.Sprintf("The save of %s is damaged (%v).\n\nRestore the newest valid backup (%s)? "+
		"The damaged file is kept next to it.", strings.TrimSuffix(name, filepath.Ext(name)), verr, filepath.Base(bak))
	fmt.Printf("SAVE DAMAGED %s: %v; valid backup %s\n", name, verr, filepath.Base(bak))

	if !AskUser("Damaged save", msg, "Leave it", "Restore") {
		return nil, fmt.Errorf("d2s: %s is damaged (%v); backup %s was not restored", name, verr, filepath.Base(bak))
	}

	keep := fmt.Sprintf("%s%s%d", path, CorruptSuffix, time.Now().UnixNano())
	if err := writeFileAtomic(keep, bad); err != nil {
		return nil, fmt.Errorf("keeping the damaged %s: %w", name, err)
	}

	unlock, err := lockD2S(path)
	if err != nil {
		return nil, err
	}
	defer unlock()

	if err := writeFileAtomic(path, good); err != nil {
		return nil, err
	}

	fmt.Printf("SAVE RESTORED %s from %s (damaged copy: %s)\n", name, filepath.Base(bak), filepath.Base(keep))

	return good, nil
}

// placeholderCodes are the base items the engine used to write into the first
// save of every new hero (hard-coded test gear of the inventory screen, issue
// 795 leftovers; fixed by the starter-kit change). A real new hero never owns
// them.
//
//nolint:gochecknoglobals // constant set
var placeholderCodes = map[string]bool{
	"crn": true, "plt": true, "vbt": true, "vbl": true, "lgl": true, "rin": true, "amu": true, "kit": true,
}

// NoRepairSuffix marks a hero whose player declined the placeholder repair.
const NoRepairSuffix = ".norepair"

// PlaceholderItems lists the items of a save that RepairPlaceholders would
// remove: only for a hero of level 1 with 0 experience, and only plain
// (normal quality, unsocketed, nothing socketed into them) items with a
// placeholder code. Anything else is the player's and is never touched.
func PlaceholderItems(c *d2s.Character) []int {
	if c == nil || c.Body == nil || c.Body.Attributes.Level != 1 || c.Body.Attributes.Experience != 0 {
		return nil
	}

	var idx []int

	for i := range c.Items {
		it := &c.Items[i]
		if it.Ear || it.Location == d2s.LocationSocketed || !placeholderCodes[trimCode(it.Code)] {
			continue
		}

		if it.Simple || it.Quality != d2s.QualityNormal || it.Socketed || len(it.Children) > 0 {
			continue
		}

		idx = append(idx, i)
	}

	return idx
}

// RepairPlaceholders removes the placeholder items (see PlaceholderItems) from
// a save and returns the new bytes and the codes that were removed. It returns
// nil when there is nothing to repair. It does not write anything.
func RepairPlaceholders(data []byte, tables *d2s.ItemTables) ([]byte, []string, error) {
	c, err := d2s.Parse(data, tables)
	if err != nil {
		return nil, nil, err
	}

	idx := PlaceholderItems(c)
	if len(idx) == 0 {
		return nil, nil, nil
	}

	drop := map[int]bool{}

	var codes []string

	for _, i := range idx {
		drop[i] = true

		codes = append(codes, trimCode(c.Items[i].Code))
	}

	kept := make([]d2s.Item, 0, len(c.Items))

	for i := range c.Items {
		if !drop[i] {
			kept = append(kept, c.Items[i])
		}
	}

	c.Items = kept

	out, err := d2s.Write(c, tables)
	if err != nil {
		return nil, nil, err
	}

	if _, err := d2s.Parse(out, tables); err != nil {
		return nil, nil, fmt.Errorf("repaired file does not parse: %w", err)
	}

	return out, codes, nil
}

// offerPlaceholderRepair asks whether the placeholder items of a hero made
// before the starter-kit fix should be removed. Never automatic: nothing
// changes unless the player agrees; a "no" is remembered (Name.d2s.norepair)
// so the question is not asked at every start. The repair writes a backup
// first (CommitD2S). On any problem the original bytes are returned.
func (f *HeroStateFactory) offerPlaceholderRepair(path string, data []byte) []byte {
	if _, err := os.Stat(path + NoRepairSuffix); err == nil {
		return data
	}

	tables := f.d2sTables

	if tables == nil {
		if f.asset == nil {
			return data
		}

		var err error
		if tables, err = f.loadD2SItemTables(); err != nil {
			return data
		}
	}

	fixed, codes, err := RepairPlaceholders(data, tables)
	if err != nil || fixed == nil {
		return data
	}

	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	fmt.Printf("SAVE REPAIR offered for %s: placeholder items %v\n", name, codes)

	msg := fmt.Sprintf("%s is a new hero (level 1, no experience) that still holds test items written by an old version: %s.\n\n"+
		"Remove them? A backup of the save is made first.", name, strings.Join(codes, ", "))

	if !AskUser("Old placeholder items", msg, "Keep them", "Remove") {
		_ = writeFileAtomic(path+NoRepairSuffix, []byte("declined\n"))

		return data
	}

	if _, err := CommitD2S(path, fixed, SumD2S(data)); err != nil {
		fmt.Printf("SAVE REPAIR of %s failed, nothing changed: %v\n", name, err)

		return data
	}

	fmt.Printf("SAVE REPAIR done for %s: removed %v (backup %s)\n", name, codes, filepath.Base(BackupPath(path, 1)))

	return fixed
}
