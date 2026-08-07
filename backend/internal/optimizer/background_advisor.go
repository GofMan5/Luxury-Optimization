package optimizer

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	defaultBackgroundSampleMS = 1500
	minBackgroundSampleMS     = 500
	maxBackgroundSampleMS     = 5000
	maxBackgroundProcesses    = 64
	maxBackgroundLinks        = 8
	maxBackgroundInventory    = 4096
	maxBackgroundWarnings     = 32
)

type BackgroundThresholds struct {
	MediumCPUPercent float64 `json:"medium_cpu_percent"`
	HighCPUPercent   float64 `json:"high_cpu_percent"`
	MediumIOMBPS     float64 `json:"medium_io_mb_s"`
	HighIOMBPS       float64 `json:"high_io_mb_s"`
}

type BackgroundStartupLink struct {
	Scope string `json:"scope"`
	Name  string `json:"name"`
	State string `json:"state"`
}

type BackgroundServiceLink struct {
	Name       string `json:"name"`
	Display    string `json:"display_name"`
	System     bool   `json:"system"`
	Critical   bool   `json:"critical"`
	Manageable bool   `json:"manageable"`
}

type BackgroundProcess struct {
	PID          uint32                  `json:"pid"`
	Name         string                  `json:"name"`
	Executable   string                  `json:"executable,omitempty"`
	CPUPercent   float64                 `json:"cpu_percent"`
	WorkingSetMB float64                 `json:"working_set_mb"`
	ReadMBPS     float64                 `json:"read_mb_s"`
	WriteMBPS    float64                 `json:"write_mb_s"`
	Threads      uint32                  `json:"threads"`
	Impact       string                  `json:"impact"`
	Advice       string                  `json:"advice"`
	Startup      []BackgroundStartupLink `json:"startup"`
	Services     []BackgroundServiceLink `json:"services"`
	created      uint64
	serviceNames []string
}

type BackgroundReport struct {
	GeneratedAt         time.Time            `json:"generated_at"`
	SampleMS            int64                `json:"sample_ms"`
	LogicalProcessors   int                  `json:"logical_processors"`
	ObservedProcesses   int                  `json:"observed_processes"`
	MeasuredProcesses   int                  `json:"measured_processes"`
	CorrelatedProcesses int                  `json:"correlated_processes"`
	SkippedProcesses    int                  `json:"skipped_processes"`
	ObservedCPUPercent  float64              `json:"observed_cpu_percent"`
	ReadMBPS            float64              `json:"read_mb_s"`
	WriteMBPS           float64              `json:"write_mb_s"`
	Thresholds          BackgroundThresholds `json:"thresholds"`
	Processes           []BackgroundProcess  `json:"processes"`
	Warnings            []string             `json:"warnings,omitempty"`
}

type backgroundCounter struct {
	PID        uint32
	Created    uint64
	Name       string
	CPU        uint64
	WorkingSet uint64
	ReadBytes  uint64
	WriteBytes uint64
	Threads    uint32
}

type backgroundSnapshot struct {
	CapturedAt time.Time
	Counters   map[uint32]backgroundCounter
	TotalCPU   uint64
	Skipped    int
	Warnings   []string
}

func (s *Service) BackgroundAdvisor(ctx context.Context, sampleMS int) (BackgroundReport, error) {
	if sampleMS == 0 {
		sampleMS = defaultBackgroundSampleMS
	}
	if sampleMS < minBackgroundSampleMS || sampleMS > maxBackgroundSampleMS {
		return BackgroundReport{}, errors.New("sample_ms must be between 500 and 5000")
	}
	return collectBackgroundReport(ctx, time.Duration(sampleMS)*time.Millisecond)
}

