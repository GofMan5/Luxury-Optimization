package main

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	getProcessAffinityMask = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetProcessAffinityMask")
	setProcessAffinityMask = windows.NewLazySystemDLL("kernel32.dll").NewProc("SetProcessAffinityMask")
)

func processPriorityClass(value string) (uint32, error) {
	switch strings.ToLower(value) {
	case "normal":
		return 0, nil
	case "above-normal":
		return windows.ABOVE_NORMAL_PRIORITY_CLASS, nil
	case "high":
		return windows.HIGH_PRIORITY_CLASS, nil
	default:
		return 0, errors.New("priority поддерживает только normal, above-normal и high")
	}
}

func parseAffinity(value string) (uintptr, error) {
	if value == "" {
		return 0, nil
	}
	if runtime.NumCPU() > strconv.IntSize {
		return 0, errors.New("affinity не поддерживается для системы с несколькими processor groups")
	}
	mask, err := strconv.ParseUint(value, 0, strconv.IntSize)
	if err != nil || mask == 0 {
		return 0, errors.New("affinity должна быть ненулевой CPU mask, например 0xFF")
	}
	if runtime.NumCPU() < strconv.IntSize && mask >= uint64(1)<<runtime.NumCPU() {
		return 0, fmt.Errorf("affinity использует отсутствующий CPU; доступно %d логических процессоров", runtime.NumCPU())
	}
	return uintptr(mask), nil
}

func tuneGameProcess(pid uint32, priority string, affinity uintptr) error {
	priorityClass, err := processPriorityClass(priority)
	if err != nil {
		return err
	}
	if priorityClass == 0 && affinity == 0 {
		return nil
	}
	handle, err := windows.OpenProcess(windows.PROCESS_SET_INFORMATION|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	if priorityClass != 0 {
		if err := windows.SetPriorityClass(handle, priorityClass); err != nil {
			return err
		}
		actual, err := windows.GetPriorityClass(handle)
		if err != nil {
			return fmt.Errorf("priority read-back: %w", err)
		}
		if actual != priorityClass {
			return fmt.Errorf("priority read-back: получено 0x%X вместо 0x%X", actual, priorityClass)
		}
	}
	if affinity != 0 {
		result, _, callErr := setProcessAffinityMask.Call(uintptr(handle), affinity)
		if result == 0 {
			if callErr == windows.ERROR_SUCCESS {
				callErr = errors.New("SetProcessAffinityMask failed")
			}
			return callErr
		}
		var actual, system uintptr
		result, _, callErr = getProcessAffinityMask.Call(uintptr(handle), uintptr(unsafe.Pointer(&actual)), uintptr(unsafe.Pointer(&system)))
		if result == 0 {
			if callErr == windows.ERROR_SUCCESS {
				callErr = errors.New("GetProcessAffinityMask failed")
			}
			return callErr
		}
		if actual != affinity {
			return fmt.Errorf("affinity read-back: получено 0x%X вместо 0x%X", actual, affinity)
		}
	}
	return nil
}
