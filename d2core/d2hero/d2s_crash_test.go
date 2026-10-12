package d2hero

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// envCrashHelper makes TestCrashHelperProcess act as a save-writing child process: it saves heroes of the
// given name in a loop, with as many different contents as it likes, until it is killed.
const envCrashHelper = "OD2_TEST_SAVE_HELPER"

// TestCrashHelperProcess is not a test: it is the child of the multi process tests below.
func TestCrashHelperProcess(t *testing.T) {
	path := os.Getenv(envCrashHelper)
	if path == "" {
		t.Skip("child process of the crash tests")
	}

	seed0, _ := strconv.Atoi(os.Getenv("OD2_TEST_SAVE_SEED"))
	count, _ := strconv.Atoi(os.Getenv("OD2_TEST_SAVE_COUNT")) // 0 = until killed

	rotateMinAge = 0

	for i := 0; count == 0 || i < count; i++ {
		// unknown content (known ""): every writer gets through, like two windows that autosave the same hero
		if _, err := CommitD2S(path, hero(t, "Hero", uint32(seed0+i)), ""); err != nil && !errors.Is(err, ErrSaveLocked) {
			fmt.Fprintf(os.Stderr, "child commit: %v\n", err)
			os.Exit(3)
		}
	}
}

func startHelper(t *testing.T, path string, seed, count int) *exec.Cmd {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=^TestCrashHelperProcess$")
	cmd.Env = append(os.Environ(), envCrashHelper+"="+path, "OD2_TEST_SAVE_SEED="+strconv.Itoa(seed),
		"OD2_TEST_SAVE_COUNT="+strconv.Itoa(count))

	var out bytes.Buffer

	cmd.Stdout, cmd.Stderr = &out, &out

	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	return cmd
}

// everySaveFileValid checks the save and each of its backups: whatever exists is a complete, valid .d2s.
func everySaveFileValid(t *testing.T, path string) {
	t.Helper()

	for _, p := range append(BackupCandidates(path), path) {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}

		if err := ValidateD2S(b); err != nil {
			t.Errorf("%s is damaged: %v", filepath.Base(p), err)
		}
	}
}

// TestCommitD2SKill9MidSave kills (SIGKILL) a process that saves in a loop at random moments, many times over.
// Whatever the moment, the save and its backups stay valid, the save lock does not outlive the process, the next
// save works at once and cleans the temp files the dead process left.
func TestCommitD2SKill9MidSave(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("flock based, not for Windows")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "Hero.d2s")

	noRotationDelay(t)

	if _, err := CommitD2S(path, hero(t, "Hero", 1), ""); err != nil {
		t.Fatal(err)
	}

	leftovers := 0

	for round := 0; round < 25; round++ {
		cmd := startHelper(t, path, 1000*round+100, 0)

		time.Sleep(time.Duration(30+(round*37)%120) * time.Millisecond)
		_ = cmd.Process.Kill() // SIGKILL: no deferred unlock, no temp file cleanup
		_ = cmd.Wait()

		everySaveFileValid(t, path)

		for _, n := range listNames(t, dir) {
			if strings.Contains(n, ".tmp") {
				leftovers++
			}
		}

		start := time.Now()
		if _, err := CommitD2S(path, hero(t, "Hero", uint32(round+5)), ""); err != nil {
			t.Fatalf("round %d: the save after the kill failed: %v", round, err)
		}

		if time.Since(start) > 2*time.Second {
			t.Errorf("round %d: the save after the kill waited %v for the lock of a dead process", round, time.Since(start))
		}

		for _, n := range listNames(t, dir) {
			if strings.Contains(n, ".tmp") {
				t.Fatalf("round %d: temp file %s survived a following save", round, n)
			}
		}
	}

	t.Logf("temp files seen after a kill, before the next save cleaned them: %d", leftovers)
}

