package tui_test

import (
	"testing"

	"github.com/jackuait/ghost-tab/internal/tui"
)

func TestTranslateRune_Russian(t *testing.T) {
	tests := []struct {
		input    rune
		expected rune
	}{
		// Navigation keys
		{'о', 'j'}, // j - move down
		{'л', 'k'}, // k - move up
		// Action keys
		{'ф', 'a'}, // a - add project
		{'в', 'd'}, // d - delete project
		{'ы', 's'}, // s - settings
		{'щ', 'o'}, // o - open once
		{'з', 'p'}, // p - plain terminal
		{'и', 'b'}, // b - back from settings
		{'й', 'q'}, // q - quit delete mode
		// Confirm keys
		{'н', 'y'}, // y - confirm
		{'т', 'n'}, // n - deny
		// Other letters for completeness
		{'ц', 'w'}, {'у', 'e'}, {'к', 'r'}, {'е', 't'},
		{'г', 'u'}, {'ш', 'i'},
		{'р', 'h'}, {'д', 'l'},
		{'я', 'z'}, {'ч', 'x'}, {'с', 'c'}, {'м', 'v'},
		{'ь', 'm'},
		// Uppercase
		{'О', 'J'}, {'Л', 'K'},
		{'Ф', 'A'}, {'В', 'D'}, {'Ы', 'S'},
		{'Щ', 'O'}, {'З', 'P'},
		{'Н', 'Y'}, {'Т', 'N'},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := tui.TranslateRune(tt.input)
			if got != tt.expected {
				t.Errorf("TranslateRune(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTranslateRune_Ukrainian(t *testing.T) {
	tests := []struct {
		input    rune
		expected rune
	}{
		// Ukrainian-specific letters (differ from Russian)
		{'і', 's'}, // Ukrainian і on s key (Russian has ы)
		{'І', 'S'}, // uppercase
		// Shared with Russian (verify they still work)
		{'о', 'j'}, {'л', 'k'}, {'ф', 'a'},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := tui.TranslateRune(tt.input)
			if got != tt.expected {
				t.Errorf("TranslateRune(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTranslateRune_Hebrew(t *testing.T) {
	tests := []struct {
		input    rune
		expected rune
	}{
		// Navigation keys
		{'ח', 'j'}, // j - move down
		{'ל', 'k'}, // k - move up
		// Action keys
		{'ש', 'a'}, // a - add project
		{'ג', 'd'}, // d - delete project
		{'ד', 's'}, // s - settings
		{'ם', 'o'}, // o - open once
		{'פ', 'p'}, // p - plain terminal
		{'נ', 'b'}, // b - back from settings
		// Confirm keys
		{'ט', 'y'}, // y - confirm
		{'מ', 'n'}, // n - deny
		// Other letters
		{'ק', 'e'}, {'ר', 'r'}, {'א', 't'},
		{'ו', 'u'}, {'ן', 'i'},
		{'י', 'h'}, {'ך', 'l'},
		{'כ', 'f'}, {'ע', 'g'},
		{'ז', 'z'}, {'ס', 'x'}, {'ב', 'c'}, {'ה', 'v'},
		{'צ', 'm'},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := tui.TranslateRune(tt.input)
			if got != tt.expected {
				t.Errorf("TranslateRune(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTranslateRune_Arabic(t *testing.T) {
	tests := []struct {
		input    rune
		expected rune
	}{
		// Navigation keys
		{'ت', 'j'}, // j - move down
		{'ن', 'k'}, // k - move up
		// Action keys
		{'ش', 'a'}, // a - add project
		{'ي', 'd'}, // d - delete project
		{'س', 's'}, // s - settings
		{'خ', 'o'}, // o - open once
		{'ح', 'p'}, // p - plain terminal
		// Confirm keys
		{'غ', 'y'}, // y - confirm
		{'ى', 'n'}, // n - deny
		// Other letters
		{'ض', 'q'}, {'ص', 'w'}, {'ث', 'e'}, {'ق', 'r'},
		{'ف', 't'}, {'ع', 'u'}, {'ه', 'i'},
		{'ب', 'f'}, {'ل', 'g'}, {'ا', 'h'}, {'م', 'l'},
		{'ئ', 'z'}, {'ء', 'x'}, {'ؤ', 'c'}, {'ر', 'v'},
		{'ة', 'm'},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := tui.TranslateRune(tt.input)
			if got != tt.expected {
				t.Errorf("TranslateRune(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTranslateRune_EnglishPassthrough(t *testing.T) {
	// English ASCII letters should pass through unchanged
	for r := 'a'; r <= 'z'; r++ {
		got := tui.TranslateRune(r)
		if got != r {
			t.Errorf("TranslateRune(%q) = %q, want passthrough", r, got)
		}
	}
	for r := 'A'; r <= 'Z'; r++ {
		got := tui.TranslateRune(r)
		if got != r {
			t.Errorf("TranslateRune(%q) = %q, want passthrough", r, got)
		}
	}
}

func TestTranslateRune_NumbersPassthrough(t *testing.T) {
	for r := '0'; r <= '9'; r++ {
		got := tui.TranslateRune(r)
		if got != r {
			t.Errorf("TranslateRune(%q) = %q, want passthrough", r, got)
		}
	}
}

func TestTranslateRune_UnknownPassthrough(t *testing.T) {
	unknowns := []rune{'€', '£', '¥', '§', '日', '本', '🎉'}
	for _, r := range unknowns {
		got := tui.TranslateRune(r)
		if got != r {
			t.Errorf("TranslateRune(%q) = %q, want passthrough", r, got)
		}
	}
}

func TestTranslateRune_SpacePassthrough(t *testing.T) {
	got := tui.TranslateRune(' ')
	if got != ' ' {
		t.Errorf("TranslateRune(' ') = %q, want ' '", got)
	}
}
