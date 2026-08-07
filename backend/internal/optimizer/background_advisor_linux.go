package optimizer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func backgroundAdvisorAvailable() bool {
	if _, err := linuxTotalCPU(); err != nil {
		return false
	}
	_, err := os.Stat("/proc/self/stat")
	return err == nil
}

func readBackgroundSnapshot(ctx context.Context) (backgroundSnapshot, error) {
	totalCPU, err := linuxTotalCPU()
	if err != nil {
		return backgroundSnapshot{}, err
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return backgroundSnapshot{}, err
	}
	if len(entries) > 32768 {
		return backgroundSnapshot{}, errors.New("/proc contains too many entries")
	}
	snapshot := backgroundSnapshot{CapturedAt: time.Now(), Counters: make(map[uint32]backgroundCounter), TotalCPU: totalCPU}
	ioUnavailable := 0
	for index, entry := range entries {
		if index%64 == 0 {
			if err := ctx.Err(); err != nil {
				return backgroundSnapshot{}, err
			}
		}
		if !entry.IsDir() {
			continue
		}
		pidValue, err := strconv.ParseUint(entry.Name(), 10, 32)
		if err != nil || pidValue == 0 {
			continue
		}
		pid := uint32(pidValue)
		counter, err := linuxProcessCounter(pid)
		if err != nil {
			snapshot.Skipped++
			continue
		}
		readBytes, writeBytes, err := linuxProcessIO(pid)
		if err != nil {
			ioUnavailable++
		} else {
			counter.ReadBytes, counter.WriteBytes = readBytes, writeBytes
		}
		snapshot.Counters[pid] = counter
	}
	if ioUnavailable > 0 {
		snapshot.Warnings = append(snapshot.Warnings, fmt.Sprintf("I/O counters unavailable for %d processes", ioUnavailable))
	}
	return snapshot, nil
}

func linuxTotalCPU() (uint64, error) {
	data, err := readSmallFile("/proc/stat", 1<<20)
	if err != nil {
		return 0, err
	}
	line, _, _ := strings.Cut(string(data), "\n")
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, errors.New("/proc/stat does not contain aggregate CPU counters")
	}
	var total uint64
	for _, field := range fields[1:min(len(fields), 9)] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil || total > math.MaxUint64-value {
			return 0, errors.New("/proc/stat contains invalid CPU counters")
		}
		total += value
	}
	return total, nil
}

func linuxProcessCounter(pid uint32) (backgroundCounter, error) {
	data, err := readSmallFile(filepath.Join("/proc", strconv.FormatUint(uint64(pid), 10), "stat"), 64<<10)
	if err != nil {
		return backgroundCounter{}, err
	}
	line := strings.TrimSpace(string(data))
	open, close := strings.IndexByte(line, '('), strings.LastIndexByte(line, ')')
	if open < 1 || close <= open || close+1 >= len(line) {
		return backgroundCounter{}, errors.New("invalid /proc process stat")
	}
	fields := strings.Fields(line[close+1:])
	if len(fields) <= 21 {
		return backgroundCounter{}, errors.New("short /proc process stat")
	}
	user, err1 := strconv.ParseUint(fields[11], 10, 64)
	kernel, err2 := strconv.ParseUint(fields[12], 10, 64)
	created, err3 := strconv.ParseUint(fields[19], 10, 64)
	rssPages, err4 := strconv.ParseInt(fields[21], 10, 64)
	threads, err5 := strconv.ParseUint(fields[17], 10, 32)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || user > math.MaxUint64-kernel {
		return backgroundCounter{}, errors.New("invalid /proc process counters")
	}
	workingSet := uint64(0)
	if rssPages > 0 && uint64(rssPages) <= math.MaxUint64/uint64(os.Getpagesize()) {
		workingSet = uint64(rssPages) * uint64(os.Getpagesize())
	}
	return backgroundCounter{PID: pid, Created: created, Name: line[open+1 : close], CPU: user + kernel, WorkingSet: workingSet, Threads: uint32(threads)}, nil
}

func linuxProcessIO(pid uint32) (uint64, uint64, error) {
	data, err := readSmallFile(filepath.Join("/proc", strconv.FormatUint(uint64(pid), 10), "io"), 64<<10)
	if err != nil {
		return 0, 0, err
	}
	var readBytes, writeBytes uint64
	for _, line := range strings.Split(string(data), "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found || (key != "read_bytes" && key != "write_bytes") {
			continue
		}
		parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return 0, 0, err
		}
		if key == "read_bytes" {
			readBytes = parsed
		} else {
			writeBytes = parsed
		}
	}
	return readBytes, writeBytes, nil
}

func backgroundCPUCapacity(before, after backgroundSnapshot, _ time.Duration) uint64 {
	if after.TotalCPU <= before.TotalCPU {
		return 0
	}
	return after.TotalCPU - before.TotalCPU
}

func backgroundExecutablePath(pid uint32, created uint64) string {
	counter, err := linuxProcessCounter(pid)
	if err != nil || counter.Created != created {
		return ""
	}
	path, err := os.Readlink(filepath.Join("/proc", strconv.FormatUint(uint64(pid), 10), "exe"))
	if err != nil || !filepath.IsAbs(path) {
		return ""
	}
	counter, err = linuxProcessCounter(pid)
	if err != nil || counter.Created != created {
		return ""
	}
	return path
}

func backgroundProcessServiceNames(pid uint32, created uint64) []string {
	counter, err := linuxProcessCounter(pid)
	if err != nil || counter.Created != created {
		return nil
	}
	data, err := readSmallFile(filepath.Join("/proc", strconv.FormatUint(uint64(pid), 10), "cgroup"), 64<<10)
	if err != nil {
		return nil
	}
	counter, err = linuxProcessCounter(pid)
	if err != nil || counter.Created != created {
		return nil
	}
	seen := make(map[string]bool)
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}
		for _, part := range strings.Split(parts[2], "/") {
			if strings.HasSuffix(part, ".service") && len(part) <= 256 {
				seen[part] = true
			}
		}
	}
	services := make([]string, 0, len(seen))
	for name := range seen {
		services = append(services, name)
	}
	sort.Strings(services)
	return services
}
