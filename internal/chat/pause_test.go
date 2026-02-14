package chat

import (
	"context"
	"testing"
	"time"
)

func TestPauseControllerToggle(t *testing.T) {
	p := NewPauseController()
	if p.IsPaused() {
		t.Fatal("new controller should not be paused")
	}
	if !p.Toggle() {
		t.Fatal("expected paused after first toggle")
	}
	if p.Toggle() {
		t.Fatal("expected resumed after second toggle")
	}
}

func TestWaitIfPausedResumes(t *testing.T) {
	p := NewPauseController()
	p.Toggle() // paused

	done := make(chan struct{})
	go func() {
		time.Sleep(80 * time.Millisecond)
		p.Toggle() // resume
		close(done)
	}()

	start := time.Now()
	if err := p.WaitIfPaused(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if time.Since(start) < 50*time.Millisecond {
		t.Fatal("expected wait while paused")
	}
	<-done
}
