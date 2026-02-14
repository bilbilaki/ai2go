package tools

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// KillProcessTree terminates a process and all its child processes
func KillProcessTree(pid int, signal string) error {
	// Validate PID
	if pid <= 0 {
		return fmt.Errorf("invalid PID: %d", pid)
	}

	// Convert signal name to signal number
	sigNum, err := SignalToNumber(signal)
	if err != nil {
		return fmt.Errorf("invalid signal: %w", err)
	}

	// Get all child PIDs recursively
	childPIDs, err := GetChildPIDsRecursive(pid)
	if err != nil {
		return fmt.Errorf("failed to get child processes: %w", err)
	}

	// Send signal to all child processes first (bottom-up)
	// This is more graceful than top-down
	for i := len(childPIDs) - 1; i >= 0; i-- {
		childPID := childPIDs[i]
		if err := SendSignal(childPID, sigNum); err != nil {
			// Log but continue with other processes
			fmt.Fprintf(os.Stderr, "Warning: Failed to send signal to child PID %d: %v\n", childPID, err)
		}
	}

	// Finally send signal to the root process
	if err := SendSignal(pid, sigNum); err != nil {
		return fmt.Errorf("failed to send signal to root PID %d: %w", pid, err)
	}

	return nil
}

// KillProcessTreeWithTimeout terminates with graceful timeout before force kill
func KillProcessTreeWithTimeout(pid int, signal string, gracefulTimeout int, force bool) error {
	if force {
		// Force kill immediately
		return KillProcessTree(pid, "KILL")
	}

	// Try graceful termination first
	gracefulSignal := signal
	if gracefulSignal == "" {
		gracefulSignal = "TERM"
	}

	// Send graceful signal
	if err := KillProcessTree(pid, gracefulSignal); err != nil {
		return fmt.Errorf("graceful termination failed: %w", err)
	}

	// Wait for processes to terminate
	if gracefulTimeout > 0 {
		deadline := time.Now().Add(time.Duration(gracefulTimeout) * time.Second)
		
		for time.Now().Before(deadline) {
			// Check if process tree still exists
			exists, err := ProcessTreeExists(pid)
			if err != nil {
				return fmt.Errorf("failed to check process tree: %w", err)
			}
			
			if !exists {
				return nil // All processes terminated
			}
			
			time.Sleep(100 * time.Millisecond)
		}
		
		// Timeout reached, force kill
		fmt.Fprintf(os.Stderr, "Graceful timeout reached, forcing kill of PID %d\n", pid)
		return KillProcessTree(pid, "KILL")
	}
	
	return nil
}

// GetChildPIDsRecursive gets all child PIDs recursively
func GetChildPIDsRecursive(pid int) ([]int, error) {
	var childPIDs []int
	
	// Use pgrep to get direct children
	cmd := exec.Command("pgrep", "-P", strconv.Itoa(pid))
	output, err := cmd.Output()
	if err != nil {
		// pgrep returns exit code 1 when no processes found
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return childPIDs, nil // No children
		}
		return nil, fmt.Errorf("pgrep failed: %w", err)
	}
	
	// Parse PIDs from output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		childPID, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil {
			continue // Skip invalid lines
		}
		
		// Add this child PID
		childPIDs = append(childPIDs, childPID)
		
		// Recursively get grandchildren
		grandChildren, err := GetChildPIDsRecursive(childPID)
		if err != nil {
			return nil, err
		}
		childPIDs = append(childPIDs, grandChildren...)
	}
	
	return childPIDs, nil
}

// Alternative implementation using /proc filesystem (Linux-specific)
func GetChildPIDsFromProc(pid int) ([]int, error) {
	var childPIDs []int
	
	// Read /proc/[pid]/task/[tid]/children if available (Linux 3.4+)
	childrenPath := fmt.Sprintf("/proc/%d/task/%d/children", pid, pid)
	if content, err := os.ReadFile(childrenPath); err == nil {
		// Parse space-separated child PIDs
		fields := strings.Fields(string(content))
		for _, field := range fields {
			childPID, err := strconv.Atoi(field)
			if err == nil {
				childPIDs = append(childPIDs, childPID)
				
				// Recursively get grandchildren
				grandChildren, err := GetChildPIDsFromProc(childPID)
				if err == nil {
					childPIDs = append(childPIDs, grandChildren...)
				}
			}
		}
		return childPIDs, nil
	}
	
	// Fallback: use ps command
	cmd := exec.Command("ps", "-o", "pid", "--no-headers", "--ppid", strconv.Itoa(pid))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ps failed: %w", err)
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		childPID, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil {
			continue
		}
		
		childPIDs = append(childPIDs, childPID)
		
		// Recursively get grandchildren
		grandChildren, err := GetChildPIDsFromProc(childPID)
		if err == nil {
			childPIDs = append(childPIDs, grandChildren...)
		}
	}
	
	return childPIDs, nil
}

