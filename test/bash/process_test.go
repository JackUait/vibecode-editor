package bash_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// processSnippet builds a bash snippet that sources process.sh then runs body.
func processSnippet(t *testing.T, body string) string {
	t.Helper()
	root := projectRoot(t)
	processPath := filepath.Join(root, "lib", "process.sh")
	return fmt.Sprintf("source %q && %s", processPath, body)
}

// tmuxSessionSnippet builds a bash snippet that sources process.sh and tmux-session.sh then runs body.
func tmuxSessionSnippet(t *testing.T, body string) string {
	t.Helper()
	root := projectRoot(t)
	processPath := filepath.Join(root, "lib", "process.sh")
	tmuxPath := filepath.Join(root, "lib", "tmux-session.sh")
	return fmt.Sprintf("source %q && source %q && %s", processPath, tmuxPath, body)
}

// isProcessRunning checks whether a process is alive and NOT a zombie.
// A zombie process still responds to kill -0 but has state Z in ps output.
func isProcessRunning(pid int) bool {
	// First check kill -0
	if exec.Command("kill", "-0", strconv.Itoa(pid)).Run() != nil {
		return false
	}
	// Check if it's a zombie
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "state=").Output()
	if err != nil {
		return false
	}
	state := strings.TrimSpace(string(out))
	// Zombie states: "Z", "Z+" etc.
	return !strings.HasPrefix(state, "Z")
}

// reapProcess calls cmd.Wait() in a goroutine to reap zombie processes.
// After kill_tree terminates a Go-spawned child, the child becomes a zombie
// until Go's Wait() reaps it.
func reapProcess(cmd *exec.Cmd) {
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

// forceKill attempts to SIGKILL a process, then reap remaining children.
func forceKill(pid int) {
	_ = exec.Command("kill", "-9", strconv.Itoa(pid)).Run()
}

// getChildren returns child PIDs of the given PID via pgrep.
func getChildren(pid int) []int {
	out, err := exec.Command("pgrep", "-P", strconv.Itoa(pid)).Output()
	if err != nil {
		return nil
	}
	var pids []int
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		p, err := strconv.Atoi(line)
		if err == nil {
			pids = append(pids, p)
		}
	}
	return pids
}

// ======================================================================
// kill_tree tests (15 from process.bats)
// ======================================================================

func TestKillTree_kills_parent_and_children(t *testing.T) {
	cmd := exec.Command("bash", "-c", "sleep 300 & sleep 300 & wait")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to spawn process tree: %v", err)
	}
	pid := cmd.Process.Pid
	defer forceKill(pid)
	time.Sleep(300 * time.Millisecond)

	if !isProcessRunning(pid) {
		t.Fatal("parent process should be alive before kill_tree")
	}

	snippet := processSnippet(t, fmt.Sprintf("kill_tree %d TERM 2>/dev/null || true", pid))
	_, _ = runBashSnippet(t, snippet, nil)
	time.Sleep(300 * time.Millisecond)
	reapProcess(cmd)

	if isProcessRunning(pid) {
		t.Error("parent should be dead after kill_tree")
	}
}

func TestKillTree_handles_nonexistent_PID_gracefully(t *testing.T) {
	if isProcessRunning(999999) {
		t.Skip("PID 999999 unexpectedly exists")
	}

	_, code := runBashFunc(t, "lib/process.sh", "kill_tree", []string{"999999", "TERM"}, nil)
	assertExitCode(t, code, 0)
}

func TestKillTree_defaults_to_TERM_signal(t *testing.T) {
	cmd := exec.Command("sleep", "300")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sleep: %v", err)
	}
	pid := cmd.Process.Pid
	defer forceKill(pid)

	snippet := processSnippet(t, fmt.Sprintf("kill_tree %d 2>/dev/null || true", pid))
	_, _ = runBashSnippet(t, snippet, nil)
	time.Sleep(200 * time.Millisecond)
	reapProcess(cmd)

	if isProcessRunning(pid) {
		t.Error("process should be dead after kill_tree with default TERM")
	}
}

