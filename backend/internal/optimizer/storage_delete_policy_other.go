//go:build !windows

package optimizer

import (
	"os"
	"path/filepath"
	"strings"
)

func storageDeleteTargetProtected(root, path string, _ os.FileInfo) bool {
	relative, err := filepath.Rel(root, path)
	if err == nil {
		first := strings.ToLower(strings.Split(relative, string(filepath.Separator))[0])
		if strings.HasPrefix(first, ".trash") || first == "lost+found" {
			return true
		}
	}
	for _, protected := range []string{"/bin", "/boot", "/dev", "/etc", "/lib", "/lib64", "/proc", "/run", "/sbin", "/sys", "/usr", "/var"} {
		if storagePathWithin(protected, path) || storagePathWithin(path, protected) {
			return true
		}
	}
	if executable, err := os.Executable(); err == nil && storagePathWithin(path, executable) {
		return true
	}
	return false
}
