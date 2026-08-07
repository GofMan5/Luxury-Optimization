package optimizer

import "os"

func validGameHistoryFile(info os.FileInfo) bool {
	return info.Mode().IsRegular() && !isReparsePoint(info)
}

func protectGameHistoryDirectory(string) error { return nil }

func protectGameHistoryFile(file *os.File) error { return protectResultFile(file) }
