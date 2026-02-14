package tools

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// ProcessInfo represents process information similar to ps output
type ProcessInfo struct {
	PID     int     `json:"pid"`
	PPID    int     `json:"ppid"`
	CPU     float64 `json:"cpu"`  // CPU usage percentage
	Memory  float64 `json:"mem"`  // Memory usage percentage
	Command string  `json:"cmd"`  // Full command line
	Name    string  `json:"name"` // Process name (first part of command)
	State   string  `json:"state"` // Process state
	User    string  `json:"user"`  // Username
	VSZ     uint64  `json:"vsz"`   // Virtual memory size in KB
	RSS     uint64  `json:"rss"`   // Resident set size in KB
	UpTime   string  `json:"uptime"` // Process uptime
	SleepTime string  `json:"sleeptime"` // Process sleep time
}

// GetProcessList returns a list of all processes with detailed information
func GetProcessList() ([]ProcessInfo, error) {
	// Get system memory info
	memInfo, err := GetMemoryInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	// Get page size
	pageSize := os.Getpagesize()

	// Read all process directories
	procDir := "/proc"
	entries, err := os.ReadDir(procDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc: %w", err)
	}

	var processes []ProcessInfo
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Process each PID directory concurrently
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pidStr := entry.Name()
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue // Skip non-numeric directories
		}

		wg.Add(1)
		go func(pid int) {
			defer wg.Done()

			process, err := GetProcessInfo(pid, pageSize, memInfo.TotalKB)
			if err != nil {
				// Skip processes we can't read (usually due to permissions)
				return
			}

			mu.Lock()
			processes = append(processes, process)
			mu.Unlock()
		}(pid)
	}

	wg.Wait()

	return processes, nil
}

// GetProcessListSimple returns basic process info (similar to bash script output)
func GetProcessListSimple() ([]map[string]string, error) {
	memInfo, err := GetMemoryInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}

	pageSize := os.Getpagesize()
	procDir := "/proc"
	entries, err := os.ReadDir(procDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc: %w", err)
	}

	var processes []map[string]string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pidStr := entry.Name()
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		process, err := GetProcessSimple(pid, pageSize, memInfo.TotalKB)
		if err != nil {
			continue
		}

		processes = append(processes, process)
	}

	return processes, nil
}

// GetProcessSimple gets basic process info (mimics bash script output)
func GetProcessSimple(pid, pageSize int, totalMemoryKB uint64) (map[string]string, error) {
	statPath := filepath.Join("/proc", strconv.Itoa(pid), "stat")
	statFile, err := os.Open(statPath)
	if err != nil {
		return nil, err
	}
	defer statFile.Close()

	// Read stat file
	statContent, err := io.ReadAll(statFile)
	if err != nil {
		return nil, err
	}

	// Parse stat file
	statStr := string(statContent)
	// Find the last closing parenthesis to handle process names with spaces
	lastParen := strings.LastIndex(statStr, ")")
	if lastParen == -1 {
		return nil, fmt.Errorf("invalid stat format")
	}

	// Split the remaining part after the command name
	remaining := strings.TrimSpace(statStr[lastParen+1:])
	fields := strings.Fields(remaining)
	
	if len(fields) < 24 {
		return nil, fmt.Errorf("stat file too short")
	}

	// Parse fields
	// Field indices (0-indexed after command name):
	// 0: state, 1: ppid, 22: utime, 23: stime, 23: rss (24th field in 1-indexed)
	ppid, _ := strconv.Atoi(fields[1])
	rssPages, _ := strconv.ParseUint(fields[23], 10, 64)

	// Calculate memory percentage (mimics bash script calculation)
	rssBytes := rssPages * uint64(pageSize)
	percentMemory := float64(rssBytes) / float64(totalMemoryKB*1024) * 100

	// Format memory percentage with one decimal place
	memPercentStr := fmt.Sprintf("%.1f", percentMemory)

	// Read command line
	cmdlinePath := filepath.Join("/proc", strconv.Itoa(pid), "cmdline")
	cmdlineContent, err := os.ReadFile(cmdlinePath)
	if err != nil {
		return nil, err
	}

	// Replace null bytes with spaces (like xargs -0)
	cmdline := strings.ReplaceAll(string(cmdlineContent), "\x00", " ")
	cmdline = strings.TrimSpace(cmdline)

	// If cmdline is empty, use process name from stat
	if cmdline == "" {
		// Extract process name from stat file (between parentheses)
		cmdline = statStr[strings.Index(statStr, "(")+1 : lastParen]
	}

	return map[string]string{
		"pid":    strconv.Itoa(pid),
		"ppid":   strconv.Itoa(ppid),
		"cpu":    "0.0", // Placeholder, would need CPU calculation
		"mem":    memPercentStr,
		"cmd":    cmdline,
	}, nil
}

