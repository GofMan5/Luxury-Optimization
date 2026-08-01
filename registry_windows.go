package main

import (
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"golang.org/x/sys/windows/registry"
)

var (
	registryUserSID string
	sidPattern      = regexp.MustCompile(`^S-\d+(?:-\d+)+$`)
)

func setRegistryUserSID(sid string) error {
	if sid != "" && !sidPattern.MatchString(sid) {
		return fmt.Errorf("некорректный SID пользователя")
	}
	registryUserSID = sid
	return nil
}

func registryView() uint32 {
	if runtime.GOARCH == "386" && isWOW64() {
		return registry.WOW64_64KEY
	}
	return 0
}

func registryLocation(hive, path string) (registry.Key, string, error) {
	switch strings.ToUpper(hive) {
	case "HKCU":
		if registryUserSID != "" {
			return registry.USERS, registryUserSID + `\` + path, nil
		}
		return registry.CURRENT_USER, path, nil
	case "HKLM":
		return registry.LOCAL_MACHINE, path, nil
	default:
		return 0, "", fmt.Errorf("неподдерживаемый hive %q", hive)
	}
}

func snapshotRegistry(change RegChange) (RegSnapshot, error) {
	snapshot := RegSnapshot{Change: change}
	root, path, err := registryLocation(change.Hive, change.Path)
	if err != nil {
		return snapshot, err
	}
	key, err := registry.OpenKey(root, path, registry.QUERY_VALUE|registryView())
	if errors.Is(err, registry.ErrNotExist) {
		return snapshot, nil
	}
	if err != nil {
		return snapshot, fmt.Errorf("открытие %s\\%s: %w", change.Hive, change.Path, err)
	}
	defer key.Close()
	_, kind, err := key.GetValue(change.Name, nil)
	if errors.Is(err, registry.ErrNotExist) {
		return snapshot, nil
	}
	if err != nil {
		return snapshot, fmt.Errorf("чтение %s: %w", change.ID, err)
	}
	snapshot.Existed = true
	snapshot.Kind = kind
	switch kind {
	case registry.DWORD:
		value, _, err := key.GetIntegerValue(change.Name)
		snapshot.DWord = uint32(value)
		return snapshot, err
	case registry.QWORD:
		value, _, err := key.GetIntegerValue(change.Name)
		snapshot.QWord = value
		return snapshot, err
	case registry.SZ, registry.EXPAND_SZ:
		value, _, err := key.GetStringValue(change.Name)
		snapshot.String = value
		return snapshot, err
	case registry.MULTI_SZ:
		value, _, err := key.GetStringsValue(change.Name)
		snapshot.Strings = value
		return snapshot, err
	case registry.BINARY:
		value, _, err := key.GetBinaryValue(change.Name)
		snapshot.Binary = value
		return snapshot, err
	default:
		return snapshot, fmt.Errorf("%s: тип реестра %d не поддерживается для безопасного отката", change.ID, kind)
	}
}

func applyRegistry(change RegChange) error {
	root, path, err := registryLocation(change.Hive, change.Path)
	if err != nil {
		return err
	}
	key, _, err := registry.CreateKey(root, path, registry.SET_VALUE|registry.QUERY_VALUE|registryView())
	if err != nil {
		return err
	}
	defer key.Close()
	switch change.Kind {
	case registry.DWORD:
		return key.SetDWordValue(change.Name, change.DWord)
	case registry.SZ:
		return key.SetStringValue(change.Name, change.String)
	default:
		return fmt.Errorf("%s: неподдерживаемый desired type %d", change.ID, change.Kind)
	}
}

func restoreRegistry(snapshot RegSnapshot) error {
	root, path, err := registryLocation(snapshot.Change.Hive, snapshot.Change.Path)
	if err != nil {
		return err
	}
	if !snapshot.Existed {
		key, err := registry.OpenKey(root, path, registry.SET_VALUE|registryView())
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		err = key.DeleteValue(snapshot.Change.Name)
		key.Close()
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		return nil
	}
	key, _, err := registry.CreateKey(root, path, registry.SET_VALUE|registryView())
	if err != nil {
		return err
	}
	defer key.Close()
	switch snapshot.Kind {
	case registry.DWORD:
		return key.SetDWordValue(snapshot.Change.Name, snapshot.DWord)
	case registry.QWORD:
		return key.SetQWordValue(snapshot.Change.Name, snapshot.QWord)
	case registry.SZ:
		return key.SetStringValue(snapshot.Change.Name, snapshot.String)
	case registry.EXPAND_SZ:
		return key.SetExpandStringValue(snapshot.Change.Name, snapshot.String)
	case registry.MULTI_SZ:
		return key.SetStringsValue(snapshot.Change.Name, snapshot.Strings)
	case registry.BINARY:
		return key.SetBinaryValue(snapshot.Change.Name, snapshot.Binary)
	default:
		return fmt.Errorf("%s: невозможно восстановить тип %d", snapshot.Change.ID, snapshot.Kind)
	}
}

func registryMatches(change RegChange) (bool, string, error) {
	snapshot, err := snapshotRegistry(change)
	if err != nil {
		return false, "", err
	}
	current := formatSnapshot(snapshot)
	if !snapshot.Existed || snapshot.Kind != change.Kind {
		return false, current, nil
	}
	switch change.Kind {
	case registry.DWORD:
		return snapshot.DWord == change.DWord, current, nil
	case registry.SZ:
		return snapshot.String == change.String, current, nil
	default:
		return false, current, nil
	}
}

func formatSnapshot(snapshot RegSnapshot) string {
	if !snapshot.Existed {
		return "не задано"
	}
	switch snapshot.Kind {
	case registry.DWORD:
		return fmt.Sprintf("%d", snapshot.DWord)
	case registry.QWORD:
		return fmt.Sprintf("%d", snapshot.QWord)
	case registry.SZ, registry.EXPAND_SZ:
		return fmt.Sprintf("%q", snapshot.String)
	case registry.MULTI_SZ:
		return strings.Join(snapshot.Strings, ", ")
	case registry.BINARY:
		return fmt.Sprintf("binary (%d байт)", len(snapshot.Binary))
	default:
		return fmt.Sprintf("тип %d", snapshot.Kind)
	}
}

func formatDesired(change RegChange) string {
	switch change.Kind {
	case registry.DWORD:
		return fmt.Sprintf("%d", change.DWord)
	case registry.SZ:
		return fmt.Sprintf("%q", change.String)
	default:
		return fmt.Sprintf("тип %d", change.Kind)
	}
}
