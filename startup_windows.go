package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/sys/windows/registry"
)

const (
	startupRunPath      = `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`
	startupDisabledPath = `SOFTWARE\GofMan3\Optimizer\StartupDisabled`
)

type StartupEntry struct {
	Scope   string `json:"scope"`
	Name    string `json:"name"`
	Command string `json:"command"`
	State   string `json:"state"`
}

type StartupReport struct {
	Entries  []StartupEntry `json:"entries"`
	Warnings []string       `json:"warnings,omitempty"`
}

func startupCommand(args []string) error {
	if len(args) == 0 || args[0] == "list" {
		if len(args) > 0 {
			args = args[1:]
		}
		set := flag.NewFlagSet("startup list", flag.ContinueOnError)
		jsonOnly := set.Bool("json", false, "вывести JSON")
		if err := set.Parse(args); err != nil {
			return err
		}
		if set.NArg() != 0 {
			return errors.New("лишние аргументы startup list")
		}
		report := listStartupEntries()
		if *jsonOnly {
			data, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}
		for _, entry := range report.Entries {
			fmt.Printf("[%s] %s (%s) — %s\n", entry.Scope, entry.Name, entry.State, entry.Command)
		}
		for _, warning := range report.Warnings {
			fmt.Println("Предупреждение:", warning)
		}
		return nil
	}
	if args[0] != "disable" && args[0] != "enable" {
		return errors.New("startup поддерживает list, disable и enable")
	}
	action := args[0]
	set := flag.NewFlagSet("startup "+action, flag.ContinueOnError)
	name := set.String("name", "", "точное имя значения HKCU Run")
	yes := set.Bool("yes", false, "подтвердить")
	if err := set.Parse(args[1:]); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("лишние аргументы startup " + action)
	}
	if err := validateStartupName(*name); err != nil {
		return err
	}
	if !*yes && !confirm(fmt.Sprintf("%s автозагрузку %q?", map[bool]string{true: "Отключить", false: "Включить"}[action == "disable"], *name)) {
		return errors.New("операция отменена")
	}
	release, err := acquireOperationLock()
	if err != nil {
		return err
	}
	defer release()
	if action == "disable" {
		err = disableStartup(*name)
	} else {
		err = enableStartup(*name)
	}
	if err == nil {
		fmt.Printf("Автозагрузка %q: %s.\n", *name, map[bool]string{true: "отключена", false: "включена"}[action == "disable"])
	}
	return err
}

func listStartupEntries() StartupReport {
	report := StartupReport{}
	readStartupKey(&report, registry.CURRENT_USER, startupRunPath, 0, "HKCU", "present")
	readStartupKey(&report, registry.LOCAL_MACHINE, startupRunPath, registry.WOW64_64KEY, "HKLM64", "present")
	readStartupKey(&report, registry.LOCAL_MACHINE, startupRunPath, registry.WOW64_32KEY, "HKLM32", "present")
	key, err := registry.OpenKey(registry.CURRENT_USER, startupDisabledPath, registry.QUERY_VALUE)
	if err == nil {
		names, readErr := key.ReadValueNames(-1)
		if readErr != nil {
			report.Warnings = append(report.Warnings, "disabled startup: "+readErr.Error())
		}
		for _, name := range names {
			values, _, readErr := key.GetStringsValue(name)
			_, command, decodeErr := decodeStartupBackup(values)
			if readErr == nil && decodeErr == nil {
				report.Entries = append(report.Entries, StartupEntry{Scope: "HKCU", Name: name, Command: command, State: "disabled_by_gofman3"})
			} else {
				report.Warnings = append(report.Warnings, "disabled startup "+name+": повреждён backup")
			}
		}
		key.Close()
	} else if !errors.Is(err, registry.ErrNotExist) {
		report.Warnings = append(report.Warnings, "disabled startup: "+err.Error())
	}
	sort.Slice(report.Entries, func(i, j int) bool {
		left, right := report.Entries[i], report.Entries[j]
		return left.Scope+"\x00"+strings.ToLower(left.Name) < right.Scope+"\x00"+strings.ToLower(right.Name)
	})
	return report
}

