package optimizer

import "os"

func storageEntryIsReparse(info os.FileInfo) bool { return isReparsePoint(info) }

func storageScanMountExclusions(string) (map[string]bool, error) { return nil, nil }
