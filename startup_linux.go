package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
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
			fmt.Printf("%-8s %-8s %s\n  %s\n", displayText(entry.State), displayText(entry.Scope), displayText(entry.Name), displayText(entry.Command))
		}
		for _, warning := range report.Warnings {
			fmt.Println("Предупреждение:", displayText(warning))
		}
		return nil
	}
	set := flag.NewFlagSet("startup "+args[0], flag.ContinueOnError)
	name := set.String("name", "", "имя .desktop из startup list")
	if err := set.Parse(args[1:]); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("лишние аргументы startup")
	}
	switch args[0] {
	case "disable":
		return disableStartup(*name)
	case "enable":
		return enableStartup(*name)
	default:
		return errors.New("startup поддерживает list, disable и enable")
	}
}

func listStartupEntries() StartupReport {
	report := StartupReport{Entries: []StartupEntry{}}
	userDir, err := userStartupDirectory()
	if err != nil {
		report.Warnings = append(report.Warnings, err.Error())
		return report
	}
	readStartupDirectory(&report, userDir, "user", "present")
	readStartupDirectory(&report, disabledStartupDirectory(userDir), "user", "disabled")
	configDirs := strings.Split(firstNonEmpty(os.Getenv("XDG_CONFIG_DIRS"), "/etc/xdg"), ":")
	for _, base := range configDirs {
		if filepath.IsAbs(base) {
			readStartupDirectory(&report, filepath.Join(base, "autostart"), "system", "read-only")
		}
	}
	sort.Slice(report.Entries, func(i, j int) bool {
		return report.Entries[i].Scope+report.Entries[i].Name+report.Entries[i].State < report.Entries[j].Scope+report.Entries[j].Name+report.Entries[j].State
	})
	return report
}

func readStartupDirectory(report *StartupReport, dir, scope, state string) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		report.Warnings = append(report.Warnings, dir+": "+err.Error())
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".desktop") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, infoErr := os.Lstat(path)
		if infoErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		values, parseErr := readDesktopEntry(path)
		if parseErr != nil {
			report.Warnings = append(report.Warnings, path+": "+parseErr.Error())
			continue
		}
		name := firstNonEmpty(values["Name"], entry.Name())
		report.Entries = append(report.Entries, StartupEntry{Scope: scope, Name: entry.Name(), Command: firstNonEmpty(values["Exec"], name), State: state})
	}
}

func readDesktopEntry(path string) (map[string]string, error) {
	data, err := readSmallFile(path, 256<<10)
	if err != nil {
		return nil, err
	}
	values := make(map[string]string)
	section := ""
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line
			continue
		}
		if section != "[Desktop Entry]" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if ok {
			values[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	if values["Type"] != "Application" || values["Exec"] == "" {
		return nil, errors.New("файл не является запускаемой Desktop Entry")
	}
	return values, nil
}

func disableStartup(name string) error {
	if err := validateStartupName(name); err != nil {
		return err
	}
	userDir, err := userStartupDirectory()
	if err != nil {
		return err
	}
	source, target := filepath.Join(userDir, name), filepath.Join(disabledStartupDirectory(userDir), name)
	if err := validateMovableStartupFile(source); err != nil {
		return fmt.Errorf("можно отключать только пользовательские XDG entries: %w", err)
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		return errors.New("startup entry уже отключена или target занят")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	if err := os.Rename(source, target); err != nil {
		return err
	}
	fmt.Println("Startup entry отключена:", name)
	return nil
}

func enableStartup(name string) error {
	if err := validateStartupName(name); err != nil {
		return err
	}
	userDir, err := userStartupDirectory()
	if err != nil {
		return err
	}
	source, target := filepath.Join(disabledStartupDirectory(userDir), name), filepath.Join(userDir, name)
	if err := validateMovableStartupFile(source); err != nil {
		return err
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		return errors.New("startup entry уже присутствует или target занят")
	}
	if err := os.MkdirAll(userDir, 0o700); err != nil {
		return err
	}
	if err := os.Rename(source, target); err != nil {
		return err
	}
	fmt.Println("Startup entry включена:", name)
	return nil
}

func userStartupDirectory() (string, error) {
	config, err := xdgDirectory("XDG_CONFIG_HOME", ".config")
	if err != nil {
		return "", err
	}
	return filepath.Join(config, "autostart"), nil
}

func disabledStartupDirectory(userDir string) string {
	return filepath.Join(userDir, ".luxury-optimization-disabled")
}

func validateStartupName(name string) error {
	if name == "" || len(name) > 255 || filepath.Base(name) != name || !strings.HasSuffix(name, ".desktop") || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return errors.New("укажите безопасное имя .desktop из startup list")
	}
	return nil
}

func validateMovableStartupFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("entry не является обычным файлом")
	}
	_, err = readDesktopEntry(path)
	return err
}
