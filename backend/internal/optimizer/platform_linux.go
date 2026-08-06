package optimizer

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var linuxOperationMu sync.Mutex

func acquireOperationLock() (func(), error) {
	linuxOperationMu.Lock()
	return linuxOperationMu.Unlock, nil
}

func detectHardware() (Hardware, error) {
	hardware := Hardware{GOARCH: runtime.GOARCH, CPUs: []CPUInfo{}, GPUs: []GPUInfo{}}
	values, osErr := readKeyValueFile("/etc/os-release", 256<<10)
	hardware.OS = OSInfo{
		Caption:      firstNonEmpty(values["PRETTY_NAME"], values["NAME"], "Linux"),
		Version:      values["VERSION_ID"],
		BuildNumber:  kernelRelease(),
		Architecture: runtime.GOARCH,
	}
	cpu, cpuErr := linuxCPUInfo()
	if cpu.Logical > 0 {
		hardware.CPUs = []CPUInfo{cpu}
	}
	hardware.GPUs = linuxGPUs()
	if matches, _ := filepath.Glob("/sys/class/power_supply/BAT*"); len(matches) > 0 {
		hardware.HasBattery = true
	}
	return hardware, errors.Join(osErr, cpuErr)
}

func linuxCapabilities() []Capability {
	_, gameModeErr := exec.LookPath("gamemoderun")
	_, systemctlErr := exec.LookPath("systemctl")
	steamAvailable := len(steamRoots()) > 0
	affinity := affinityAvailable()
	priority := canRaisePriority()
	return []Capability{
		{ID: "persistent-windows-profile", Available: false, Mode: "skipped", Detail: "Windows registry/power/NIC settings are not applicable and will not be emulated"},
		{ID: "feral-gamemode", Available: gameModeErr == nil, Mode: "session", Detail: availableDetail(gameModeErr == nil, "gamemoderun will wrap game launches", "gamemoderun not found; games launch directly")},
		{ID: "process-priority", Available: priority, Mode: "session", Detail: availableDetail(priority, "negative nice is available", "negative nice needs CAP_SYS_NICE; unsupported requests are skipped")},
		{ID: "process-affinity", Available: affinity, Mode: "session", Detail: availableDetail(affinity, "sched_setaffinity is available for explicit masks", "sched_setaffinity is unavailable and will be skipped")},
		{ID: "steam-discovery", Available: steamAvailable, Mode: "read-only", Detail: availableDetail(steamAvailable, "Steam library roots detected", "no supported Steam root detected")},
		{ID: "xdg-startup", Available: true, Mode: "reversible", Detail: "user .desktop entries can be atomically disabled and enabled"},
		{ID: "systemd-services", Available: systemctlErr == nil && isSystemdBooted(), Mode: "read-only", Detail: availableDetail(systemctlErr == nil && isSystemdBooted(), "systemd service inventory is available", "systemd is not the active service manager; command returns an empty report")},
		{ID: "self-update", Available: true, Mode: "opt-in", Detail: "GitHub Release asset is selected by OS/arch and verified against SHA256SUMS.txt"},
	}
}

func linuxCPUInfo() (CPUInfo, error) {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return CPUInfo{Logical: runtime.NumCPU(), Cores: runtime.NumCPU()}, err
	}
	defer file.Close()
	cpu := CPUInfo{Logical: runtime.NumCPU()}
	cores := make(map[string]bool)
	physical, core := "0", ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if core != "" {
				cores[physical+":"+core] = true
			}
			physical, core = "0", ""
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		switch key {
		case "model name", "Hardware":
			if cpu.Name == "" {
				cpu.Name = value
			}
		case "vendor_id":
			if cpu.Manufacturer == "" {
				cpu.Manufacturer = value
			}
		case "physical id":
			physical = value
		case "core id":
			core = value
		}
	}
	if core != "" {
		cores[physical+":"+core] = true
	}
	if len(cores) > 0 {
		cpu.Cores = len(cores)
	} else {
		cpu.Cores = cpu.Logical
	}
	if cpu.Name == "" {
		cpu.Name = runtime.GOARCH + " CPU"
	}
	return cpu, scanner.Err()
}

func linuxGPUs() []GPUInfo {
	devices, _ := filepath.Glob("/sys/class/drm/card[0-9]*/device")
	result := make([]GPUInfo, 0, len(devices))
	for _, device := range devices {
		vendorID := strings.TrimSpace(readText(filepath.Join(device, "vendor"), 64))
		deviceID := strings.TrimSpace(readText(filepath.Join(device, "device"), 64))
		driver := ""
		if target, err := filepath.EvalSymlinks(filepath.Join(device, "driver")); err == nil {
			driver = filepath.Base(target)
		}
		vendor := gpuVendorName(vendorID)
		card := filepath.Base(filepath.Dir(device))
		result = append(result, GPUInfo{Name: fmt.Sprintf("%s (%s:%s)", card, vendorID, deviceID), PNPDeviceID: vendorID + ":" + deviceID, DriverVersion: driver, Vendor: vendor})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func gpuVendorName(id string) string {
	switch strings.ToLower(id) {
	case "0x10de":
		return "NVIDIA"
	case "0x1002", "0x1022":
		return "AMD"
	case "0x8086":
		return "Intel"
	default:
		return firstNonEmpty(id, "Unknown")
	}
}

func readKeyValueFile(path string, limit int64) (map[string]string, error) {
	data, err := readSmallFile(path, limit)
	if err != nil {
		return map[string]string{}, err
	}
	result := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || key == "" {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		result[key] = value
	}
	return result, nil
}

func readText(path string, limit int64) string {
	data, err := readSmallFile(path, limit)
	if err != nil {
		return ""
	}
	return string(data)
}

func kernelRelease() string {
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func currentGovernor() string {
	return strings.TrimSpace(readText("/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor", 128))
}

func isSystemdBooted() bool {
	info, err := os.Stat("/run/systemd/system")
	return err == nil && info.IsDir()
}

func xdgDirectory(envName, fallback string) (string, error) {
	if value := os.Getenv(envName); filepath.IsAbs(value) {
		return filepath.Clean(value), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, fallback), nil
}

func canRaisePriority() bool {
	if os.Geteuid() == 0 {
		return true
	}
	data, err := readSmallFile("/proc/self/status", 1<<20)
	if err != nil {
		return false
	}
	capabilityHex := ""
	for _, line := range strings.Split(string(data), "\n") {
		if key, value, ok := strings.Cut(line, ":"); ok && key == "CapEff" {
			capabilityHex = strings.TrimSpace(value)
			break
		}
	}
	capabilities, err := strconv.ParseUint(capabilityHex, 16, 64)
	return err == nil && capabilities&(uint64(1)<<23) != 0
}

func availableDetail(available bool, yes, no string) string {
	if available {
		return yes
	}
	return no
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
