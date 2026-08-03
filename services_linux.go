package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type ServiceEntry struct {
	Name       string `json:"name"`
	Display    string `json:"display_name"`
	State      string `json:"state"`
	StartType  string `json:"start_type"`
	ProcessID  uint32 `json:"process_id,omitempty"`
	BinaryPath string `json:"binary_path,omitempty"`
}

type ServicesReport struct {
	Services []ServiceEntry `json:"services"`
	Skipped  int            `json:"skipped"`
	Warnings []string       `json:"warnings,omitempty"`
}

func servicesCommand(args []string) error {
	if len(args) > 0 && args[0] == "list" {
		args = args[1:]
	} else if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return errors.New("services поддерживает только list")
	}
	set := flag.NewFlagSet("services list", flag.ContinueOnError)
	state := set.String("state", "all", "all, running, stopped или failed")
	match := set.String("match", "", "фильтр имени/описания")
	jsonOnly := set.Bool("json", false, "вывести JSON")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("лишние аргументы services list")
	}
	report, err := listServices(*state, *match)
	if err != nil {
		return err
	}
	if *jsonOnly {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}
	for _, service := range report.Services {
		fmt.Printf("%-10s %-10s %s\n  %s\n", displayText(service.State), displayText(service.StartType), displayText(service.Name), displayText(service.Display))
	}
	for _, warning := range report.Warnings {
		fmt.Println("Предупреждение:", displayText(warning))
	}
	return nil
}

func listServices(state, match string) (ServicesReport, error) {
	state = strings.ToLower(state)
	if state != "all" && state != "running" && state != "stopped" && state != "failed" {
		return ServicesReport{}, errors.New("state поддерживает all, running, stopped и failed")
	}
	systemctl, err := exec.LookPath("systemctl")
	if err != nil || !isSystemdBooted() {
		return ServicesReport{Services: []ServiceEntry{}, Warnings: []string{"systemd не активен; service inventory безопасно пропущен"}}, nil
	}
	unitFiles, err := runSystemctl(systemctl, "list-unit-files", "--type=service", "--no-legend", "--no-pager", "--plain")
	if err != nil {
		return ServicesReport{}, err
	}
	startTypes := make(map[string]string)
	for _, line := range strings.Split(unitFiles, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			startTypes[fields[0]] = fields[1]
		}
	}
	units, err := runSystemctl(systemctl, "list-units", "--type=service", "--all", "--no-legend", "--no-pager", "--plain")
	if err != nil {
		return ServicesReport{}, err
	}
	report := ServicesReport{Services: []ServiceEntry{}}
	match = strings.ToLower(strings.TrimSpace(match))
	for _, line := range strings.Split(units, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		entry := ServiceEntry{Name: fields[0], State: fields[3], StartType: startTypes[fields[0]], Display: strings.Join(fields[4:], " ")}
		if entry.StartType == "" {
			entry.StartType = "unknown"
		}
		logicalState := entry.State
		if logicalState == "dead" || logicalState == "inactive" || logicalState == "exited" {
			logicalState = "stopped"
		}
		if state != "all" && logicalState != state {
			continue
		}
		if match != "" && !strings.Contains(strings.ToLower(entry.Name+" "+entry.Display), match) {
			continue
		}
		report.Services = append(report.Services, entry)
	}
	sort.Slice(report.Services, func(i, j int) bool { return report.Services[i].Name < report.Services[j].Name })
	return report, nil
}

func runSystemctl(path string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, path, args...)
	command.Env = append(os.Environ(), "LC_ALL=C", "LANG=C", "SYSTEMD_COLORS=0")
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		return "", errors.New("systemctl timeout")
	}
	if err != nil {
		return "", fmt.Errorf("systemctl: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}
