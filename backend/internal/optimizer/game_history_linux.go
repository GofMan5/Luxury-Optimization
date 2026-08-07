package optimizer

import "os"

func validGameHistoryFile(info os.FileInfo) bool {
	return info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func protectGameHistoryDirectory(path string) error { return os.Chmod(path, 0o700) }

func protectGameHistoryFile(file *os.File) error { return file.Chmod(0o600) }
