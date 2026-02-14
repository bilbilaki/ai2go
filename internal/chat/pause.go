package chat

import (
	"context"
	"sync"
	"time"
)

// PauseController controls cooperative pause/resume checkpoints in the chat loop.
// It is intentionally checkpoint-based so we don't break tool-call/response integrity.
type PauseController struct {
	mu     sync.RWMutex
	paused bool
}

func NewPauseController() *PauseController {
	return &PauseController{}
}

func (p *PauseController) Toggle() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.paused = !p.paused
	return p.paused
}

func (p *PauseController) IsPaused() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.paused
}

func (p *PauseController) WaitIfPaused(ctx context.Context) error {
	if p == nil {
		return nil
	}
	for p.IsPaused() {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
			}
		} else {
			time.Sleep(200 * time.Millisecond)
		}
	}
	return nil
}