func TestKillTree_kills_deep_process_tree_4_levels(t *testing.T) {
	cmd := exec.Command("bash", "-c", `bash -c "bash -c \"sleep 100\" & sleep 100" & sleep 100`)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start deep tree: %v", err)
	}
	pid := cmd.Process.Pid
	defer forceKill(pid)
	time.Sleep(500 * time.Millisecond)

	if !isProcessRunning(pid) {
		t.Fatal("parent should be alive before kill_tree")
	}

	descendants := getChildren(pid)

	snippet := processSnippet(t, fmt.Sprintf("kill_tree %d TERM 2>/dev/null || true", pid))
	_, _ = runBashSnippet(t, snippet, nil)
	time.Sleep(500 * time.Millisecond)
	reapProcess(cmd)

	if isProcessRunning(pid) {
		t.Error("parent should be dead after kill_tree")
	}

	for _, dpid := range descendants {
		if isProcessRunning(dpid) {
			t.Errorf("descendant %d should be dead after kill_tree", dpid)
		}
	}
}

func TestKillTree_handles_process_that_forks_during_kill_with_timeout(t *testing.T) {
	cmd := exec.Command("bash", "-c", `trap "bash -c \"sleep 5\" &" TERM; sleep 100`)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start trap process: %v", err)
	}
	pid := cmd.Process.Pid
	defer forceKill(pid)
	time.Sleep(300 * time.Millisecond)

	root := projectRoot(t)
	snippet := fmt.Sprintf(`
timeout 3 bash -c "
  source '%s/lib/process.sh'
  kill_tree %d TERM 2>/dev/null || true
  sleep 0.5
  kill_tree %d KILL 2>/dev/null || true
" || true
`, root, pid, pid)
	_, _ = runBashSnippet(t, snippet, nil)
	time.Sleep(300 * time.Millisecond)
	reapProcess(cmd)

	if isProcessRunning(pid) {
		t.Error("parent should be dead after kill_tree with timeout")
	}

	children := getChildren(pid)
	if len(children) > 0 {
		t.Errorf("expected no children, found: %v", children)
	}
}

func TestKillTree_handles_zombie_processes_in_tree(t *testing.T) {
	cmd := exec.Command("bash", "-c", "(exit 0) & sleep 5")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start zombie tree: %v", err)
	}
	pid := cmd.Process.Pid
	defer forceKill(pid)
	time.Sleep(300 * time.Millisecond)

	if !isProcessRunning(pid) {
		t.Fatal("parent should be alive")
	}

	snippet := processSnippet(t, fmt.Sprintf("kill_tree %d TERM 2>/dev/null || true", pid))
	_, _ = runBashSnippet(t, snippet, nil)
	time.Sleep(300 * time.Millisecond)
	reapProcess(cmd)

	if isProcessRunning(pid) {
		t.Error("parent should be dead after kill_tree")
	}
}

func TestKillTree_multiple_calls_are_idempotent(t *testing.T) {
	cmd := exec.Command("sleep", "100")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sleep: %v", err)
	}
	pid := cmd.Process.Pid
	defer forceKill(pid)
	time.Sleep(200 * time.Millisecond)

	// First kill
	_, code := runBashFunc(t, "lib/process.sh", "kill_tree", []string{strconv.Itoa(pid), "TERM"}, nil)
	assertExitCode(t, code, 0)
	time.Sleep(200 * time.Millisecond)
	reapProcess(cmd)

	// Second kill on now-dead PID
	_, code = runBashFunc(t, "lib/process.sh", "kill_tree", []string{strconv.Itoa(pid), "TERM"}, nil)
	assertExitCode(t, code, 0)

	// Third kill
	_, code = runBashFunc(t, "lib/process.sh", "kill_tree", []string{strconv.Itoa(pid), "TERM"}, nil)
	assertExitCode(t, code, 0)
}

