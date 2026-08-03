package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	if os.Geteuid() == 0 {
		return CleanResult{}, errors.New("очистка запрещена от root")
	}
	root := filepath.Clean(os.TempDir())
	if !filepath.IsAbs(root) {
		return CleanResult{}, errors.New("temporary directory is not absolute")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return CleanResult{}, err
	}
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	result := CleanResult{}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "luxury-optimization-") {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if !pathWithin(root, path) {
			result.Skipped++
			continue
		}
		info, infoErr := os.Lstat(path)
		if infoErr != nil || info.Mode()&os.ModeSymlink != 0 || info.ModTime().After(cutoff) {
			result.Skipped++
			continue
		}
		if info.IsDir() {
			if err := os.Remove(path); err != nil {
				result.Skipped++
			} else {
				result.Dirs++
			}
			continue
		}
		if !info.Mode().IsRegular() {
			result.Skipped++
			continue
		}
		if err := os.Remove(path); err != nil {
			result.Skipped++
		} else {
			result.Files++
			result.Bytes += info.Size()
		}
	}
	return result, nil
}
