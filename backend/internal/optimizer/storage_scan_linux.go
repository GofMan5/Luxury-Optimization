package optimizer

import (
	"os"
	"path/filepath"
	"strings"
)

func storageEntryIsReparse(info os.FileInfo) bool { return info.Mode()&os.ModeSymlink != 0 }

func storageScanMountExclusions(root string) (map[string]bool, error) {
	data, err := readSmallFile("/proc/self/mountinfo", 4<<20)
	if err != nil {
		return nil, err
	}
	root = filepath.Clean(root)
	result := make(map[string]bool)
	for _, mount := range parseMountInfo(string(data)) {
		path := filepath.Clean(mount.path)
		if path != root && (root == "/" || strings.HasPrefix(path, root+string(filepath.Separator))) {
			result[path] = true
		}
	}
	return result, nil
}