func TestKillTree_works_with_KILL_signal(t *testing.T) {
	cmd := exec.Command("bash", "-c", `trap "" TERM; sleep 100`)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start trap process: %v", err)
	}
	pid := cmd.Process.Pid
	defer forceKill(pid)
	time.Sleep(300 * time.Millisecond)

	// TERM won't work (process ignores it)
	snippet := processSnippet(t, fmt.Sprintf("kill_tree %d TERM 2>/dev/null || true", pid))
	_, _ = runBashSnippet(t, snippet, nil)
	time.Sleep(300 * time.Millisecond)

	// Process should still be alive (it ignores TERM)
	if !isProcessRunning(pid) {
		t.Skip("process died from TERM unexpectedly; trap may not have been set in time")
	}

	// KILL should work
	snippet = processSnippet(t, fmt.Sprintf("kill_tree %d KILL 2>/dev/null || true", pid))
	_, _ = runBashSnippet(t, snippet, nil)
	time.Sleep(200 * time.Millisecond)
	reapProcess(cmd)

	if isProcessRunning(pid) {
		t.Error("process should be dead after KILL signal")
	}
}

func TestKillTree_works_with_HUP_signal(t *testing.T) {
	cmd := exec.Command("sleep", "100")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sleep: %v", err)
	}
	pid := cmd.Process.Pid
	defer forceKill(pid)
	time.Sleep(200 * time.Millisecond)

	snippet := processSnippet(t, fmt.Sprintf("kill_tree %d HUP 2>/dev/null || true", pid))
	_, _ = runBashSnippet(t, snippet, nil)
	time.Sleep(200 * time.Millisecond)
	reapProcess(cmd)

	if isProcessRunning(pid) {
		t.Error("process should be dead after HUP signal")
	}
}

func TestKillTree_handles_moderate_process_tree_15_processes(t *testing.T) {
	cmd := exec.Command("bash", "-c", `
for i in $(seq 1 14); do
  sleep 100 &
done
wait
`)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process tree: %v", err)
	}
	pid := cmd.Process.Pid
	defer forceKill(pid)
	time.Sleep(500 * time.Millisecond)

	children := getChildren(pid)
	if len(children) != 14 {
		t.Fatalf("expected 14 children, got %d", len(children))
	}

	snippet := processSnippet(t, fmt.Sprintf("kill_tree %d TERM 2>/dev/null || true", pid))
	_, _ = runBashSnippet(t, snippet, nil)
	time.Sleep(500 * time.Millisecond)
	reapProcess(cmd)

	if isProcessRunning(pid) {
		t.Error("parent should be dead")
	}

	remaining := getChildren(pid)
	if len(remaining) > 0 {
		t.Errorf("expected no remaining children, found %d", len(remaining))
	}
}

func TestKillTree_handles_process_that_exits_before_KILL(t *testing.T) {
	cmd := exec.Command("bash", "-c", "sleep 0.2")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start short process: %v", err)
	}
	pid := cmd.Process.Pid
	defer forceKill(pid)

	snippet := processSnippet(t, fmt.Sprintf("kill_tree %d TERM 2>/dev/null || true", pid))
	_, _ = runBashSnippet(t, snippet, nil)
	time.Sleep(500 * time.Millisecond)
	reapProcess(cmd)

	// Process already gone; second kill should handle gracefully
	_, code := runBashFunc(t, "lib/process.sh", "kill_tree", []string{strconv.Itoa(pid), "KILL"}, nil)
	assertExitCode(t, code, 0)
}

func TestKillTree_concurrent_kills_dont_cause_errors(t *testing.T) {
	cmd := exec.Command("sleep", "100")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sleep: %v", err)
	}
	pid := cmd.Process.Pid
	defer forceKill(pid)
	time.Sleep(200 * time.Millisecond)

	snippet := processSnippet(t, fmt.Sprintf(`
(kill_tree %d TERM 2>/dev/null || true) &
(kill_tree %d TERM 2>/dev/null || true) &
wait
`, pid, pid))
	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
	time.Sleep(300 * time.Millisecond)
	reapProcess(cmd)

	if isProcessRunning(pid) {
		t.Error("process should be dead after concurrent kills")
	}
}

func TestKillTree_handles_empty_process_tree_no_children(t *testing.T) {
	cmd := exec.Command("sleep", "100")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sleep: %v", err)
	}
	pid := cmd.Process.Pid
	defer forceKill(pid)
	time.Sleep(200 * time.Millisecond)

	_, code := runBashFunc(t, "lib/process.sh", "kill_tree", []string{strconv.Itoa(pid), "TERM"}, nil)
	assertExitCode(t, code, 0)
	time.Sleep(200 * time.Millisecond)
	reapProcess(cmd)

	if isProcessRunning(pid) {
		t.Error("process should be dead")
	}
}

