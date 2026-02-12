package tui_test

import (
	"testing"

	"github.com/jackuait/ghost-tab/internal/tui"
)

func TestZzzAnimation_InitialFrame(t *testing.T) {
	z := tui.NewZzzAnimation()
	if z.Frame() != 0 {
		t.Errorf("initial frame should be 0, got %d", z.Frame())
	}
}

func TestZzzAnimation_TickAdvancesFrame(t *testing.T) {
	z := tui.NewZzzAnimation()
	z.Tick()
	if z.Frame() != 1 {
		t.Errorf("after tick: expected frame 1, got %d", z.Frame())
	}
}

func TestZzzAnimation_FrameWraps(t *testing.T) {
	z := tui.NewZzzAnimation()
	totalFrames := z.TotalFrames()
	for i := 0; i < totalFrames; i++ {
		z.Tick()
	}
	if z.Frame() != 0 {
		t.Errorf("after %d ticks: expected frame 0 (wrapped), got %d", totalFrames, z.Frame())
	}
}

func TestZzzAnimation_FramesDiffer(t *testing.T) {
	z := tui.NewZzzAnimation()
	frame0 := z.View()
	z.Tick()
	frame1 := z.View()

	if frame0 == frame1 {
		t.Error("consecutive Zzz frames should differ")
	}
}

func TestZzzAnimation_Reset(t *testing.T) {
	z := tui.NewZzzAnimation()
	z.Tick()
	z.Tick()
	z.Reset()
	if z.Frame() != 0 {
		t.Errorf("after reset: expected frame 0, got %d", z.Frame())
	}
}

func TestZzzAnimation_ViewContainsZ(t *testing.T) {
	z := tui.NewZzzAnimation()
	view := z.View()
	if view == "" {
		t.Error("Zzz view should not be empty")
	}
}
