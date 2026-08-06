package optimizer

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

type ServiceEntry struct {
	Name         string   `json:"name"`
	Display      string   `json:"display_name"`
	State        string   `json:"state"`
	StartType    string   `json:"start_type"`
	ProcessID    uint32   `json:"process_id,omitempty"`
	BinaryPath   string   `json:"binary_path,omitempty"`
	System       bool     `json:"system"`
	Description  string   `json:"description,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	Critical     bool     `json:"critical"`
	Manageable   bool     `json:"manageable"`
}

const serviceBackupPath = `SOFTWARE\GofMan3\Optimizer\ServiceBackups`

type ServicesReport struct {
	Services []ServiceEntry `json:"services"`
	Skipped  int            `json:"skipped"`
}

func servicesCommand(args []string) error {
	if len(args) > 0 && args[0] == "set" {
		return serviceSetCommand(args[1:])
	}
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
		fmt.Printf("%-8s %-10s %s (%s)\n", displayText(service.State), displayText(service.StartType), displayText(service.Display), displayText(service.Name))
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
	report := ServicesReport{Services: []ServiceEntry{}}
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
		critical := criticalWindowsService(name)
		entry := ServiceEntry{Name: name, Display: config.DisplayName, State: serviceState(status.State), StartType: serviceStartType(config.StartType), ProcessID: status.ProcessId, BinaryPath: config.BinaryPathName, System: isWindowsSystemService(config.BinaryPathName), Description: config.Description, Dependencies: append([]string(nil), config.Dependencies...), Critical: critical, Manageable: !critical || config.StartType == windows.SERVICE_DISABLED}
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

func serviceSetCommand(args []string) error {
	set := flag.NewFlagSet("services set", flag.ContinueOnError)
	name := set.String("name", "", "точное имя службы")
	enabled := set.Bool("enabled", false, "включить службу")
	yes := set.Bool("yes", false, "подтвердить")
	quiet := set.Bool("quiet", false, "не печатать результат")
	parentPID := set.Uint("parent-pid", 0, "internal: PID исходного процесса")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 || validateServiceName(*name) != nil {
		return errors.New("укажите корректное точное имя службы")
	}
	if !*yes && !confirm(fmt.Sprintf("%s запуск службы %q?", map[bool]string{true: "Включить", false: "Отключить"}[*enabled], *name)) {
		return errors.New("операция отменена")
	}
	if !isAdministrator() {
		return runElevatedAndWait([]string{"services", "set", "--name", *name, "--enabled=" + strconv.FormatBool(*enabled), "--yes", "--quiet", "--parent-pid", strconv.Itoa(os.Getpid())})
	}
	if *parentPID != 0 {
		if _, err := userSIDFromOptimizerProcess(uint32(*parentPID)); err != nil {
			return err
		}
	}
	result, err := setServiceEnabled(*name, *enabled)
	if err == nil && !*quiet {
		fmt.Println(result.Message)
	}
	return err
}

func setServiceEnabled(name string, enabled bool) (MutationResult, error) {
	if err := validateServiceName(name); err != nil {
		return MutationResult{}, err
	}
	if !enabled && criticalWindowsService(name) {
		return MutationResult{}, errors.New("критическая служба Windows защищена от отключения")
	}
	release, err := acquireOperationLock()
	if err != nil {
		return MutationResult{}, err
	}
	defer release()
	manager, err := mgr.Connect()
	if err != nil {
		return MutationResult{}, err
	}
	defer manager.Disconnect()
	service, err := manager.OpenService(name)
	if err != nil {
		return MutationResult{}, err
	}
	defer service.Close()
	config, err := service.Config()
	if err != nil {
		return MutationResult{}, err
	}
	if enabled && config.StartType != windows.SERVICE_DISABLED {
		return MutationResult{Message: "Служба уже включена."}, nil
	}
	if !enabled {
		if config.StartType == windows.SERVICE_DISABLED {
			return MutationResult{Message: "Служба уже отключена."}, nil
		}
		if err := saveServiceBackup(name, config.StartType, config.DelayedAutoStart); err != nil {
			return MutationResult{}, err
		}
		config.StartType, config.DelayedAutoStart = windows.SERVICE_DISABLED, false
	} else {
		startType, delayed, found, err := loadServiceBackup(name)
		if err != nil {
			return MutationResult{}, err
		}
		if found {
			config.StartType, config.DelayedAutoStart = startType, delayed
		} else {
			config.StartType, config.DelayedAutoStart = windows.SERVICE_DEMAND_START, false
		}
	}
	if err := service.UpdateConfig(config); err != nil {
		return MutationResult{}, err
	}
	actual, err := service.Config()
	if err != nil || actual.StartType != config.StartType || actual.DelayedAutoStart != config.DelayedAutoStart {
		if err != nil {
			return MutationResult{}, err
		}
		return MutationResult{}, errors.New("настройка службы не прошла read-back")
	}
	if enabled {
		if err := deleteServiceBackup(name); err != nil {
			return MutationResult{}, err
		}
	}
	action := "отключена для следующих запусков"
	if enabled {
		action = "включена; восстановлен исходный тип запуска либо безопасный Manual"
	}
	return MutationResult{Changed: true, Message: fmt.Sprintf("Служба %q %s. Текущий процесс службы не завершался.", name, action)}, nil
}

func saveServiceBackup(name string, startType uint32, delayed bool) error {
	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, serviceBackupPath, registry.QUERY_VALUE|registry.SET_VALUE|registryView())
	if err != nil {
		return err
	}
	defer key.Close()
	if existing, _, readErr := key.GetStringsValue(name); readErr == nil {
		_, _, decodeErr := decodeServiceBackup(existing)
		return decodeErr
	} else if !errors.Is(readErr, registry.ErrNotExist) {
		return readErr
	}
	values := []string{"v1", strconv.FormatUint(uint64(startType), 10), strconv.FormatBool(delayed)}
	if err := key.SetStringsValue(name, values); err != nil {
		return err
	}
	actual, _, err := key.GetStringsValue(name)
	if err != nil || strings.Join(actual, "\x00") != strings.Join(values, "\x00") {
		return errors.New("backup службы не прошёл read-back")
	}
	return nil
}

func loadServiceBackup(name string) (uint32, bool, bool, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, serviceBackupPath, registry.QUERY_VALUE|registryView())
	if errors.Is(err, registry.ErrNotExist) {
		return 0, false, false, nil
	}
	if err != nil {
		return 0, false, false, err
	}
	defer key.Close()
	values, _, err := key.GetStringsValue(name)
	if errors.Is(err, registry.ErrNotExist) {
		return 0, false, false, nil
	}
	if err != nil {
		return 0, false, false, err
	}
	startType, delayed, err := decodeServiceBackup(values)
	return startType, delayed, err == nil, err
}

func deleteServiceBackup(name string) error {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, serviceBackupPath, registry.SET_VALUE|registryView())
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer key.Close()
	err = key.DeleteValue(name)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	return err
}

func restoreServiceBackups() error {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, serviceBackupPath, registry.QUERY_VALUE|registry.SET_VALUE|registryView())
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer key.Close()
	names, err := key.ReadValueNames(-1)
	if err != nil {
		return err
	}
	manager, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer manager.Disconnect()
	var problems []error
	for _, name := range names {
		values, _, err := key.GetStringsValue(name)
		if err != nil {
			problems = append(problems, fmt.Errorf("служба %s: %w", name, err))
			continue
		}
		startType, delayed, err := decodeServiceBackup(values)
		if err != nil {
			problems = append(problems, fmt.Errorf("служба %s: %w", name, err))
			continue
		}
		service, err := manager.OpenService(name)
		if err != nil {
			problems = append(problems, fmt.Errorf("служба %s: %w", name, err))
			continue
		}
		config, err := service.Config()
		if err == nil {
			config.StartType, config.DelayedAutoStart = startType, delayed
			err = service.UpdateConfig(config)
		}
		if err == nil {
			actual, readErr := service.Config()
			if readErr != nil || actual.StartType != startType || actual.DelayedAutoStart != delayed {
				err = errors.New("исходный тип запуска службы не прошёл read-back")
			}
		}
		service.Close()
		if err != nil {
			problems = append(problems, fmt.Errorf("служба %s: %w", name, err))
			continue
		}
		if err := key.DeleteValue(name); err != nil {
			problems = append(problems, fmt.Errorf("служба %s backup: %w", name, err))
		}
	}
	return errors.Join(problems...)
}

func decodeServiceBackup(values []string) (uint32, bool, error) {
	if len(values) != 3 || values[0] != "v1" {
		return 0, false, errors.New("backup службы повреждён")
	}
	value, err := strconv.ParseUint(values[1], 10, 32)
	if err != nil || (value != windows.SERVICE_AUTO_START && value != windows.SERVICE_DEMAND_START && value != windows.SERVICE_DISABLED && value != windows.SERVICE_BOOT_START && value != windows.SERVICE_SYSTEM_START) {
		return 0, false, errors.New("backup службы содержит неверный тип запуска")
	}
	delayed, err := strconv.ParseBool(values[2])
	if err != nil {
		return 0, false, errors.New("backup службы содержит неверный delayed-флаг")
	}
	return uint32(value), delayed, nil
}

func validateServiceName(name string) error {
	if name == "" || len(name) > 256 || strings.ContainsAny(name, `\/"]`) {
		return errors.New("неверное имя службы")
	}
	for _, value := range name {
		if unicode.IsControl(value) {
			return errors.New("неверное имя службы")
		}
	}
	return nil
}

