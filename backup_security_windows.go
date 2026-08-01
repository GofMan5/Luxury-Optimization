package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	backupDigestKey = `SOFTWARE\GofMan3\Optimizer\BackupDigests`
	maxBackupSize   = 4 << 20
)

var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func ensureSecureDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(path, 0o700); err != nil {
			return err
		}
		info, err = os.Lstat(path)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("путь состояния не является каталогом: %s", path)
	}
	pathPointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	handle, err := windows.CreateFile(
		pathPointer,
		windows.READ_CONTROL|windows.WRITE_DAC|windows.WRITE_OWNER,
		windows.FILE_SHARE_READ,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return fmt.Errorf("безопасное открытие %s: %w", path, err)
	}
	defer windows.CloseHandle(handle)
	var fileInfo windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &fileInfo); err != nil {
		return err
	}
	if fileInfo.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return fmt.Errorf("каталог состояния является reparse point: %s", path)
	}
	descriptor, err := windows.SecurityDescriptorFromString(`O:BAG:BAD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)`)
	if err != nil {
		return err
	}
	owner, _, err := descriptor.Owner()
	if err != nil {
		return err
	}
	group, _, err := descriptor.Group()
	if err != nil {
		return err
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	return windows.SetSecurityInfo(
		handle,
		windows.SE_FILE_OBJECT,
		windows.OWNER_SECURITY_INFORMATION|windows.GROUP_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		owner,
		group,
		dacl,
		nil,
	)
}

func productionStateDirectory() (string, error) {
	base, err := appDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "state"), nil
}

func isProductionBackupPath(path string) bool {
	directory, err := productionStateDirectory()
	if err != nil {
		return false
	}
	return strings.EqualFold(filepath.Clean(filepath.Dir(path)), filepath.Clean(directory)) && backupFilePattern.MatchString(filepath.Base(path))
}

func backupDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum)
}

func prepareBackupSeal(id, digest string) error {
	if !digestPattern.MatchString(digest) || !strings.HasSuffix(id, "Z") {
		return errors.New("неверный идентификатор или digest backup")
	}
	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, backupDigestKey, registry.QUERY_VALUE|registry.SET_VALUE|registryView())
	if err != nil {
		return err
	}
	defer key.Close()
	values := []string{digest}
	if previous, _, err := key.GetStringsValue(id); err == nil {
		for _, value := range previous {
			if digestPattern.MatchString(value) && value != digest {
				values = append(values, value)
				break
			}
		}
	} else if !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	return key.SetStringsValue(id, values)
}

func commitBackupSeal(id, digest string) error {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, backupDigestKey, registry.SET_VALUE|registryView())
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringsValue(id, []string{digest})
}

func verifyBackupSeal(path, id string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || isReparsePoint(info) || info.Size() <= 0 || info.Size() > maxBackupSize {
		return errors.New("backup не является допустимым обычным файлом")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	digest := backupDigest(data)
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, backupDigestKey, registry.QUERY_VALUE|registryView())
	if err != nil {
		return errors.New("для backup отсутствует доверенная печать")
	}
	defer key.Close()
	values, _, err := key.GetStringsValue(id)
	if err != nil {
		return errors.New("для backup отсутствует доверенная печать")
	}
	for _, value := range values {
		if value == digest {
			return nil
		}
	}
	return errors.New("SHA-256 backup не совпадает с доверенной печатью HKLM")
}
