package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

type CleanResult struct {
	Files   int
	Dirs    int
	Bytes   int64
	Skipped int
}

func cleanTemporaryFiles(days int) (CleanResult, error) {
	if days < 1 || days > 30 {
		return CleanResult{}, fmt.Errorf("возраст файлов должен быть от 1 до 30 дней")
	}
	if isAdministrator() {
		return CleanResult{}, errors.New("очистка запрещена в elevated-процессе; запустите оптимизатор без прав администратора")
	}
	userTemp, err := localTempDirectory()
	if err != nil {
		return CleanResult{}, err
	}
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	handle, err := lockCleanupRoot(userTemp)
	if err != nil {
		return CleanResult{}, fmt.Errorf("открытие каталога очистки: %w", err)
	}
	defer windows.CloseHandle(handle)
	resolved, err := filepath.Abs(userTemp)
	if err != nil {
		return CleanResult{}, err
	}
	return cleanTree(resolved, cutoff)
}

func lockCleanupRoot(path string) (windows.Handle, error) {
	pointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	handle, err := windows.CreateFile(
		pointer,
		windows.FILE_LIST_DIRECTORY|windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return 0, err
	}
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		windows.CloseHandle(handle)
		return 0, err
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 || info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 {
		windows.CloseHandle(handle)
		return 0, errors.New("корень очистки является reparse point или не каталогом")
	}
	return handle, nil
}

func cleanTree(root string, cutoff time.Time) (CleanResult, error) {
	root = filepath.Clean(root)
	volumeRoot := filepath.VolumeName(root) + string(filepath.Separator)
	if strings.EqualFold(root, volumeRoot) || root == string(filepath.Separator) {
		return CleanResult{}, fmt.Errorf("небезопасный корень очистки %q", root)
	}
	var result CleanResult
	var directories []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			result.Skipped++
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, `..\`) {
			return fmt.Errorf("путь вышел за корень очистки: %s", path)
		}
		info, err := entry.Info()
		if err != nil {
			result.Skipped++
			return nil
		}
		if isReparsePoint(info) {
			result.Skipped++
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			directories = append(directories, path)
			return nil
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err != nil {
				result.Skipped++
			} else {
				result.Files++
				result.Bytes += info.Size()
			}
		}
		return nil
	})
	sort.Slice(directories, func(i, j int) bool { return len(directories[i]) > len(directories[j]) })
	for _, directory := range directories {
		info, infoErr := os.Stat(directory)
		if infoErr == nil && info.ModTime().Before(cutoff) {
			if removeErr := os.Remove(directory); removeErr == nil {
				result.Dirs++
			}
		}
	}
	return result, err
}

func isReparsePoint(info os.FileInfo) bool {
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return ok && data.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}
