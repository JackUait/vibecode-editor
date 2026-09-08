package bash_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The pop mutex is a mkdir lock, and _sweep_stale_lock is the only thing that
// may break one it does not hold. It decides by age, and it read the age as
// `stat -f %m "$lock" 2>/dev/null || echo 0` — so a lock it could not stat
// came back as epoch 0, which is ~1.8 billion seconds "old" and always past
// the 10s orphan threshold. The stat fails for one ordinary reason: the lock
// was released between the `-d` test and the stat. By the time the sweep then
// rmdir'd it, another launch had legitimately taken a FRESH lock at that path
// — and the sweep deleted that one, so two pops ran inside the mutex at once,
// read the same queue head, and restored the same project into two tabs.
//
// An unreadable age must mean "nothing to sweep", never "infinitely stale".
func TestSweepStaleLock_keeps_a_lock_whose_age_it_cannot_read(t *testing.T) {
	dir := t.TempDir()
	lock := filepath.Join(dir, "restore-queue.lock")
	if err := os.Mkdir(lock, 0755); err != nil {
		t.Fatalf("mkdir lock: %v", err)
	}

	// A stat that fails is the observable form of "the lock moved under us".
	shim := filepath.Join(dir, "shim")
	if err := os.MkdirAll(shim, 0755); err != nil {
		t.Fatalf("mkdir shim: %v", err)
	}
	if err := os.WriteFile(filepath.Join(shim, "stat"), []byte("#!/bin/bash\nexit 1\n"), 0755); err != nil {
		t.Fatalf("write stat shim: %v", err)
	}

	root := projectRoot(t)
	script := `
export PATH=` + quote(shim) + `:"$PATH"
source ` + quote(filepath.Join(root, "lib", "session-restore.sh")) + `
_sweep_stale_lock ` + quote(lock) + `
if [ -d ` + quote(lock) + ` ]; then echo LOCK-KEPT; else echo LOCK-DESTROYED; fi
`
	out, code := runBashSnippet(t, script, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "LOCK-KEPT")
}

// A genuinely orphaned lock must still be swept, or a wrapper killed between
// its pre-acquire and its pop blocks every later restore forever.
func TestSweepStaleLock_still_sweeps_a_genuinely_orphaned_lock(t *testing.T) {
	dir := t.TempDir()
	lock := filepath.Join(dir, "restore-queue.lock")
	if err := os.Mkdir(lock, 0755); err != nil {
		t.Fatalf("mkdir lock: %v", err)
	}
	// A builder that died after its pre-acquire leaves the owner stamp behind.
	if err := os.WriteFile(filepath.Join(lock, "owner"), []byte("4242\n"), 0644); err != nil {
		t.Fatalf("write owner: %v", err)
	}

	root := projectRoot(t)
	script := `
touch -t 202001010101 ` + quote(lock) + `
source ` + quote(filepath.Join(root, "lib", "session-restore.sh")) + `
_sweep_stale_lock ` + quote(lock) + `
if [ -d ` + quote(lock) + ` ]; then echo LOCK-KEPT; else echo LOCK-SWEPT; fi
`
	out, code := runBashSnippet(t, script, nil)
	assertExitCode(t, code, 0)
	assertContains(t, out, "LOCK-SWEPT")
}

// The invariant the storm guard actually rests on: whatever the interleaving,
// an entry is consumed by exactly one pop. Many poppers race a short queue;
// every line that comes back must be distinct. A duplicate here is a project
// restored into two tabs.
func TestRestoreQueuePop_never_hands_one_entry_to_two_poppers(t *testing.T) {
	dir := t.TempDir()
	boot := "boot-1"

	const entries = 6
	var queue strings.Builder
	for i := 0; i < entries; i++ {
		queue.WriteString(boot + "|/p/proj-" + string(rune('a'+i)) + "|opencode||||\n")
	}
	writeTempFile(t, dir, "restore-queue", queue.String())

	root := projectRoot(t)
	// More poppers than entries, all released together, so the losers exercise
	// the drained/empty paths while the winners race on the mutex.
	script := `
source ` + quote(filepath.Join(root, "lib", "session-restore.sh")) + `
out=` + quote(filepath.Join(dir, "pops")) + `
: > "$out"
pids=()
for i in $(seq 1 12); do
  (
    e="$(restore_queue_pop ` + quote(dir) + ` ` + quote(boot) + `)"
    [ -n "$e" ] && printf '%s\n' "$e" >> "$out"
    exit 0
  ) &
  pids+=($!)
done
for p in "${pids[@]}"; do wait "$p"; done
`
	_, code := runBashSnippet(t, script, nil)
	assertExitCode(t, code, 0)

	data, err := os.ReadFile(filepath.Join(dir, "pops"))
	if err != nil {
		t.Fatalf("read pops: %v", err)
	}
	seen := map[string]int{}
	total := 0
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		seen[line]++
		total++
	}
	for entry, n := range seen {
		if n > 1 {
			t.Errorf("entry %q was popped %d times; each entry must be consumed exactly once\nall pops:\n%s", entry, n, data)
		}
	}
	if total > entries {
		t.Errorf("popped %d entries from a queue of %d", total, entries)
	}
}
