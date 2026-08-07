package optimizer

import (
	"fmt"
	"testing"
	"time"
)

func TestBackgroundAdvisorRanksAndCorrelatesWithoutUnsafeAdvice(t *testing.T) {
	before := backgroundSnapshot{Counters: map[uint32]backgroundCounter{
		10: {PID: 10, Created: 1, Name: "Discord.exe", CPU: 100, WorkingSet: 300 << 20, ReadBytes: 1000, WriteBytes: 2000, Threads: 8},
		20: {PID: 20, Created: 2, Name: "svchost.exe", CPU: 100, WorkingSet: 100 << 20, Threads: 12},
		30: {PID: 30, Created: 3, Name: "idle.exe", CPU: 100, WorkingSet: 20 << 20, Threads: 2},
		40: {PID: 40, Created: 4, Name: "VendorAgent.exe", CPU: 100, WorkingSet: 80 << 20, Threads: 6},
	}}
	after := backgroundSnapshot{Counters: map[uint32]backgroundCounter{
		10: {PID: 10, Created: 1, Name: "Discord.exe", CPU: 160, WorkingSet: 320 << 20, ReadBytes: 1000 + 3<<20, WriteBytes: 2000, Threads: 9},
		20: {PID: 20, Created: 2, Name: "svchost.exe", CPU: 120, WorkingSet: 110 << 20, Threads: 12},
		30: {PID: 30, Created: 3, Name: "idle.exe", CPU: 101, WorkingSet: 20 << 20, Threads: 2},
		40: {PID: 40, Created: 4, Name: "VendorAgent.exe", CPU: 120, WorkingSet: 82 << 20, Threads: 6},
	}}
	processes, measured, _, _, _ := compareBackgroundSnapshots(before, after, 1000, time.Second, BackgroundThresholds{MediumCPUPercent: 1, HighCPUPercent: 5, MediumIOMBPS: 2, HighIOMBPS: 10})
	if measured != 4 || len(processes) != 4 || processes[0].PID != 10 || processes[0].Impact != "high" {
		t.Fatalf("unexpected ranking: measured=%d processes=%+v", measured, processes)
	}
	processes[0].Executable = `C:\Users\test\AppData\Local\Discord\Discord.exe`
	for index := range processes {
		if processes[index].PID == 40 {
			processes[index].Executable = `C:\Program Files\Vendor\VendorAgent.exe`
		}
	}
	startup := StartupReport{Entries: []StartupEntry{{Scope: "HKCU", Name: "Discord", Command: `"C:\Users\test\AppData\Local\Discord\Discord.exe" --start-minimized`, State: "present"}}}
	services := ServicesReport{Services: []ServiceEntry{{Name: "BFE", Display: "Base Filtering Engine", ProcessID: 20, System: true, Critical: true, Manageable: false}, {Name: "VendorAgent", Display: "Vendor Update Agent", ProcessID: 40, Manageable: true}}}
	if correlated := decorateBackgroundProcesses(processes, startup, services); correlated != 3 {
		t.Fatalf("unexpected correlation count: %d", correlated)
	}
	byPID := map[uint32]BackgroundProcess{}
	for _, process := range processes {
		byPID[process.PID] = process
	}
	if byPID[10].Advice != "review_startup" || len(byPID[10].Startup) != 1 {
		t.Fatalf("startup advice missing: %+v", byPID[10])
	}
	if byPID[20].Advice != "protected_service" || len(byPID[20].Services) != 1 {
		t.Fatalf("protected service advice missing: %+v", byPID[20])
	}
	if byPID[30].Advice != "observe" {
		t.Fatalf("idle process received action advice: %+v", byPID[30])
	}
	if byPID[40].Advice != "review_service" || len(byPID[40].Services) != 1 {
		t.Fatalf("third-party service advice missing: %+v", byPID[40])
	}
}

func TestBackgroundAdvisorRejectsPIDReuse(t *testing.T) {
	before := backgroundSnapshot{Counters: map[uint32]backgroundCounter{10: {PID: 10, Created: 1, Name: "old.exe", CPU: 10}}}
	after := backgroundSnapshot{Counters: map[uint32]backgroundCounter{10: {PID: 10, Created: 2, Name: "new.exe", CPU: 20}}}
	processes, measured, _, _, _ := compareBackgroundSnapshots(before, after, 100, time.Second, BackgroundThresholds{MediumCPUPercent: 1, HighCPUPercent: 5, MediumIOMBPS: 2, HighIOMBPS: 10})
	if measured != 0 || len(processes) != 0 {
		t.Fatalf("PID reuse was measured: %+v", processes)
	}
}

func TestBackgroundThresholdsScaleForHighCoreCounts(t *testing.T) {
	standard := backgroundThresholdsFor(16)
	highCore := backgroundThresholdsFor(128)
	if standard.MediumCPUPercent != 1 || standard.HighCPUPercent != 5 || highCore.MediumCPUPercent >= 1 || highCore.HighCPUPercent >= 5 {
		t.Fatalf("unexpected topology thresholds: standard=%+v highCore=%+v", standard, highCore)
	}
}

func TestBackgroundAdvisorCapsProcessesAndWarnings(t *testing.T) {
	before := backgroundSnapshot{Counters: make(map[uint32]backgroundCounter)}
	after := backgroundSnapshot{Counters: make(map[uint32]backgroundCounter)}
	for pid := uint32(1); pid <= 70; pid++ {
		before.Counters[pid] = backgroundCounter{PID: pid, Created: uint64(pid), Name: fmt.Sprintf("process-%d", pid), CPU: 1}
		after.Counters[pid] = backgroundCounter{PID: pid, Created: uint64(pid), Name: fmt.Sprintf("process-%d", pid), CPU: uint64(pid + 1)}
	}
	processes, measured, _, _, _ := compareBackgroundSnapshots(before, after, 10_000, time.Second, BackgroundThresholds{MediumCPUPercent: 1, HighCPUPercent: 5, MediumIOMBPS: 2, HighIOMBPS: 10})
	if measured != 70 || len(processes) != maxBackgroundProcesses {
		t.Fatalf("unexpected process cap: measured=%d returned=%d", measured, len(processes))
	}
	warnings := make([]string, 40)
	for index := range warnings {
		warnings[index] = fmt.Sprintf("warning-%d", index)
	}
	if actual := appendUniqueStrings(nil, warnings...); len(actual) != maxBackgroundWarnings {
		t.Fatalf("unexpected warning cap: %d", len(actual))
	}
}

func TestBackgroundStartupMatchingUsesExecutableBoundaries(t *testing.T) {
	if !backgroundStartupMatches(`/usr/bin/discord --start-minimized`, "/usr/bin/discord") {
		t.Fatal("exact executable did not match")
	}
	if backgroundStartupMatches(`/opt/notdiscord-helper`, "/opt/notdiscord") || backgroundStartupMatches(`discord --start-minimized`, "") {
		t.Fatal("substring-only or unresolved executable matched")
	}
}
