package tui

import "fmt"

// ZzzAnimation renders an animated sleeping indicator with Z characters
// that float upward in a cycle.
type ZzzAnimation struct {
	frame  int
	frames []string
}

// NewZzzAnimation creates a new Zzz animation with 4 frames.
func NewZzzAnimation() *ZzzAnimation {
	dim := "\033[2m"
	reset := "\033[0m"

	frames := []string{
		dim + "        z" + reset + "\n" +
			dim + "      Z" + reset + "\n" +
			"    Z",
		dim + "       z" + reset + "\n" +
			dim + "     Z" + reset + "\n" +
			"   Z",
		dim + "      z" + reset + "\n" +
			dim + "    Z" + reset + "\n" +
			"  Z",
		dim + "       z" + reset + "\n" +
			dim + "     Z" + reset + "\n" +
			"   Z",
	}

	return &ZzzAnimation{
		frame:  0,
		frames: frames,
	}
}

// Frame returns the current frame index.
func (z *ZzzAnimation) Frame() int {
	return z.frame
}

// TotalFrames returns the number of animation frames.
func (z *ZzzAnimation) TotalFrames() int {
	return len(z.frames)
}

// Tick advances to the next frame.
func (z *ZzzAnimation) Tick() {
	z.frame = (z.frame + 1) % len(z.frames)
}

// Reset returns to frame 0.
func (z *ZzzAnimation) Reset() {
	z.frame = 0
}

// View returns the current frame's Zzz string.
func (z *ZzzAnimation) View() string {
	if len(z.frames) == 0 {
		return ""
	}
	return z.frames[z.frame]
}

// ViewColored returns the current frame with the given ANSI color applied.
func (z *ZzzAnimation) ViewColored(color string) string {
	if len(z.frames) == 0 {
		return ""
	}
	return fmt.Sprintf("%s%s\033[0m", color, z.frames[z.frame])
}