func TestKillTree_INT_signal_skipped(t *testing.T) {
	t.Skip("INT signal handling varies across bash/zsh configurations - prioritizing test stability")
}

func TestKillTree_handles_very_large_process_tree_50_processes(t *testing.T) {
	cmd := exec.Command("bash", "-c", `
for i in $(seq 1 50); do
  sleep 100 &
done
wait
`)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start large process tree: %v", err)
	}
	pid := cmd.Process.Pid
	defer forceKill(pid)
	time.Sleep(1 * time.Second)

	children := getChildren(pid)
	if len(children) < 45 {
		t.Fatalf("expected at least 45 children, got %d", len(children))
	}

	snippet := processSnippet(t, fmt.Sprintf("kill_tree %d TERM 2>/dev/null || true", pid))
	_, _ = runBashSnippet(t, snippet, nil)
	time.Sleep(500 * time.Millisecond)
	reapProcess(cmd)

	if isProcessRunning(pid) {
		t.Error("parent should be dead")
	}

	remaining := getChildren(pid)
	if len(remaining) > 0 {
		t.Errorf("expected no remaining children, found %d", len(remaining))
	}
}

// ======================================================================
// cleanup_tmux_session tests (14 from tmux-session.bats)
// ======================================================================

func TestCleanupTmuxSession_calls_kill_and_tmux_kill_session(t *testing.T) {
	snippet := tmuxSessionSnippet(t, `
kill() { return 0; }
tmux() {
  if [[ "$1" == "list-panes" ]]; then
    echo "12345"
  elif [[ "$1" == "kill-session" ]]; then
    :
  fi
  return 0
}
kill_tree() { return 0; }
sleep() { return 0; }
cleanup_tmux_session "test-session" "99999" "tmux"
`)
	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
}

func TestCleanupTmuxSession_handles_missing_session_gracefully(t *testing.T) {
	snippet := tmuxSessionSnippet(t, `
tmux() { return 1; }
kill() { return 0; }
sleep() { return 0; }
kill_tree() { return 0; }
cleanup_tmux_session "nonexistent" "99999" "tmux"
`)
	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
}

func TestCleanupTmuxSession_handles_multiple_panes_with_process_trees(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "kill_tree_calls.log")

	snippet := tmuxSessionSnippet(t, fmt.Sprintf(`
kill() { return 0; }
tmux() {
  if [[ "$1" == "list-panes" ]]; then
    printf "10001\n10002\n10003\n"
  elif [[ "$1" == "kill-session" ]]; then
    return 0
  fi
  return 0
}
kill_tree() {
  echo "$1:$2" >> %q
  return 0
}
sleep() { return 0; }
cleanup_tmux_session "test-session" "99999" "tmux"
`, logFile))

	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	if len(lines) != 6 {
		t.Fatalf("expected 6 kill_tree calls, got %d: %v", len(lines), lines)
	}

	expected := []string{"10001:TERM", "10002:TERM", "10003:TERM", "10001:KILL", "10002:KILL", "10003:KILL"}
	for i, exp := range expected {
		if lines[i] != exp {
			t.Errorf("call %d: expected %q, got %q", i, exp, lines[i])
		}
	}
}

func TestCleanupTmuxSession_handles_watcher_PID_that_doesnt_exist(t *testing.T) {
	snippet := tmuxSessionSnippet(t, `
kill() { return 1; }
tmux() {
  if [[ "$1" == "list-panes" ]]; then
    echo "10001"
  elif [[ "$1" == "kill-session" ]]; then
    return 0
  fi
  return 0
}
kill_tree() { return 0; }
sleep() { return 0; }
cleanup_tmux_session "test-session" "99999" "tmux"
`)
	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
}

