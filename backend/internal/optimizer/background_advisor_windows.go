package optimizer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const maxBackgroundSnapshotBytes = 32 << 20

func readBackgroundSnapshot(ctx context.Context) (backgroundSnapshot, error) {
	size := uint32(1 << 20)
	var buffer []byte
	for {
		if size > maxBackgroundSnapshotBytes {
			return backgroundSnapshot{}, errors.New("process snapshot exceeds the size limit")
		}
		buffer = make([]byte, size)
		var needed uint32
		err := windows.NtQuerySystemInformation(windows.SystemProcessInformation, unsafe.Pointer(&buffer[0]), size, &needed)
		if err == nil {
			break
		}
		if err != windows.STATUS_INFO_LENGTH_MISMATCH {
			return backgroundSnapshot{}, fmt.Errorf("process snapshot: %w", err)
		}
		if needed > size {
			size = needed
		} else {
			size *= 2
		}
	}

	snapshot := backgroundSnapshot{CapturedAt: time.Now(), Counters: make(map[uint32]backgroundCounter)}
	headerSize := uint64(unsafe.Sizeof(windows.SYSTEM_PROCESS_INFORMATION{}))
	base := uintptr(unsafe.Pointer(&buffer[0]))
	limit := base + uintptr(len(buffer))
	var offset uint64
	for {
		if err := ctx.Err(); err != nil {
			return backgroundSnapshot{}, err
		}
		if offset+headerSize > uint64(len(buffer)) {
			return backgroundSnapshot{}, errors.New("process snapshot contains an invalid offset")
		}
		info := (*windows.SYSTEM_PROCESS_INFORMATION)(unsafe.Pointer(&buffer[offset]))
		if info.UniqueProcessID != 0 && info.UniqueProcessID <= uintptr(^uint32(0)) && info.CreateTime > 0 && info.UserTime >= 0 && info.KernelTime >= 0 {
			pid := uint32(info.UniqueProcessID)
			name := windowsProcessName(info.ImageName, base, limit)
			if name != "" {
				user, kernel := nonNegativeInt64(info.UserTime), nonNegativeInt64(info.KernelTime)
				if user > math.MaxUint64-kernel {
					snapshot.Skipped++
				} else {
					snapshot.Counters[pid] = backgroundCounter{PID: pid, Created: nonNegativeInt64(info.CreateTime), Name: name, CPU: user + kernel, WorkingSet: uint64(info.WorkingSetSize), ReadBytes: nonNegativeInt64(info.ReadTransferCount), WriteBytes: nonNegativeInt64(info.WriteTransferCount), Threads: info.NumberOfThreads}
				}
			}
		}
		if info.NextEntryOffset == 0 {
			break
		}
		if uint64(info.NextEntryOffset) < headerSize || offset+uint64(info.NextEntryOffset) <= offset {
			return backgroundSnapshot{}, errors.New("process snapshot contains an invalid next entry")
		}
		offset += uint64(info.NextEntryOffset)
	}
	runtime.KeepAlive(buffer)
	return snapshot, nil
}

func windowsProcessName(value windows.NTUnicodeString, base, limit uintptr) string {
	if value.Buffer == nil || value.Length == 0 || value.Length%2 != 0 {
		return ""
	}
	start := uintptr(unsafe.Pointer(value.Buffer))
	length := uintptr(value.Length)
	if start < base || start+length < start || start+length > limit {
		return ""
	}
	return windows.UTF16ToString(unsafe.Slice(value.Buffer, int(value.Length/2)))
}

func backgroundCPUCapacity(_, _ backgroundSnapshot, elapsed time.Duration) uint64 {
	logical := runtime.NumCPU()
	if logical < 1 || elapsed <= 0 {
		return 0
	}
	ticks := nonNegativeInt64(int64(elapsed / (100 * time.Nanosecond)))
	logicalCount := nonNegativeInt(logical)
	if ticks == 0 || logicalCount == 0 || ticks > math.MaxUint64/logicalCount {
		return 0
	}
	return ticks * logicalCount
}

func backgroundExecutablePath(pid uint32, created uint64) string {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(handle)
	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &creation, &exit, &kernel, &user); err != nil || filetimeValue(creation) != created {
		return ""
	}
	const imagePathBufferSize uint32 = 32768
	buffer := make([]uint16, int(imagePathBufferSize))
	size := imagePathBufferSize
	if err := windows.QueryFullProcessImageName(handle, 0, &buffer[0], &size); err != nil || size == 0 || size > imagePathBufferSize {
		return ""
	}
	return windows.UTF16ToString(buffer[:size])
}

func backgroundProcessServiceNames(uint32, uint64) []string { return nil }

func nonNegativeInt64(value int64) uint64 {
	if value < 0 {
		return 0
	}
	return uint64(value)
}

func nonNegativeInt(value int) uint64 {
	if value < 0 {
		return 0
	}
	return uint64(value)
}

func filetimeValue(value windows.Filetime) uint64 {
	return uint64(value.HighDateTime)<<32 | uint64(value.LowDateTime)
}