// GetProcessInfo gets detailed process information
func GetProcessInfo(pid, pageSize int, totalMemoryKB uint64) (ProcessInfo, error) {
	var info ProcessInfo
	info.PID = pid

	// Read stat file
	statPath := filepath.Join("/proc", strconv.Itoa(pid), "stat")
	statContent, err := os.ReadFile(statPath)
	if err != nil {
		return info, err
	}

	statStr := string(statContent)
	lastParen := strings.LastIndex(statStr, ")")
	if lastParen == -1 {
		return info, fmt.Errorf("invalid stat format")
	}

	// Extract process name
	info.Name = statStr[strings.Index(statStr, "(")+1 : lastParen]

	// Parse remaining fields
	remaining := strings.TrimSpace(statStr[lastParen+1:])
	fields := strings.Fields(remaining)
	
	if len(fields) < 24 {
		return info, fmt.Errorf("stat file too short")
	}

	// Parse fields
	info.State = fields[0]
	info.PPID, _ = strconv.Atoi(fields[1])
	
	// Parse CPU times (fields 13 & 14 in 1-indexed, 12 & 13 in 0-indexed)
	utime, _ := strconv.ParseUint(fields[12], 10, 64)
	stime, _ := strconv.ParseUint(fields[13], 10, 64)
	info.UpTime = fmt.Sprintf("%d", utime)
	info.SleepTime = fmt.Sprintf("%d", stime)
	// Memory info (RSS in pages)
	rssPages, _ := strconv.ParseUint(fields[23], 10, 64)
	info.RSS = rssPages * uint64(pageSize) / 1024 // Convert to KB
	info.Memory = float64(info.RSS) / float64(totalMemoryKB) * 100

	// Read command line
	cmdlinePath := filepath.Join("/proc", strconv.Itoa(pid), "cmdline")
	cmdlineContent, err := os.ReadFile(cmdlinePath)
	if err == nil {
		cmdline := strings.ReplaceAll(string(cmdlineContent), "\x00", " ")
		info.Command = strings.TrimSpace(cmdline)
		if info.Command == "" {
			info.Command = info.Name
		}
	} else {
		info.Command = info.Name
	}

	// Read status file for additional info
	statusPath := filepath.Join("/proc", strconv.Itoa(pid), "status")
	if statusContent, err := os.ReadFile(statusPath); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(statusContent)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Uid:") {
				// Parse UID
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					uid, _ := strconv.Atoi(fields[1])
					info.User = GetUserName(uid)
				}
			} else if strings.HasPrefix(line, "VmSize:") {
				// Parse virtual memory size
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					vszKB, _ := strconv.ParseUint(fields[1], 10, 64)
					info.VSZ = vszKB
				}
			}
		}
	}

	// CPU calculation would require sampling over time
	// For now, set to 0.0 like the bash script
	info.CPU = 0.0

	return info, nil
}

// MemoryInfo holds system memory information
type MemoryInfo struct {
	TotalKB uint64 `json:"total_kb"`
	FreeKB  uint64 `json:"free_kb"`
	UsedKB  uint64 `json:"used_kb"`
}