func collectBackgroundReport(ctx context.Context, requested time.Duration) (BackgroundReport, error) {
	before, err := readBackgroundSnapshot(ctx)
	if err != nil {
		return BackgroundReport{}, err
	}
	timer := time.NewTimer(requested)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return BackgroundReport{}, ctx.Err()
	case <-timer.C:
	}
	after, err := readBackgroundSnapshot(ctx)
	if err != nil {
		return BackgroundReport{}, err
	}
	elapsed := after.CapturedAt.Sub(before.CapturedAt)
	if elapsed <= 0 {
		return BackgroundReport{}, errors.New("process snapshot clock did not advance")
	}
	capacity := backgroundCPUCapacity(before, after, elapsed)
	thresholds := backgroundThresholdsFor(runtime.NumCPU())
	processes, measured, cpu, read, write := compareBackgroundSnapshots(before, after, capacity, elapsed, thresholds)

	startup := listStartupEntries()
	services, serviceErr := listServices("running", "")
	warnings := appendUniqueStrings(nil, before.Warnings...)
	warnings = appendUniqueStrings(warnings, after.Warnings...)
	if len(startup.Entries) > maxBackgroundInventory {
		startup.Entries = startup.Entries[:maxBackgroundInventory]
		warnings = appendUniqueStrings(warnings, "startup correlation inventory truncated at 4096 entries")
	}
	if serviceErr == nil && len(services.Services) > maxBackgroundInventory {
		services.Services = services.Services[:maxBackgroundInventory]
		warnings = appendUniqueStrings(warnings, "service correlation inventory truncated at 4096 entries")
	}
	for _, warning := range startup.Warnings {
		warnings = appendUniqueStrings(warnings, "startup: "+warning)
	}
	if serviceErr != nil {
		warnings = appendUniqueStrings(warnings, "service inventory: "+serviceErr.Error())
		services = ServicesReport{Services: []ServiceEntry{}}
	}
	for index := range processes {
		processes[index].Executable = cleanBackgroundText(backgroundExecutablePath(processes[index].PID, processes[index].created), 1024)
		processes[index].serviceNames = backgroundProcessServiceNames(processes[index].PID, processes[index].created)
	}
	correlated := decorateBackgroundProcesses(processes, startup, services)
	return BackgroundReport{
		GeneratedAt:         time.Now().UTC(),
		SampleMS:            elapsed.Milliseconds(),
		LogicalProcessors:   runtime.NumCPU(),
		ObservedProcesses:   len(after.Counters),
		MeasuredProcesses:   measured,
		CorrelatedProcesses: correlated,
		SkippedProcesses:    max(before.Skipped, after.Skipped),
		ObservedCPUPercent:  roundBackgroundMetric(math.Min(cpu, 100)),
		ReadMBPS:            roundBackgroundMetric(read),
		WriteMBPS:           roundBackgroundMetric(write),
		Thresholds:          thresholds,
		Processes:           processes,
		Warnings:            warnings,
	}, nil
}

func compareBackgroundSnapshots(before, after backgroundSnapshot, capacity uint64, elapsed time.Duration, thresholds BackgroundThresholds) ([]BackgroundProcess, int, float64, float64, float64) {
	if capacity == 0 || elapsed <= 0 {
		return []BackgroundProcess{}, 0, 0, 0, 0
	}
	seconds := elapsed.Seconds()
	processes := make([]BackgroundProcess, 0, len(after.Counters))
	measured := 0
	totalCPU, totalRead, totalWrite := 0.0, 0.0, 0.0
	for pid, current := range after.Counters {
		previous, found := before.Counters[pid]
		if !found || previous.Created != current.Created || current.CPU < previous.CPU || backgroundOwnProcess(pid, current.Name) {
			continue
		}
		measured++
		cpuPercent := float64(current.CPU-previous.CPU) / float64(capacity) * 100
		readRate := backgroundByteRate(previous.ReadBytes, current.ReadBytes, seconds)
		writeRate := backgroundByteRate(previous.WriteBytes, current.WriteBytes, seconds)
		item := BackgroundProcess{
			PID:          pid,
			Name:         cleanBackgroundText(current.Name, 260),
			CPUPercent:   roundBackgroundMetric(math.Max(0, math.Min(cpuPercent, 100))),
			WorkingSetMB: roundBackgroundMetric(float64(current.WorkingSet) / (1024 * 1024)),
			ReadMBPS:     roundBackgroundMetric(readRate),
			WriteMBPS:    roundBackgroundMetric(writeRate),
			Threads:      current.Threads,
			Startup:      []BackgroundStartupLink{},
			Services:     []BackgroundServiceLink{},
			created:      current.Created,
		}
		item.Impact = backgroundImpact(item, thresholds)
		item.Advice = "observe"
		processes = append(processes, item)
		totalCPU += item.CPUPercent
		totalRead += readRate
		totalWrite += writeRate
	}
	sort.SliceStable(processes, func(i, j int) bool {
		left, right := backgroundActivityScore(processes[i]), backgroundActivityScore(processes[j])
		if left != right {
			return left > right
		}
		if processes[i].WorkingSetMB != processes[j].WorkingSetMB {
			return processes[i].WorkingSetMB > processes[j].WorkingSetMB
		}
		return processes[i].PID < processes[j].PID
	})
	if len(processes) > maxBackgroundProcesses {
		processes = processes[:maxBackgroundProcesses]
	}
	return processes, measured, totalCPU, totalRead, totalWrite
}

