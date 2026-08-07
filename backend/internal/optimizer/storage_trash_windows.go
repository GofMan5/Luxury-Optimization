package optimizer

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	fileOperationDelete   = 3
	fileOperationSilent   = 0x0004
	fileOperationNoPrompt = 0x0010
	fileOperationUndo     = 0x0040
	fileOperationNoUI     = 0x0400
)

var shellFileOperation = windows.NewLazySystemDLL("shell32.dll").NewProc("SHFileOperationW")

type shellFileOperationData struct {
	window        uintptr
	operation     uint32
	from          *uint16
	to            *uint16
	flags         uint16
	aborted       int32
	nameMappings  uintptr
	progressTitle *uint16
}

func moveStorageTargetToTrash(path string) error {
	from, err := windows.UTF16FromString(path)
	if err != nil {
		return err
	}
	from = append(from, 0)
	operation := shellFileOperationData{
		operation: fileOperationDelete,
		from:      &from[0],
		flags:     fileOperationSilent | fileOperationNoPrompt | fileOperationUndo | fileOperationNoUI,
	}
	result, _, _ := shellFileOperation.Call(uintptr(unsafe.Pointer(&operation)))
	if result != 0 {
		return fmt.Errorf("SHFileOperationW result %d", result)
	}
	if operation.aborted != 0 {
		return errors.New("recycle bin operation was aborted")
	}
	return nil
}
