//go:build windows

package chat

import "os"

func RegisterPauseSignal(ch chan<- os.Signal) {}

func StopPauseSignal(ch chan<- os.Signal) {}
