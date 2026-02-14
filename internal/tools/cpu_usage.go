package tools
import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)
// CPUStats holds CPU time information
type CPUStats struct {
	User      uint64
	Nice      uint64
	System    uint64
	Idle      uint64
	IOWait    uint64
	IRQ       uint64
	SoftIRQ   uint64
	Steal     uint64
	Guest     uint64
	GuestNice uint64
}

// ProcessStats holds process CPU time information
type ProcessStats struct {
	PID  int
	UTime uint64 // user time in jiffies (clock ticks)
	STime uint64 // system time in jiffies (clock ticks)
}

// GetTotalCPUTime reads total CPU time from /proc/stat
func GetTotalCPUTime() (uint64, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, fmt.Errorf("failed to open /proc/stat: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 11 {
				return 0, fmt.Errorf("invalid /proc/stat format")
			}

			var total uint64
			for i := 1; i < len(fields); i++ {
				val, err := strconv.ParseUint(fields[i], 10, 64)
				if err != nil {
					return 0, fmt.Errorf("failed to parse CPU time: %w", err)
				}
				total += val
			}
			return total, nil
		}
	}

	return 0, fmt.Errorf("cpu line not found in /proc/stat")
}

// GetProcessStats reads process CPU times from /proc/<pid>/stat
func GetProcessStats(pid int) (*ProcessStats, error) {
	statPath := filepath.Join("/proc", strconv.Itoa(pid), "stat")
	content, err := os.ReadFile(statPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read process stat: %w", err)
	}

	// Parse the stat file
	// Format: pid (comm) state ppid pgrp session tty_nr tpgid flags ...
	// We need fields 13 (utime) and 14 (stime) (0-indexed)
	statStr := string(content)
	
	// Find the last closing parenthesis to handle process names with spaces
	lastParen := strings.LastIndex(statStr, ")")
	if lastParen == -1 {
		return nil, fmt.Errorf("invalid stat format")
	}

	// Split the remaining part after the command name
	remaining := strings.TrimSpace(statStr[lastParen+1:])
	fields := strings.Fields(remaining)
	
	if len(fields) < 14 {
		return nil, fmt.Errorf("stat file too short")
	}

	// utime is field 13 (index 12), stime is field 14 (index 13)
	utime, err := strconv.ParseUint(fields[12], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse utime: %w", err)
	}

	stime, err := strconv.ParseUint(fields[13], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse stime: %w", err)
	}

	return &ProcessStats{
		PID:   pid,
		UTime: utime,
		STime: stime,
	}, nil
}

// GetProcessCPUUsage calculates CPU usage percentage for multiple PIDs
// Returns a map of PID to CPU usage percentage
func GetProcessCPUUsage(pids []int) (map[int]float64, error) {
	// Get initial total CPU time
	totalBefore, err := GetTotalCPUTime()
	if err != nil {
		return nil, err
	}

	// Get initial process stats
	processBefore := make(map[int]*ProcessStats)
	for _, pid := range pids {
		stats, err := GetProcessStats(pid)
		if err != nil {
			// If process doesn't exist, use zero values
			processBefore[pid] = &ProcessStats{PID: pid, UTime: 0, STime: 0}
		} else {
			processBefore[pid] = stats
		}
	}

	// Wait for 1 second
	time.Sleep(1 * time.Second)

	// Get final total CPU time
	totalAfter, err := GetTotalCPUTime()
	if err != nil {
		return nil, err
	}

	// Calculate CPU usage for each process
	results := make(map[int]float64)
	totalDelta := float64(totalAfter - totalBefore)
	
	if totalDelta == 0 {
		return results, nil // Avoid division by zero
	}

	for _, pid := range pids {
		beforeStats := processBefore[pid]
		
		// Get current process stats
		afterStats, err := GetProcessStats(pid)
		if err != nil {
			// Process might have terminated
			results[pid] = 0
			continue
		}

		// Calculate process CPU time delta
		beforeTotal := beforeStats.UTime + beforeStats.STime
		afterTotal := afterStats.UTime + afterStats.STime
		processDelta := float64(afterTotal - beforeTotal)

		// Calculate CPU usage percentage
		cpuUsage := (processDelta / totalDelta) * 100
		results[pid] = cpuUsage
	}

	return results, nil
}

// GetProcessCPUUsageSimple is a simpler version that returns CPU usage as integers
// Similar to the bash script's output
func GetProcessCPUUsageSimple(pids []int) (map[int]int, error) {
	results, err := GetProcessCPUUsage(pids)
	if err != nil {
		return nil, err
	}

	// Convert to integers (rounding)
	intResults := make(map[int]int)
	for pid, usage := range results {
		intResults[pid] = int(usage + 0.5) // Round to nearest integer
	}

	return intResults, nil
}

// MonitorProcesses continuously monitors CPU usage of given PIDs
// interval: monitoring interval in seconds
// duration: total monitoring duration in seconds (0 for infinite)
func MonitorProcesses(pids []int, interval, duration float64) error {
	if interval <= 0 {
		interval = 1.0
	}

	startTime := time.Now()
	
	for {
		// Check if we should stop
		if duration > 0 && time.Since(startTime).Seconds() >= duration {
			break
		}

		// Get CPU usage
		usage, err := GetProcessCPUUsage(pids)
		if err != nil {
			return fmt.Errorf("failed to get CPU usage: %w", err)
		}

		// Print results
		fmt.Printf("\n[%s] CPU Usage:\n", time.Now().Format("15:04:05"))
		for _, pid := range pids {
			if cpu, ok := usage[pid]; ok {
				fmt.Printf("  PID %d: %.1f%%\n", pid, cpu)
			} else {
				fmt.Printf("  PID %d: Not found\n", pid)
			}
		}

		// Wait for next interval
		time.Sleep(time.Duration(interval * float64(time.Second)))
	}

	return nil
}