func decorateBackgroundProcesses(processes []BackgroundProcess, startup StartupReport, services ServicesReport) int {
	serviceByPID := make(map[uint32][]ServiceEntry)
	serviceByName := make(map[string]ServiceEntry)
	for _, service := range services.Services {
		if service.ProcessID != 0 {
			serviceByPID[service.ProcessID] = append(serviceByPID[service.ProcessID], service)
		}
		serviceByName[strings.ToLower(service.Name)] = service
	}
	correlated := 0
	for index := range processes {
		process := &processes[index]
		for _, entry := range startup.Entries {
			if !backgroundStartupMatches(entry.Command, process.Executable) {
				continue
			}
			link := BackgroundStartupLink{Scope: cleanBackgroundText(entry.Scope, 64), Name: cleanBackgroundText(entry.Name, 256), State: cleanBackgroundText(entry.State, 64)}
			if !containsBackgroundStartup(process.Startup, link) {
				process.Startup = append(process.Startup, link)
			}
		}
		matchedServices := append([]ServiceEntry(nil), serviceByPID[process.PID]...)
		for _, name := range process.serviceNames {
			if service, found := serviceByName[strings.ToLower(name)]; found {
				matchedServices = append(matchedServices, service)
			}
		}
		allServiceLinks := make([]BackgroundServiceLink, 0, len(matchedServices))
		for _, service := range matchedServices {
			link := BackgroundServiceLink{Name: cleanBackgroundText(service.Name, 256), Display: cleanBackgroundText(service.Display, 256), System: service.System, Critical: service.Critical, Manageable: service.Manageable}
			if !containsBackgroundService(allServiceLinks, link.Name) {
				allServiceLinks = append(allServiceLinks, link)
			}
		}
		sort.Slice(allServiceLinks, func(i, j int) bool {
			return strings.ToLower(allServiceLinks[i].Name) < strings.ToLower(allServiceLinks[j].Name)
		})
		process.Advice = backgroundAdvice(*process, allServiceLinks)
		if len(process.Startup) > maxBackgroundLinks {
			process.Startup = process.Startup[:maxBackgroundLinks]
		}
		if len(allServiceLinks) > maxBackgroundLinks {
			allServiceLinks = allServiceLinks[:maxBackgroundLinks]
		}
		process.Services = allServiceLinks
		process.serviceNames = nil
		if len(process.Startup) > 0 || len(process.Services) > 0 {
			correlated++
		}
	}
	return correlated
}

func backgroundImpact(process BackgroundProcess, thresholds BackgroundThresholds) string {
	ioRate := process.ReadMBPS + process.WriteMBPS
	if process.CPUPercent >= thresholds.HighCPUPercent || ioRate >= thresholds.HighIOMBPS {
		return "high"
	}
	if process.CPUPercent >= thresholds.MediumCPUPercent || ioRate >= thresholds.MediumIOMBPS {
		return "medium"
	}
	return "low"
}

