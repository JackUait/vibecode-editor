package bash_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
)

// A quit request is a signal, and the trap only raises a flag — so the render
// block it lands in runs on to its final printf, with every $() stage of that
// block killed underneath it. An emptied clamp_scroll slices the body from the
// top while the bar still reports the offset it was built with, and an emptied
// frame paints a blank pane. That garbage is the last thing the pane shows.
func TestCompactView_paints_nothing_once_a_quit_is_requested(t *testing.T) {
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Skip("zsh not available")
	}

	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	writeTempFile(t, dir, "seed.txt", "seed\n")
	git("add", "seed.txt")
	git("commit", "-q", "-m", "seed")
	// Enough rows that one repaint stays in flight for hundreds of ms, so the
	// signal below reliably lands inside the render rather than around it.
	for i := 0; i < 400; i++ {
		writeTempFile(t, dir, fmt.Sprintf("f%03d.txt", i), "x\n")
	}
	git("add", ".")

	module := filepath.Join(projectRoot(t), "lib", "compact-view.sh")
	cmd := exec.Command(zsh, "-c", "source "+module+" && compact_view "+dir)
	env := []string{}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "TMUX=") {
			continue
		}
		env = append(env, e)
	}
	cmd.Env = append(env, "COMPACT_VIEW_INTERVAL=1", "TERM=xterm")

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 12, Cols: 60})
	if err != nil {
		t.Fatalf("start compact_view: %v", err)
	}
	defer func() { _ = ptmx.Close() }()

	var mu sync.Mutex
	var out bytes.Buffer
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				mu.Lock()
				out.Write(buf[:n])
				mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
	length := func() int { mu.Lock(); defer mu.Unlock(); return out.Len() }

	for i := 0; i < 200 && length() == 0; i++ {
		time.Sleep(20 * time.Millisecond)
	}
	if length() == 0 {
		t.Fatal("compact_view painted no first frame")
	}
	time.Sleep(500 * time.Millisecond)

	_, _ = ptmx.Write([]byte("\x1b[<65;12;5M")) // wheel down: starts a repaint
	time.Sleep(150 * time.Millisecond)

	// Marking under the lock keeps the reader from appending between the mark and
	// the Ctrl-C, so everything after the mark is genuinely post-quit output.
	mu.Lock()
	mark := out.Len()
	_, quitErr := ptmx.Write([]byte{0x03}) // Ctrl-C: SIGINT to the whole pane
	mu.Unlock()
	if quitErr != nil {
		t.Fatalf("send Ctrl-C: %v", quitErr)
	}

	quitAt := time.Now()
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	select {
	case <-done:
		t.Logf("exited %v after Ctrl-C", time.Since(quitAt))
	case <-time.After(20 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("compact_view did not exit after a quit request")
	}

	mu.Lock()
	after := out.String()[mark:]
	mu.Unlock()
	if strings.Contains(after, "\x1b[H") {
		frame := after[strings.Index(after, "\x1b[H"):]
		if len(frame) > 400 {
			frame = frame[:400]
		}
		t.Errorf("compact_view painted a frame after the quit request; the render it was "+
			"interrupted in cannot be trusted.\nframe: %q", frame)
	}
}
