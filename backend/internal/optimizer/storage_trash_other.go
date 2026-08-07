//go:build !windows

package optimizer

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func randomStorageTrashSuffix() (string, error) {
	value := make([]byte, 8)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func moveStorageTargetToTrash(path string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	trash := filepath.Join(dataHome, "Trash")
	files, info := filepath.Join(trash, "files"), filepath.Join(trash, "info")
	if err := os.MkdirAll(files, 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(info, 0o700); err != nil {
		return err
	}
	suffix, err := randomStorageTrashSuffix()
	if err != nil {
		return err
	}
	name := filepath.Base(path) + ".luxury-" + suffix
	infoPath := filepath.Join(info, name+".trashinfo")
	metadata, err := os.OpenFile(infoPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	keepInfo := false
	defer func() {
		_ = metadata.Close()
		if !keepInfo {
			_ = os.Remove(infoPath)
		}
	}()
	contents := fmt.Sprintf("[Trash Info]\nPath=%s\nDeletionDate=%s\n", url.PathEscape(path), time.Now().Format("2006-01-02T15:04:05"))
	if _, err := metadata.WriteString(contents); err != nil {
		return err
	}
	if err := metadata.Sync(); err != nil {
		return err
	}
	if err := metadata.Close(); err != nil {
		return err
	}
	if err := os.Rename(path, filepath.Join(files, name)); err != nil {
		return fmt.Errorf("trash directory must be on the same filesystem: %w", err)
	}
	keepInfo = true
	return nil
}