func readStartupKey(report *StartupReport, root registry.Key, path string, view uint32, scope, state string) {
	key, err := registry.OpenKey(root, path, registry.QUERY_VALUE|view)
	if errors.Is(err, registry.ErrNotExist) {
		return
	}
	if err != nil {
		report.Warnings = append(report.Warnings, scope+": "+err.Error())
		return
	}
	defer key.Close()
	names, err := key.ReadValueNames(-1)
	if err != nil {
		report.Warnings = append(report.Warnings, scope+": "+err.Error())
		return
	}
	for _, name := range names {
		command, kind, err := key.GetStringValue(name)
		if err == nil && (kind == registry.SZ || kind == registry.EXPAND_SZ) {
			report.Entries = append(report.Entries, StartupEntry{Scope: scope, Name: name, Command: command, State: state})
		}
	}
}

func disableStartup(name string) error {
	disabled, _, err := registry.CreateKey(registry.CURRENT_USER, startupDisabledPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer disabled.Close()
	existing, _, backupErr := disabled.GetStringsValue(name)
	if backupErr != nil && !errors.Is(backupErr, registry.ErrNotExist) {
		return backupErr
	}
	run, err := registry.OpenKey(registry.CURRENT_USER, startupRunPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer run.Close()
	command, kind, err := run.GetStringValue(name)
	if errors.Is(err, registry.ErrNotExist) && backupErr == nil {
		_, _, decodeErr := decodeStartupBackup(existing)
		return decodeErr
	}
	if err != nil {
		return fmt.Errorf("HKCU Run %q: %w", name, err)
	}
	if kind != registry.SZ && kind != registry.EXPAND_SZ {
		return errors.New("поддерживаются только строковые startup-команды")
	}
	backup := encodeStartupBackup(kind, command)
	if backupErr == nil && !slices.Equal(existing, backup) {
		return errors.New("для этого startup-имени уже существует другой backup")
	}
	if err := disabled.SetStringsValue(name, backup); err != nil {
		return err
	}
	if actual, _, err := disabled.GetStringsValue(name); err != nil || !slices.Equal(actual, backup) {
		return errors.New("backup startup-команды не прошёл read-back")
	}
	if actual, actualKind, err := run.GetStringValue(name); err != nil || actual != command || actualKind != kind {
		return errors.New("startup-команда изменилась во время создания backup")
	}
	if err := run.DeleteValue(name); err != nil {
		return err
	}
	if _, _, err := run.GetStringValue(name); !errors.Is(err, registry.ErrNotExist) {
		return errors.New("startup-команда осталась в HKCU Run")
	}
	return nil
}

func enableStartup(name string) error {
	disabled, err := registry.OpenKey(registry.CURRENT_USER, startupDisabledPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer disabled.Close()
	values, _, err := disabled.GetStringsValue(name)
	if err != nil {
		return err
	}
	kind, command, err := decodeStartupBackup(values)
	if err != nil {
		return err
	}
	run, _, err := registry.CreateKey(registry.CURRENT_USER, startupRunPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer run.Close()
	if actual, actualKind, readErr := run.GetStringValue(name); readErr == nil {
		if actual == command && actualKind == kind {
			return disabled.DeleteValue(name)
		}
		return errors.New("HKCU Run уже содержит другое значение с таким именем")
	} else if !errors.Is(readErr, registry.ErrNotExist) {
		return readErr
	}
	if kind == registry.EXPAND_SZ {
		err = run.SetExpandStringValue(name, command)
	} else {
		err = run.SetStringValue(name, command)
	}
	if err != nil {
		return err
	}
	actual, actualKind, err := run.GetStringValue(name)
	if err != nil || actual != command || actualKind != kind {
		return errors.New("восстановленная startup-команда не прошла read-back")
	}
	return disabled.DeleteValue(name)
}

func validateStartupName(name string) error {
	if strings.TrimSpace(name) == "" || len(name) > 256 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return errors.New("укажите корректное startup-имя длиной до 256 символов")
	}
	return nil
}

func encodeStartupBackup(kind uint32, command string) []string {
	return []string{strconv.FormatUint(uint64(kind), 10), command}
}

func decodeStartupBackup(values []string) (uint32, string, error) {
	if len(values) != 2 {
		return 0, "", errors.New("неверный startup backup")
	}
	kind, err := strconv.ParseUint(values[0], 10, 32)
	if err != nil || (uint32(kind) != registry.SZ && uint32(kind) != registry.EXPAND_SZ) {
		return 0, "", errors.New("неподдерживаемый тип startup backup")
	}
	return uint32(kind), values[1], nil
}