func criticalWindowsService(name string) bool {
	_, critical := map[string]struct{}{
		"appinfo": {}, "bfe": {}, "brokerinfrastructure": {}, "coremessagingregistrar": {}, "cryptsvc": {}, "dcomlaunch": {}, "dhcp": {}, "dnscache": {}, "eventlog": {}, "gpsvc": {}, "lanmanworkstation": {}, "lsm": {}, "mpssvc": {}, "nlasvc": {}, "nsi": {}, "plugplay": {}, "power": {}, "profsvc": {}, "rpceptmapper": {}, "rpcss": {}, "samss": {}, "schedule": {}, "securityhealthservice": {}, "sgrmbroker": {}, "staterepository": {}, "systemeventsbroker": {}, "trustedinstaller": {}, "usermanager": {}, "wdisystemhost": {}, "wdnisvc": {}, "windefend": {}, "winmgmt": {}, "wuauserv": {},
	}[strings.ToLower(name)]
	return critical
}

func isWindowsSystemService(binaryPath string) bool {
	value := strings.ToLower(strings.TrimSpace(binaryPath))
	if strings.Contains(value, `%systemroot%`) || strings.Contains(value, `\system32\`) || strings.Contains(value, `\syswow64\`) {
		return true
	}
	windowsDir, err := windowsDirectory()
	return err == nil && strings.Contains(value, strings.ToLower(filepath.Clean(windowsDir))+`\`)
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
