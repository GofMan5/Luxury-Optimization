package main

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func acquireUpdateLock() (func(), error) {
	configPath, err := updateConfigPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(configPath+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		lock.Close()
		return nil, errors.New("другое обновление уже выполняется")
	}
	return func() {
		_ = unix.Flock(int(lock.Fd()), unix.LOCK_UN)
		_ = lock.Close()
	}, nil
}

func installDownloadedUpdate(pendingPath, executablePath string) (string, error) {
	pendingInfo, err := os.Lstat(pendingPath)
	if err != nil {
		return "", err
	}
	executableInfo, err := os.Lstat(executablePath)
	if err != nil {
		return "", err
	}
	if !pendingInfo.Mode().IsRegular() || pendingInfo.Mode()&os.ModeSymlink != 0 || !executableInfo.Mode().IsRegular() || executableInfo.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("self-update требует обычные файлы без symlink")
	}
	if filepath.Dir(pendingPath) != filepath.Dir(executablePath) {
		return "", errors.New("pending update должен находиться рядом с executable")
	}
	if err := os.Chmod(pendingPath, executableInfo.Mode().Perm()); err != nil {
		return "", err
	}
	if err := os.Rename(pendingPath, executablePath); err != nil {
		return "", err
	}
	return "Обновление установлено атомарно; следующий запуск использует новую версию.", nil
}