// SendSignal sends a signal to a process
func SendSignal(pid int, sig syscall.Signal) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}
	
	// Try to send signal
	if err := process.Signal(sig); err != nil {
		// Check if process already terminated
		if err.Error() == "os: process already finished" {
			return nil // Process already dead, not an error
		}
		return fmt.Errorf("failed to send signal: %w", err)
	}
	
	return nil
}

// SignalToNumber converts signal name to signal number
func SignalToNumber(signal string) (syscall.Signal, error) {
	// Common signals
	signals := map[string]syscall.Signal{
		"HUP":     syscall.SIGHUP,    // 1
		"INT":     syscall.SIGINT,    // 2
		"QUIT":    syscall.SIGQUIT,   // 3
		"KILL":    syscall.SIGKILL,   // 9
		"TERM":    syscall.SIGTERM,   // 15
		"USR1":    syscall.SIGUSR1,   // 10
		"USR2":    syscall.SIGUSR2,   // 12
		"STOP":    syscall.SIGSTOP,   // 19
		"CONT":    syscall.SIGCONT,   // 18
		"TSTP":    syscall.SIGTSTP,   // 20
		"CHLD":    syscall.SIGCHLD,   // 17
		"PIPE":    syscall.SIGPIPE,   // 13
		"ALRM":    syscall.SIGALRM,   // 14
		"VTALRM":  syscall.SIGVTALRM, // 26
		"PROF":    syscall.SIGPROF,   // 27
		"WINCH":   syscall.SIGWINCH,  // 28
	}
	
	// Check if signal is a number
	if num, err := strconv.Atoi(signal); err == nil {
		return syscall.Signal(num), nil
	}
	
	// Look up by name (case-insensitive)
	sigUpper := strings.ToUpper(signal)
	if sig, ok := signals[sigUpper]; ok {
		return sig, nil
	}
	
	// Try syscall.Signal_<name> if available
	switch sigUpper {
	case "ABRT":
		return syscall.SIGABRT, nil
	case "BUS":
		return syscall.SIGBUS, nil
	case "FPE":
		return syscall.SIGFPE, nil
	case "ILL":
		return syscall.SIGILL, nil
	case "SEGV":
		return syscall.SIGSEGV, nil
	case "TRAP":
		return syscall.SIGTRAP, nil
	}
	
	return 0, fmt.Errorf("unknown signal: %s", signal)
}

// ProcessTreeExists checks if any process in the tree still exists
func ProcessTreeExists(rootPID int) (bool, error) {
	// Check if root process exists
	if _, err := os.FindProcess(rootPID); err != nil {
		return false, nil // Root process doesn't exist
	}
	
	// Check if process is still running by sending signal 0
	process, _ := os.FindProcess(rootPID)
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return false, nil // Process not running
	}
	
	// Check children
	childPIDs, err := GetChildPIDsRecursive(rootPID)
	if err != nil {
		return true, nil // Assume exists if we can't check
	}
	
	// Check if any child still exists
	for _, pid := range childPIDs {
		childProc, _ := os.FindProcess(pid)
		if err := childProc.Signal(syscall.Signal(0)); err == nil {
			return true, nil // At least one child still running
		}
	}
	
	return false, nil // No running processes in tree
}

// GetProcessTree returns the process tree as a structured map
func GetProcessTree(rootPID int) (map[int][]int, error) {
	tree := make(map[int][]int)
	
	var buildTree func(int) error
	buildTree = func(pid int) error {
		_, err := GetChildPIDsRecursive(pid)
		if err != nil {
			return err
		}
		
		// Filter to only direct children
		var directChildren []int
		cmd := exec.Command("pgrep", "-P", strconv.Itoa(pid))
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}
				childPID, err := strconv.Atoi(strings.TrimSpace(line))
				if err == nil {
					directChildren = append(directChildren, childPID)
				}
			}
		}
		
		tree[pid] = directChildren
		
		// Recursively build tree for children
		for _, childPID := range directChildren {
			if err := buildTree(childPID); err != nil {
				return err
			}
		}
		
		return nil
	}
	
	if err := buildTree(rootPID); err != nil {
		return nil, err
	}
	
	return tree, nil
}