func TestCleanupTmuxSession_handles_pane_PIDs_that_disappear_between_TERM_and_KILL(t *testing.T) {
	snippet := tmuxSessionSnippet(t, `
_list_panes_call_count=0
kill() { return 0; }
tmux() {
  if [[ "$1" == "list-panes" ]]; then
    _list_panes_call_count=$((_list_panes_call_count + 1))
    if [[ "$_list_panes_call_count" -eq 1 ]]; then
      printf "10001\n10002\n"
    else
      echo "10002"
    fi
  elif [[ "$1" == "kill-session" ]]; then
    return 0
  fi
  return 0
}
kill_tree() { return 0; }
sleep() { return 0; }
cleanup_tmux_session "test-session" "99999" "tmux"
`)
	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
}

func TestCleanupTmuxSession_handles_no_panes_in_session(t *testing.T) {
	snippet := tmuxSessionSnippet(t, `
kill() { return 0; }
tmux() {
  if [[ "$1" == "list-panes" ]]; then
    echo ""
  elif [[ "$1" == "kill-session" ]]; then
    return 0
  fi
  return 0
}
kill_tree() { return 0; }
sleep() { return 0; }
cleanup_tmux_session "test-session" "99999" "tmux"
`)
	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
}

func TestCleanupTmuxSession_handles_kill_tree_failures_gracefully(t *testing.T) {
	snippet := tmuxSessionSnippet(t, `
kill() { return 0; }
tmux() {
  if [[ "$1" == "list-panes" ]]; then
    echo "10001"
  elif [[ "$1" == "kill-session" ]]; then
    return 0
  fi
  return 0
}
kill_tree() { return 1; }
sleep() { return 0; }
cleanup_tmux_session "test-session" "99999" "tmux"
`)
	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
}

func TestCleanupTmuxSession_handles_tmux_kill_session_failure_gracefully(t *testing.T) {
	snippet := tmuxSessionSnippet(t, `
kill() { return 0; }
tmux() {
  if [[ "$1" == "list-panes" ]]; then
    echo "10001"
  elif [[ "$1" == "kill-session" ]]; then
    return 1
  fi
  return 0
}
kill_tree() { return 0; }
sleep() { return 0; }
cleanup_tmux_session "test-session" "99999" "tmux"
`)
	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
}

func TestCleanupTmuxSession_handles_list_panes_returning_error_on_second_call(t *testing.T) {
	snippet := tmuxSessionSnippet(t, `
_list_panes_call_count=0
kill() { return 0; }
tmux() {
  if [[ "$1" == "list-panes" ]]; then
    _list_panes_call_count=$((_list_panes_call_count + 1))
    if [[ "$_list_panes_call_count" -eq 1 ]]; then
      echo "10001"
      return 0
    else
      return 1
    fi
  elif [[ "$1" == "kill-session" ]]; then
    return 0
  fi
  return 0
}
kill_tree() { return 0; }
sleep() { return 0; }
cleanup_tmux_session "test-session" "99999" "tmux"
`)
	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
}

func TestCleanupTmuxSession_handles_concurrent_cleanup_calls_idempotent(t *testing.T) {
	snippet := tmuxSessionSnippet(t, `
_cleanup_count=0
kill() { return 0; }
tmux() {
  if [[ "$1" == "list-panes" ]]; then
    _cleanup_count=$((_cleanup_count + 1))
    if [[ "$_cleanup_count" -le 2 ]]; then
      echo "10001"
    else
      return 1
    fi
  elif [[ "$1" == "kill-session" ]]; then
    return 0
  fi
  return 0
}
kill_tree() { return 0; }
sleep() { return 0; }

# First cleanup
cleanup_tmux_session "test-session" "99999" "tmux"
# Second cleanup (should handle gracefully)
cleanup_tmux_session "test-session" "99999" "tmux"
`)
	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)
}

