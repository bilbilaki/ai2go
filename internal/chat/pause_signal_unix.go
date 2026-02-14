//go:build !windows

package chat

import (
	"os"
	"os/signal"
	"syscall"
)

func RegisterPauseSignal(ch chan<- os.Signal) {
	signal.Notify(ch, syscall.SIGTSTP)
}

func StopPauseSignal(ch chan<- os.Signal) {
	signal.Stop(ch)
}