func backgroundThresholdsFor(logicalProcessors int) BackgroundThresholds {
	if logicalProcessors < 1 {
		logicalProcessors = 1
	}
	logical := float64(logicalProcessors)
	return BackgroundThresholds{
		MediumCPUPercent: roundBackgroundMetric(math.Max(0.1, math.Min(1, 25/logical))),
		HighCPUPercent:   roundBackgroundMetric(math.Max(0.5, math.Min(5, 100/logical))),
		MediumIOMBPS:     2,
		HighIOMBPS:       10,
	}
}

func backgroundAdvice(process BackgroundProcess, services []BackgroundServiceLink) string {
	for _, service := range services {
		if service.Critical || service.System || !service.Manageable {
			return "protected_service"
		}
	}
	if process.Impact == "low" {
		return "observe"
	}
	for _, entry := range process.Startup {
		if entry.State == "present" && (entry.Scope == "HKCU" || strings.EqualFold(entry.Scope, "user")) {
			return "review_startup"
		}
	}
	for _, service := range services {
		if process.Executable != "" && service.Manageable && !service.System && !service.Critical {
			return "review_service"
		}
	}
	return "observe"
}

func backgroundStartupMatches(command, executable string) bool {
	command = normalizeBackgroundCommand(command)
	if command == "" || executable == "" {
		return false
	}
	path := normalizeBackgroundCommand(filepath.Clean(executable))
	return len(path) >= 4 && backgroundCommandHasToken(command, path)
}

func backgroundCommandHasToken(command, token string) bool {
	for offset := 0; ; {
		index := strings.Index(command[offset:], token)
		if index < 0 {
			return false
		}
		index += offset
		end := index + len(token)
		beforeOK := index == 0 || backgroundCommandBoundary(command[index-1])
		afterOK := end == len(command) || backgroundCommandBoundary(command[end])
		if beforeOK && afterOK {
			return true
		}
		offset = index + 1
	}
}

func backgroundCommandBoundary(value byte) bool {
	return value == ' ' || value == '\t' || value == '"' || value == '\'' || value == '/' || value == '\\' || value == '=' || value == ',' || value == ';'
}

func normalizeBackgroundCommand(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), `\`, "/")
	if runtime.GOOS == "windows" {
		value = strings.ToLower(value)
	}
	return value
}

func backgroundActivityScore(process BackgroundProcess) float64 {
	return process.CPUPercent*10 + (process.ReadMBPS+process.WriteMBPS)*2
}

func backgroundByteRate(before, after uint64, seconds float64) float64 {
	if after < before || seconds <= 0 {
		return 0
	}
	return float64(after-before) / seconds / (1024 * 1024)
}

func backgroundOwnProcess(pid uint32, name string) bool {
	if pid == 0 || int(pid) == os.Getpid() {
		return true
	}
	value := strings.ToLower(name)
	return strings.Contains(value, "luxury-optimization")
}

func cleanBackgroundText(value string, limit int) string {
	value = strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, value))
	if len(value) > limit {
		value = value[:limit]
		for !utf8.ValidString(value) {
			value = value[:len(value)-1]
		}
	}
	return value
}

func roundBackgroundMetric(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func containsBackgroundStartup(links []BackgroundStartupLink, candidate BackgroundStartupLink) bool {
	for _, link := range links {
		if link.Scope == candidate.Scope && link.Name == candidate.Name && link.State == candidate.State {
			return true
		}
	}
	return false
}

func containsBackgroundService(links []BackgroundServiceLink, name string) bool {
	for _, link := range links {
		if strings.EqualFold(link.Name, name) {
			return true
		}
	}
	return false
}

func appendUniqueStrings(values []string, additions ...string) []string {
	for _, addition := range additions {
		if len(values) >= maxBackgroundWarnings {
			break
		}
		addition = cleanBackgroundText(addition, 1000)
		if addition == "" || slices.Contains(values, addition) {
			continue
		}
		values = append(values, addition)
	}
	return values
}