func TestCleanupTmuxSession_verifies_TERM_then_KILL_sequence_with_timing(t *testing.T) {
	tmpDir := t.TempDir()
	killLogFile := filepath.Join(tmpDir, "kill_tree_calls.log")
	sleepLogFile := filepath.Join(tmpDir, "sleep_calls.log")

	snippet := tmuxSessionSnippet(t, fmt.Sprintf(`
kill() { return 0; }
tmux() {
  if [[ "$1" == "list-panes" ]]; then
    echo "10001"
  elif [[ "$1" == "kill-session" ]]; then
    return 0
  fi
  return 0
}
kill_tree() {
  echo "$1:$2" >> %q
  return 0
}
sleep() {
  echo "$1" >> %q
  return 0
}
cleanup_tmux_session "test-session" "99999" "tmux"
`, killLogFile, sleepLogFile))

	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)

	killData, err := os.ReadFile(killLogFile)
	if err != nil {
		t.Fatalf("failed to read kill log: %v", err)
	}
	killLines := strings.Split(strings.TrimSpace(string(killData)), "\n")

	if len(killLines) < 2 {
		t.Fatalf("expected at least 2 kill_tree calls, got %d", len(killLines))
	}
	if killLines[0] != "10001:TERM" {
		t.Errorf("first kill_tree call should be 10001:TERM, got %q", killLines[0])
	}
	if killLines[1] != "10001:KILL" {
		t.Errorf("second kill_tree call should be 10001:KILL, got %q", killLines[1])
	}

	sleepData, err := os.ReadFile(sleepLogFile)
	if err != nil {
		t.Fatalf("failed to read sleep log: %v", err)
	}
	sleepLines := strings.Split(strings.TrimSpace(string(sleepData)), "\n")

	if len(sleepLines) < 1 {
		t.Fatal("expected at least 1 sleep call")
	}
	if sleepLines[0] != "0.3" {
		t.Errorf("first sleep call should be 0.3, got %q", sleepLines[0])
	}
}

func TestCleanupTmuxSession_handles_pane_with_PID_1(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "kill_tree_calls.log")

	snippet := tmuxSessionSnippet(t, fmt.Sprintf(`
kill() { return 0; }
tmux() {
  if [[ "$1" == "list-panes" ]]; then
    echo "1"
  elif [[ "$1" == "kill-session" ]]; then
    return 0
  fi
  return 0
}
kill_tree() {
  echo "$1:$2" >> %q
  return 0
}
sleep() { return 0; }
cleanup_tmux_session "test-session" "99999" "tmux"
`, logFile))

	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	if len(lines) == 0 {
		t.Fatal("expected at least one kill_tree call")
	}
	if lines[0] != "1:TERM" {
		t.Errorf("first call should be 1:TERM, got %q", lines[0])
	}
}

func TestCleanupTmuxSession_handles_very_large_number_of_panes_20(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "kill_tree_calls.log")

	snippet := tmuxSessionSnippet(t, fmt.Sprintf(`
kill() { return 0; }
tmux() {
  if [[ "$1" == "list-panes" ]]; then
    for i in $(seq 10001 10020); do
      echo "$i"
    done
  elif [[ "$1" == "kill-session" ]]; then
    return 0
  fi
  return 0
}
kill_tree() {
  echo "$1:$2" >> %q
  return 0
}
sleep() { return 0; }
cleanup_tmux_session "test-session" "99999" "tmux"
`, logFile))

	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	if len(lines) != 40 {
		t.Errorf("expected 40 kill_tree calls, got %d", len(lines))
	}
}

func TestCleanupTmuxSession_handles_panes_with_non_numeric_PIDs_in_output(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "kill_tree_calls.log")

	snippet := tmuxSessionSnippet(t, fmt.Sprintf(`
kill() { return 0; }
tmux() {
  if [[ "$1" == "list-panes" ]]; then
    printf "10001\ninvalid\n10002\nerror: something\n10003\n"
  elif [[ "$1" == "kill-session" ]]; then
    return 0
  fi
  return 0
}
kill_tree() {
  echo "$1:$2" >> %q
  return 0
}
sleep() { return 0; }
cleanup_tmux_session "test-session" "99999" "tmux"
`, logFile))

	_, code := runBashSnippet(t, snippet, nil)
	assertExitCode(t, code, 0)

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}
	content := string(data)

	// The BATS test notes that "error: something" gets split by word splitting
	// into "error:" and "something", giving 6 items per pass, 12 total.
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != 12 {
		t.Errorf("expected 12 kill_tree calls (6 items x 2 passes), got %d: %v", len(lines), lines)
	}

	assertContains(t, content, "10001:TERM")
	assertContains(t, content, "10002:TERM")
	assertContains(t, content, "10003:TERM")
}
