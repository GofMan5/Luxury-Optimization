package optimizer

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

func storageDeleteTargetProtected(root, path string, info os.FileInfo) bool {
	if data, ok := info.Sys().(*syscall.Win32FileAttributeData); ok && data.FileAttributes&windows.FILE_ATTRIBUTE_SYSTEM != 0 {
		return true
	}
	relative, err := filepath.Rel(root, path)
	if err == nil {
		first := strings.ToLower(strings.Split(relative, string(filepath.Separator))[0])
		if strings.HasPrefix(first, "$") || map[string]bool{"system volume information": true, "recovery": true, "boot": true, "efi": true, "config.msi": true}[first] {
			return true
		}
	}
	if sameStoragePath(filepath.Dir(path), root) && map[string]bool{"pagefile.sys": true, "hiberfil.sys": true, "swapfile.sys": true, "bootmgr": true, "bootnxt": true, "dumpstack.log.tmp": true}[strings.ToLower(filepath.Base(path))] {
		return true
	}
	for _, protected := range recursiveWindowsStoragePaths() {
		if storagePathWithin(protected, path) || storagePathWithin(path, protected) {
			return true
		}
	}
	for _, protected := range identityWindowsStoragePaths() {
		if storagePathWithin(path, protected) {
			return true
		}
	}
	if executable, err := os.Executable(); err == nil && storagePathWithin(path, executable) {
		return true
	}
	return false
}

func recursiveWindowsStoragePaths() []string {
	paths := make([]string, 0, 4)
	if path, err := windowsDirectory(); err == nil {
		paths = append(paths, path)
	}
	for _, id := range []*windows.KNOWNFOLDERID{windows.FOLDERID_ProgramData, windows.FOLDERID_ProgramFiles, windows.FOLDERID_ProgramFilesX86} {
		if path, err := windows.KnownFolderPath(id, windows.KF_FLAG_DEFAULT); err == nil {
			paths = append(paths, filepath.Clean(path))
		}
	}
	return paths
}

func identityWindowsStoragePaths() []string {
	paths := make([]string, 0, 2)
	for _, id := range []*windows.KNOWNFOLDERID{windows.FOLDERID_UserProfiles, windows.FOLDERID_Profile} {
		if path, err := windows.KnownFolderPath(id, windows.KF_FLAG_DEFAULT); err == nil {
			paths = append(paths, filepath.Clean(path))
		}
	}
	return paths
}
