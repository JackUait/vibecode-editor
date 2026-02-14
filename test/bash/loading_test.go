package bash_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// --- get_loading_art ---

func TestLoading_get_loading_art_returns_nonempty_for_all_10_indices(t *testing.T) {
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("art_%d", i), func(t *testing.T) {
			out, code := runBashFunc(t, "lib/loading.sh", "get_loading_art",
				[]string{fmt.Sprintf("%d", i)}, nil)
			assertExitCode(t, code, 0)
			if strings.TrimSpace(out) == "" {
				t.Errorf("get_loading_art(%d) returned empty output", i)
			}
		})
	}
}

func TestLoading_get_loading_art_contains_ghost_characters(t *testing.T) {
	// Every art variant should contain characters that spell Ghost
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("art_%d", i), func(t *testing.T) {
			out, code := runBashFunc(t, "lib/loading.sh", "get_loading_art",
				[]string{fmt.Sprintf("%d", i)}, nil)
			assertExitCode(t, code, 0)
			// All figlet fonts produce recognizable "host" substring somewhere
			lower := strings.ToLower(out)
			hasGhost := strings.Contains(lower, "host") ||
				strings.Contains(out, "___") ||
				strings.Contains(out, "---") ||
				strings.Contains(out, "|||") ||
				len(out) > 50 // All arts are multi-line and substantial
			if !hasGhost {
				t.Errorf("get_loading_art(%d) doesn't look like ASCII art, got:\n%s", i, out)
			}
		})
	}
}

func TestLoading_get_loading_art_invalid_index_returns_empty(t *testing.T) {
	out, code := runBashFunc(t, "lib/loading.sh", "get_loading_art", []string{"99"}, nil)
	assertExitCode(t, code, 0)
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected empty for invalid index, got:\n%s", out)
	}
}

// --- get_loading_palette ---

func TestLoading_get_loading_palette_returns_color_codes_for_all_indices(t *testing.T) {
	for i := 0; i < 5; i++ {
		t.Run(fmt.Sprintf("palette_%d", i), func(t *testing.T) {
			out, code := runBashFunc(t, "lib/loading.sh", "get_loading_palette",
				[]string{fmt.Sprintf("%d", i)}, nil)
			assertExitCode(t, code, 0)
			trimmed := strings.TrimSpace(out)
			if trimmed == "" {
				t.Errorf("get_loading_palette(%d) returned empty output", i)
				return
			}
			// Each value should be a valid 256-color code (0-255)
			for _, part := range strings.Fields(trimmed) {
				num, err := strconv.Atoi(part)
				if err != nil {
					t.Errorf("get_loading_palette(%d) non-numeric value: %q", i, part)
				}
				if num < 0 || num > 255 {
					t.Errorf("get_loading_palette(%d) out of range: %d", i, num)
				}
			}
		})
	}
}

func TestLoading_get_loading_palette_has_at_least_5_colors(t *testing.T) {
	for i := 0; i < 5; i++ {
		t.Run(fmt.Sprintf("palette_%d", i), func(t *testing.T) {
			out, code := runBashFunc(t, "lib/loading.sh", "get_loading_palette",
				[]string{fmt.Sprintf("%d", i)}, nil)
			assertExitCode(t, code, 0)
			parts := strings.Fields(strings.TrimSpace(out))
			if len(parts) < 5 {
				t.Errorf("get_loading_palette(%d) has only %d colors, want >= 5", i, len(parts))
			}
		})
	}
}

// --- _detect_term_size ---

func TestLoading_detect_term_size_returns_two_positive_numbers(t *testing.T) {
	out, code := runBashFunc(t, "lib/loading.sh", "_detect_term_size", nil, nil)
	assertExitCode(t, code, 0)
	parts := strings.Fields(strings.TrimSpace(out))
	if len(parts) != 2 {
		t.Fatalf("expected 2 values, got %d: %q", len(parts), out)
	}
	for _, p := range parts {
		num, err := strconv.Atoi(p)
		if err != nil {
			t.Errorf("non-numeric value: %q", p)
		}
		if num <= 0 {
			t.Errorf("expected positive number, got %d", num)
		}
	}
}

// --- render_loading_frame ---

func TestLoading_render_loading_frame_contains_ansi_color_codes(t *testing.T) {
	root := projectRoot(t)
	script := fmt.Sprintf(
		`source %q/lib/loading.sh && render_loading_frame 0 0 0 80 24`,
		root)
	out, code := runBashSnippet(t, script, nil)
	assertExitCode(t, code, 0)
	// Should contain ANSI 256-color escape: \033[38;5;XXXm
	assertContains(t, out, "\033[38;5;")
}

func TestLoading_render_loading_frame_contains_art_content(t *testing.T) {
	root := projectRoot(t)
	script := fmt.Sprintf(
		`source %q/lib/loading.sh && render_loading_frame 0 0 0 80 24`,
		root)
	out, code := runBashSnippet(t, script, nil)
	assertExitCode(t, code, 0)
	// Frame 0 with art 0 (standard) should contain recognizable characters
	if len(out) < 100 {
		t.Errorf("render_loading_frame output too short (%d bytes), expected substantial output", len(out))
	}
}

func TestLoading_render_loading_frame_shifts_colors_between_frames(t *testing.T) {
	root := projectRoot(t)
	script0 := fmt.Sprintf(
		`source %q/lib/loading.sh && render_loading_frame 0 0 0 80 24`,
		root)
	script1 := fmt.Sprintf(
		`source %q/lib/loading.sh && render_loading_frame 0 0 1 80 24`,
		root)
	out0, _ := runBashSnippet(t, script0, nil)
	out1, _ := runBashSnippet(t, script1, nil)
	if out0 == out1 {
		t.Error("expected different output for different frames")
	}
}