// GetMemoryInfo reads memory information from /proc/meminfo
func GetMemoryInfo() (*MemoryInfo, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info := &MemoryInfo{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			info.TotalKB = value
		case "MemFree:":
			info.FreeKB = value
		case "MemAvailable:":
			// Use available memory if present (Linux 3.14+)
			info.FreeKB = value
		}
	}

	info.UsedKB = info.TotalKB - info.FreeKB
	return info, nil
}

// GetUserName converts UID to username
func GetUserName(uid int) string {
	// Try to read from /etc/passwd
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return strconv.Itoa(uid)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ":")
		if len(fields) >= 3 {
			fileUID, err := strconv.Atoi(fields[2])
			if err == nil && fileUID == uid {
				return fields[0]
			}
		}
	}

	return strconv.Itoa(uid)
}

// FilterProcesses filters processes based on criteria
func FilterProcesses(processes []ProcessInfo, filterFunc func(ProcessInfo) bool) []ProcessInfo {
	var filtered []ProcessInfo
	for _, p := range processes {
		if filterFunc(p) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// SortProcesses sorts processes by a field
func SortProcesses(processes []ProcessInfo, field string, descending bool) []ProcessInfo {
	sorted := make([]ProcessInfo, len(processes))
	copy(sorted, processes)

	switch field {
	case "cpu":
		if descending {
			// Sort by CPU descending
			for i := 0; i < len(sorted); i++ {
				for j := i + 1; j < len(sorted); j++ {
					if sorted[i].CPU < sorted[j].CPU {
						sorted[i], sorted[j] = sorted[j], sorted[i]
					}
				}
			}
		} else {
			// Sort by CPU ascending
			for i := 0; i < len(sorted); i++ {
				for j := i + 1; j < len(sorted); j++ {
					if sorted[i].CPU > sorted[j].CPU {
						sorted[i], sorted[j] = sorted[j], sorted[i]
					}
				}
			}
		}
	case "mem":
		if descending {
			// Sort by memory descending
			for i := 0; i < len(sorted); i++ {
				for j := i + 1; j < len(sorted); j++ {
					if sorted[i].Memory < sorted[j].Memory {
						sorted[i], sorted[j] = sorted[j], sorted[i]
					}
				}
			}
		} else {
			// Sort by memory ascending
			for i := 0; i < len(sorted); i++ {
				for j := i + 1; j < len(sorted); j++ {
					if sorted[i].Memory > sorted[j].Memory {
						sorted[i], sorted[j] = sorted[j], sorted[i]
					}
				}
			}
		}
	case "pid":
		if descending {
			// Sort by PID descending
			for i := 0; i < len(sorted); i++ {
				for j := i + 1; j < len(sorted); j++ {
					if sorted[i].PID < sorted[j].PID {
						sorted[i], sorted[j] = sorted[j], sorted[i]
					}
				}
			}
		} else {
			// Sort by PID ascending
			for i := 0; i < len(sorted); i++ {
				for j := i + 1; j < len(sorted); j++ {
					if sorted[i].PID > sorted[j].PID {
						sorted[i], sorted[j] = sorted[j], sorted[i]
					}
				}
			}
		}
	}

	return sorted
}

// GetTopProcesses returns top N processes by CPU or memory usage
func GetTopProcesses(n int, sortBy string) ([]ProcessInfo, error) {
	processes, err := GetProcessList()
	if err != nil {
		return nil, err
	}

	// Sort processes
	sorted := SortProcesses(processes, sortBy, true)

	// Return top N
	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n], nil
}

// FormatProcessList formats process list as tab-separated string (like bash script)
func FormatProcessList(processes []map[string]string) string {
	var output strings.Builder
	for _, proc := range processes {
		output.WriteString(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n",
			proc["pid"],
			proc["ppid"],
			proc["cpu"],
			proc["mem"],
			proc["cmd"],
		))
	}
	return output.String()
}