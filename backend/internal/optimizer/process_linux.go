package optimizer

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func processPriorityClass(value string) (int, error) {
	switch strings.ToLower(value) {
	case "normal":
		return 0, nil
	case "above-normal":
		return -5, nil
	case "high":
		return -10, nil
	default:
		return 0, errors.New("priority поддерживает только normal, above-normal и high")
	}
}

func parseAffinity(value string) (uintptr, error) {
	if value == "" {
		return 0, nil
	}
	mask, err := strconv.ParseUint(value, 0, strconv.IntSize)
	if err != nil || mask == 0 {
		return 0, errors.New("affinity должна быть ненулевой CPU mask, например 0xFF")
	}
	var allowed unix.CPUSet
	if err := unix.SchedGetaffinity(0, &allowed); err != nil {
		return 0, fmt.Errorf("не удалось прочитать доступный CPU set: %w", err)
	}
	for cpu := 0; cpu < strconv.IntSize; cpu++ {
		if mask&(uint64(1)<<cpu) != 0 && !allowed.IsSet(cpu) {
			return 0, fmt.Errorf("affinity использует CPU %d вне доступного cpuset", cpu)
		}
	}
	return uintptr(mask), nil
}

func tuneGameProcess(pid int, priority string, affinity uintptr) ([]string, error) {
	requestedNice, err := processPriorityClass(priority)
	if err != nil {
		return nil, err
	}
	var warnings []string
	if requestedNice < 0 {
		if !canRaisePriority() {
			warnings = append(warnings, "priority пропущен: нужен CAP_SYS_NICE")
		} else if err := unix.Setpriority(unix.PRIO_PROCESS, pid, requestedNice); err != nil {
			return warnings, fmt.Errorf("setpriority: %w", err)
		} else if actual, err := unix.Getpriority(unix.PRIO_PROCESS, pid); err != nil {
			return warnings, fmt.Errorf("priority read-back: %w", err)
		} else if actual != requestedNice {
			return warnings, fmt.Errorf("priority read-back: получено %d вместо %d", actual, requestedNice)
		}
	}
	if affinity != 0 && !affinityAvailable() {
		warnings = append(warnings, "affinity пропущен: sched_getaffinity недоступен")
	} else if affinity != 0 {
		var requested unix.CPUSet
		for cpu := 0; cpu < strconv.IntSize; cpu++ {
			if affinity&(uintptr(1)<<cpu) != 0 {
				requested.Set(cpu)
			}
		}
		if err := unix.SchedSetaffinity(pid, &requested); err != nil {
			return warnings, fmt.Errorf("sched_setaffinity: %w", err)
		}
		var actual unix.CPUSet
		if err := unix.SchedGetaffinity(pid, &actual); err != nil {
			return warnings, fmt.Errorf("affinity read-back: %w", err)
		}
		for cpu := 0; cpu < 1024; cpu++ {
			requestedCPU := cpu < strconv.IntSize && affinity&(uintptr(1)<<cpu) != 0
			if actual.IsSet(cpu) != requestedCPU {
				return warnings, errors.New("affinity read-back не совпал с requested mask")
			}
		}
	}
	return warnings, nil
}

func affinityAvailable() bool {
	var current unix.CPUSet
	return unix.SchedGetaffinity(0, &current) == nil
}
