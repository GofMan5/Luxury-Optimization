package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
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
}

func servicesCommand(args []string) error {
	if len(args) > 0 && args[0] == "list" {
		args = args[1:]
	} else if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return errors.New("services поддерживает только list")
	}
	set := flag.NewFlagSet("services list", flag.ContinueOnError)
	jsonOnly := set.Bool("json", false, "вывести JSON")
	state := set.String("state", "all", "all, running или stopped")
	match := set.String("match", "", "фильтр имени или display name")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("лишние аргументы services list")
	}
	if *state != "all" && *state != "running" && *state != "stopped" {
		return errors.New("state поддерживает all, running и stopped")
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
		fmt.Printf("%-8s %-10s %s (%s)\n", service.State, service.StartType, service.Display, service.Name)
	}
	if report.Skipped > 0 {
		fmt.Println("Пропущено недоступных служб:", report.Skipped)
	}
	return nil
}

func listServices(state, match string) (ServicesReport, error) {
	handle, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT|windows.SC_MANAGER_ENUMERATE_SERVICE)
	if err != nil {
		return ServicesReport{}, err
	}
	manager := &mgr.Mgr{Handle: handle}
	defer manager.Disconnect()
	names, err := manager.ListServices()
	if err != nil {
		return ServicesReport{}, err
	}
	report := ServicesReport{}
	match = strings.ToLower(match)
	for _, name := range names {
		namePointer, pointerErr := windows.UTF16PtrFromString(name)
		if pointerErr != nil {
			report.Skipped++
			continue
		}
		serviceHandle, err := windows.OpenService(manager.Handle, namePointer, windows.SERVICE_QUERY_CONFIG|windows.SERVICE_QUERY_STATUS)
		if err != nil {
			report.Skipped++
			continue
		}
		service := &mgr.Service{Name: name, Handle: serviceHandle}
		config, configErr := service.Config()
		status, statusErr := service.Query()
		service.Close()
		if configErr != nil || statusErr != nil {
			report.Skipped++
			continue
		}
		entry := ServiceEntry{Name: name, Display: config.DisplayName, State: serviceState(status.State), StartType: serviceStartType(config.StartType), ProcessID: status.ProcessId, BinaryPath: config.BinaryPathName}
		if state != "all" && entry.State != state {
			continue
		}
		if match != "" && !strings.Contains(strings.ToLower(entry.Name+" "+entry.Display), match) {
			continue
		}
		report.Services = append(report.Services, entry)
	}
	sort.Slice(report.Services, func(i, j int) bool {
		return strings.ToLower(report.Services[i].Display) < strings.ToLower(report.Services[j].Display)
	})
	return report, nil
}

func serviceState(state svc.State) string {
	if state == svc.Running {
		return "running"
	}
	if state == svc.Stopped {
		return "stopped"
	}
	return "pending"
}

func serviceStartType(value uint32) string {
	switch value {
	case windows.SERVICE_AUTO_START:
		return "automatic"
	case windows.SERVICE_DEMAND_START:
		return "manual"
	case windows.SERVICE_DISABLED:
		return "disabled"
	case windows.SERVICE_BOOT_START:
		return "boot"
	case windows.SERVICE_SYSTEM_START:
		return "system"
	default:
		return fmt.Sprintf("0x%X", value)
	}
}