// TestRemoveStaleTemps: a temp file of a dead writer is removed by the next save; files of other heroes stay.
func TestRemoveStaleTemps(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Hero.d2s")

	stale := []string{".Hero.d2s.tmp123", ".Hero.d2s.bak1.tmp9", ".Hero.d2s.conflict.tmp77"}
	keep := []string{".Other.d2s.tmp5", "Hero.od2", ".Hero.od2.tmp1"}

	for _, n := range append(append([]string{}, stale...), keep...) {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := CommitD2S(path, hero(t, "Hero", 1), ""); err != nil {
		t.Fatal(err)
	}

	for _, n := range stale {
		if _, err := os.Stat(filepath.Join(dir, n)); err == nil {
			t.Errorf("%s was not removed", n)
		}
	}

	for _, n := range keep {
		if _, err := os.Stat(filepath.Join(dir, n)); err != nil {
			t.Errorf("%s was removed: %v", n, err)
		}
	}
}

// TestCommitD2STwoProcessesAtOnce: two processes save the same hero in turns without a break; the file is valid
// at every moment a reader looks, and valid at the end.
func TestCommitD2STwoProcessesAtOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("flock based, not for Windows")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "Hero.d2s")

	if _, err := CommitD2S(path, hero(t, "Hero", 1), ""); err != nil {
		t.Fatal(err)
	}

	a, b := startHelper(t, path, 10000, 60), startHelper(t, path, 20000, 60)

	var wg sync.WaitGroup

	stop := make(chan struct{})
	bad := make(chan string, 1)

	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			select {
			case <-stop:
				return
			default:
			}

			if d, err := os.ReadFile(path); err != nil || ValidateD2S(d) != nil {
				select {
				case bad <- fmt.Sprintf("a reader saw a bad save (err %v)", err):
				default:
				}

				return
			}
		}
	}()

	for _, c := range []*exec.Cmd{a, b} {
		if err := c.Wait(); err != nil {
			t.Errorf("child: %v (%v)", err, c.Stderr)
		}
	}

	close(stop)
	wg.Wait()

	select {
	case msg := <-bad:
		t.Fatal(msg)
	default:
	}

	everySaveFileValid(t, path)

	for _, n := range listNames(t, dir) {
		if strings.Contains(n, ".tmp") || strings.Contains(n, "corrupt") {
			t.Errorf("unexpected file %s", n)
		}
	}
}

// TestCommitD2SReadOnlyFolder: a save folder that cannot be written is an error, never a damaged save, and the
// previous save stays as it was; once the folder can be written again the save goes through.
func TestCommitD2SReadOnlyFolder(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a folder permission that applies to this user")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "Hero.d2s")
	first := hero(t, "Hero", 1)

	if _, err := CommitD2S(path, first, ""); err != nil {
		t.Fatal(err)
	}

	before := listNames(t, dir)

	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	if _, err := CommitD2S(path, hero(t, "Hero", 2), ""); err == nil {
		t.Fatal("a save into a read only folder reported success")
	}

	if got := readFile(t, path); !bytes.Equal(got, first) {
		t.Error("the save changed although the folder is read only")
	}

	if after := listNames(t, dir); strings.Join(after, ",") != strings.Join(before, ",") {
		t.Errorf("the failed save left files: before %v after %v", before, after)
	}

	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	second := hero(t, "Hero", 3)
	if _, err := CommitD2S(path, second, ""); err != nil {
		t.Fatalf("the save after the folder became writable again: %v", err)
	}

	if got := readFile(t, path); !bytes.Equal(got, second) {
		t.Error("the save after the folder became writable did not land")
	}
}

// TestCommitD2SReadOnlyFile: a save file that is read only (copied from a disc, a flag set by the player) is still
// replaced atomically by rename, with the old version kept as a backup: no loss, no error.
func TestCommitD2SReadOnlyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Hero.d2s")
	first := hero(t, "Hero", 1)

	if _, err := CommitD2S(path, first, ""); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}

	second := hero(t, "Hero", 2)
	if _, err := CommitD2S(path, second, ""); err != nil {
		t.Fatalf("saving over a read only file: %v", err)
	}

	if got := readFile(t, path); !bytes.Equal(got, second) {
		t.Error("the new version did not land")
	}

	if got := readFile(t, BackupPath(path, 1)); !bytes.Equal(got, first) {
		t.Error("the old version is not in the backup")
	}
}
