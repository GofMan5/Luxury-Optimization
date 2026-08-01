package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var guidPattern = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)

var maximumPowerSettings = []PowerSetting{
	{Subgroup: "54533251-82be-4824-96c1-47b60b740d00", Setting: "bc5038f7-23e0-4960-96da-33abaf5935ec", Value: 100, Name: "Максимальное состояние CPU от сети"},
	{Subgroup: "501a4d13-42af-4429-9fd1-a8218c268e20", Setting: "ee12f906-d277-404b-b6da-e5fa1a576df5", Value: 0, Name: "PCIe Link State Power Management off"},
	{Subgroup: "2a737441-1930-4402-8d77-b2bebba308a3", Setting: "48e6b7a6-50f5-4782-a5d4-53bb8f07e226", Value: 0, Name: "USB selective suspend off"},
}

var optionalMaximumPowerSettings = []PowerSetting{
	{Subgroup: "54533251-82be-4824-96c1-47b60b740d00", Setting: "36687f9e-e3a5-4dbf-b1dc-15eb381c6863", Value: 0, Name: "CPU Energy Performance Preference 0"},
	{Subgroup: "54533251-82be-4824-96c1-47b60b740d00", Setting: "be337238-0d82-4146-a960-4f3749d470c7", Value: 2, Name: "CPU Boost mode: aggressive"},
}

var (
	wow64Once  sync.Once
	wow64Value bool
)

func isWOW64() bool {
	wow64Once.Do(func() {
		_ = windows.IsWow64Process(windows.CurrentProcess(), &wow64Value)
	})
	return wow64Value
}

func systemTool(name string) string {
	directory, err := trustedSystemDirectory()
	if err != nil {
		// An absolute non-existent path fails closed without consulting inherited env.
		directory = `C:\__gofman3_invalid_windows_directory__`
	}
	return filepath.Join(directory, name)
}

func windowsDirectory() (string, error) {
	root, err := windows.GetSystemWindowsDirectory()
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(root) {
		return "", errors.New("windows directory is not absolute")
	}
	return filepath.Clean(root), nil
}

func trustedSystemDirectory() (string, error) {
	root, err := windowsDirectory()
	if err != nil {
		return "", err
	}
	name := "System32"
	if isWOW64() {
		name = "Sysnative"
	}
	return filepath.Join(root, name), nil
}

func programDataDirectory() (string, error) {
	path, err := windows.KnownFolderPath(windows.FOLDERID_ProgramData, windows.KF_FLAG_DEFAULT)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(path) {
		return "", errors.New("ProgramData directory is not absolute")
	}
	return filepath.Clean(path), nil
}

func localTempDirectory() (string, error) {
	path, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, windows.KF_FLAG_DEFAULT)
	if err != nil {
		return "", err
	}
	return filepath.Join(path, "Temp"), nil
}

func powershellPath() string {
	return systemTool(filepath.Join("WindowsPowerShell", "v1.0", "powershell.exe"))
}

func runCommand(timeout time.Duration, path string, args ...string) ([]byte, error) {
	return runCommandWithEnvironment(timeout, nil, path, args...)
}

func runCommandWithEnvironment(timeout time.Duration, environment []string, path string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if environment != nil {
		cmd.Env = environment
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := stdout.Bytes()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return output, fmt.Errorf("таймаут команды %s", filepath.Base(path))
	}
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(string(output))
		}
		if len(message) > 1200 {
			message = message[:1200] + "…"
		}
		if message == "" {
			return output, fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
		return output, fmt.Errorf("%s: %w: %s", filepath.Base(path), err, message)
	}
	return output, nil
}

func encodedPowerShell(script string, timeout time.Duration) ([]byte, error) {
	runes := utf16.Encode([]rune(script))
	data := make([]byte, len(runes)*2)
	for i, value := range runes {
		binary.LittleEndian.PutUint16(data[i*2:], value)
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	environment, err := trustedPowerShellEnvironment()
	if err != nil {
		return nil, err
	}
	return runCommandWithEnvironment(timeout, environment, powershellPath(), "-NoLogo", "-NoProfile", "-NonInteractive", "-EncodedCommand", encoded)
}

func trustedPowerShellEnvironment() ([]string, error) {
	windowsRoot, err := windowsDirectory()
	if err != nil {
		return nil, err
	}
	// Sysnative is only a 32-bit parent alias. The spawned 64-bit PowerShell
	// must receive paths in its own System32 namespace.
	childSystemDirectory := filepath.Join(windowsRoot, "System32")
	programData, err := programDataDirectory()
	if err != nil {
		return nil, err
	}
	overrides := map[string]string{
		"SYSTEMROOT":   windowsRoot,
		"WINDIR":       windowsRoot,
		"PROGRAMDATA":  programData,
		"COMSPEC":      filepath.Join(childSystemDirectory, "cmd.exe"),
		"PATH":         childSystemDirectory + ";" + windowsRoot,
		"PATHEXT":      ".COM;.EXE;.BAT;.CMD",
		"PSMODULEPATH": filepath.Join(childSystemDirectory, "WindowsPowerShell", "v1.0", "Modules"),
	}
	temporary, err := localTempDirectory()
	if err != nil {
		return nil, err
	}
	overrides["TEMP"] = temporary
	overrides["TMP"] = temporary
	result := make([]string, 0, len(overrides))
	for name, value := range overrides {
		result = append(result, name+"="+value)
	}
	return result, nil
}

func detectHardware() (Hardware, error) {
	hardware := Hardware{GOARCH: runtime.GOARCH}
	script := `$ErrorActionPreference = 'Stop'
$PSModuleAutoLoadingPreference = 'None'
Import-Module -Name "$PSHOME\Modules\Microsoft.PowerShell.Utility\Microsoft.PowerShell.Utility.psd1" -ErrorAction Stop
Import-Module -Name "$PSHOME\Modules\CimCmdlets\CimCmdlets.psd1" -ErrorAction Stop
[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false)
$os = CimCmdlets\Get-CimInstance Win32_OperatingSystem | Select-Object Caption,Version,BuildNumber,OSArchitecture
$cpus = @(CimCmdlets\Get-CimInstance Win32_Processor | ForEach-Object {
  [pscustomobject]@{name=$_.Name;manufacturer=$_.Manufacturer;cores=$_.NumberOfCores;logical_processors=$_.NumberOfLogicalProcessors}
})
$gpus = @(CimCmdlets\Get-CimInstance Win32_VideoController | ForEach-Object {
  [pscustomobject]@{name=$_.Name;pnp_device_id=$_.PNPDeviceID;driver_version=$_.DriverVersion}
})
[pscustomobject]@{
  OS=[pscustomobject]@{Caption=$os.Caption;Version=$os.Version;BuildNumber=$os.BuildNumber;Architecture=$os.OSArchitecture}
  CPUs=$cpus
  GPUs=$gpus
  HasBattery=(@(CimCmdlets\Get-CimInstance Win32_Battery -ErrorAction SilentlyContinue).Count -gt 0)
} | Microsoft.PowerShell.Utility\ConvertTo-Json -Compress -Depth 5`
	output, err := encodedPowerShell(script, 20*time.Second)
	if err != nil {
		hardware.CPUs = []CPUInfo{{Name: strings.TrimSpace(os.Getenv("PROCESSOR_IDENTIFIER")), Logical: runtime.NumCPU()}}
		return hardware, err
	}
	var raw struct {
		OS struct {
			Caption      string
			Version      string
			BuildNumber  string
			Architecture string
		}
		CPUs       []CPUInfo
		GPUs       []GPUInfo
		HasBattery bool
	}
	if err := json.Unmarshal(bytes.TrimSpace(output), &raw); err != nil {
		return hardware, fmt.Errorf("разбор сведений о железе: %w", err)
	}
	hardware.OS = OSInfo(raw.OS)
	hardware.CPUs = raw.CPUs
	hardware.GPUs = raw.GPUs
	hardware.HasBattery = raw.HasBattery
	for i := range hardware.GPUs {
		hardware.GPUs[i].Vendor = gpuVendor(hardware.GPUs[i].PNPDeviceID, hardware.GPUs[i].Name)
	}
	return hardware, nil
}

func isAdministrator() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

func mouseParameters(set bool, values *[3]int32) error {
	action := uintptr(0x0003) // SPI_GETMOUSE
	flags := uintptr(0)
	if set {
		action = 0x0004 // SPI_SETMOUSE
		flags = 0x0002  // SPIF_SENDCHANGE; registry is handled by the transaction.
	}
	result, _, callErr := windows.NewLazySystemDLL("user32.dll").NewProc("SystemParametersInfoW").Call(
		action, 0, uintptr(unsafe.Pointer(values)), flags,
	)
	if result == 0 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return callErr
		}
		return errors.New("SystemParametersInfoW mouse operation failed")
	}
	return nil
}

func captureMouseParameters() (MouseSnapshot, error) {
	values := [3]int32{}
	if err := mouseParameters(false, &values); err != nil {
		return MouseSnapshot{}, err
	}
	return MouseSnapshot{Threshold1: values[0], Threshold2: values[1], Speed: values[2], Captured: true}, nil
}

func applyMouseParameters(snapshot MouseSnapshot) error {
	values := [3]int32{snapshot.Threshold1, snapshot.Threshold2, snapshot.Speed}
	return mouseParameters(true, &values)
}

func currentUserSID() (string, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", err
	}
	return user.User.Sid.String(), nil
}

func userSIDFromOptimizerProcess(pid uint32) (string, error) {
	sid, _, err := optimizerProcessIdentity(pid)
	return sid, err
}

func optimizerProcessIdentity(pid uint32) (string, string, error) {
	if pid == 0 {
		return "", "", errors.New("parent PID is missing")
	}
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", "", fmt.Errorf("open parent process: %w", err)
	}
	defer windows.CloseHandle(process)
	buffer := make([]uint16, 32768)
	size := uint32(len(buffer))
	if err := windows.QueryFullProcessImageName(process, 0, &buffer[0], &size); err != nil {
		return "", "", fmt.Errorf("query parent image: %w", err)
	}
	parentPath := filepath.Clean(windows.UTF16ToString(buffer[:size]))
	currentPath, err := os.Executable()
	if err != nil {
		return "", "", err
	}
	currentPath = filepath.Clean(currentPath)
	if !strings.EqualFold(parentPath, currentPath) {
		return "", "", errors.New("elevated request parent is not this optimizer executable")
	}
	var token windows.Token
	if err := windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token); err != nil {
		return "", "", fmt.Errorf("open parent token: %w", err)
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return "", "", err
	}
	localData, err := token.KnownFolderPath(windows.FOLDERID_LocalAppData, windows.KF_FLAG_DEFAULT)
	if err != nil {
		return "", "", err
	}
	return user.User.Sid.String(), filepath.Join(localData, "Temp"), nil
}

type shellExecuteInfo struct {
	Size        uint32
	Mask        uint32
	Window      uintptr
	Verb        *uint16
	File        *uint16
	Parameters  *uint16
	Directory   *uint16
	Show        int32
	Instance    uintptr
	IDList      uintptr
	Class       *uint16
	ClassKey    uintptr
	HotKey      uint32
	IconMonitor uintptr
	Process     windows.Handle
}

func runElevatedAndWait(args []string) error {
	temporary, err := localTempDirectory()
	if err != nil {
		return err
	}
	resultFile, err := os.CreateTemp(temporary, "GofMan3-result-*.json")
	if err != nil {
		return err
	}
	resultPath := resultFile.Name()
	if err := protectResultFile(resultFile); err != nil {
		resultFile.Close()
		os.Remove(resultPath)
		return err
	}
	if err := resultFile.Close(); err != nil {
		os.Remove(resultPath)
		return err
	}
	defer os.Remove(resultPath)
	args = append(args, "--result-file", resultPath)
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	verb, _ := windows.UTF16PtrFromString("runas")
	file, _ := windows.UTF16PtrFromString(exePath)
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = syscall.EscapeArg(arg)
	}
	parameters, _ := windows.UTF16PtrFromString(strings.Join(quoted, " "))
	info := shellExecuteInfo{
		Mask:       0x00000040 | 0x00000400, // SEE_MASK_NOCLOSEPROCESS | SEE_MASK_FLAG_NO_UI
		Verb:       verb,
		File:       file,
		Parameters: parameters,
		Show:       0, // hidden child; the parent TUI owns the visible terminal
	}
	info.Size = uint32(unsafe.Sizeof(info))
	proc := windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteExW")
	result, _, callErr := proc.Call(uintptr(unsafe.Pointer(&info)))
	if result == 0 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return fmt.Errorf("UAC: %w", callErr)
		}
		return errors.New("UAC: запуск отменён")
	}
	defer windows.CloseHandle(info.Process)
	if _, err := windows.WaitForSingleObject(info.Process, windows.INFINITE); err != nil {
		return err
	}
	var code uint32
	if err := windows.GetExitCodeProcess(info.Process, &code); err != nil {
		return err
	}
	if code != 0 {
		if data, readErr := os.ReadFile(resultPath); readErr == nil && len(data) <= 8192 {
			var result struct {
				Error string `json:"error"`
			}
			if json.Unmarshal(data, &result) == nil && result.Error != "" {
				return errors.New(result.Error)
			}
		}
		return fmt.Errorf("операция с правами администратора завершилась с кодом %d", code)
	}
	return nil
}

func protectResultFile(file *os.File) error {
	sid, err := currentUserSID()
	if err != nil {
		return err
	}
	descriptor, err := windows.SecurityDescriptorFromString(fmt.Sprintf(`D:P(A;;GRGW;;;%s)(A;;FA;;;SY)(A;;FA;;;BA)`, sid))
	if err != nil {
		return err
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	return windows.SetSecurityInfo(windows.Handle(file.Fd()), windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil)
}

func writeElevatedResult(path string, parentPID uint32, operationErr error) error {
	_, parentTemp, err := optimizerProcessIdentity(parentPID)
	if err != nil {
		return err
	}
	if !strings.EqualFold(filepath.Clean(filepath.Dir(path)), filepath.Clean(parentTemp)) || !strings.HasPrefix(filepath.Base(path), "GofMan3-result-") || !strings.HasSuffix(filepath.Base(path), ".json") {
		return errors.New("недопустимый путь result-файла")
	}
	pointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	handle, err := windows.CreateFile(pointer, windows.GENERIC_WRITE|windows.FILE_READ_ATTRIBUTES, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		return err
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 || info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0 {
		return errors.New("result-файл является reparse point или каталогом")
	}
	message := ""
	if operationErr != nil {
		message = operationErr.Error()
		if len(message) > 4096 {
			message = message[:4096]
		}
	}
	data, err := json.Marshal(struct {
		Error string `json:"error"`
	}{Error: message})
	if err != nil {
		return err
	}
	if _, err := windows.SetFilePointer(handle, 0, nil, windows.FILE_BEGIN); err != nil {
		return err
	}
	if err := windows.SetEndOfFile(handle); err != nil {
		return err
	}
	var written uint32
	if err := windows.WriteFile(handle, data, &written, nil); err != nil {
		return err
	}
	if int(written) != len(data) {
		return errors.New("result-файл записан не полностью")
	}
	return nil
}

func acquireOperationLock() (func(), error) {
	return acquireNamedMutex(`Global\GofMan3Optimizer-Transaction-v1`, "другая операция оптимизатора уже выполняется")
}

func acquireBoostSessionLock() (func(), error) {
	return acquireNamedMutex(`Local\GofMan3Optimizer-BoostSession-v1`, "другая игровая boost-сессия уже выполняется")
}

func acquireNamedMutex(value, busyMessage string) (func(), error) {
	name, _ := windows.UTF16PtrFromString(value)
	handle, err := windows.CreateMutex(nil, true, name)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		if handle != 0 {
			windows.CloseHandle(handle)
		}
		return nil, errors.New(busyMessage)
	}
	if err != nil {
		return nil, err
	}
	return func() {
		_ = windows.ReleaseMutex(handle)
		_ = windows.CloseHandle(handle)
	}, nil
}

func checkBoostSession(required bool) error {
	name, _ := windows.UTF16PtrFromString(`Local\GofMan3Optimizer-BoostSession-v1`)
	handle, err := windows.CreateMutex(nil, false, name)
	active := errors.Is(err, windows.ERROR_ALREADY_EXISTS)
	if handle != 0 {
		_ = windows.CloseHandle(handle)
	}
	if err != nil && !active {
		return err
	}
	if required && !active {
		return errors.New("internal boost-session lock is missing")
	}
	if !required && active {
		return errors.New("игровая boost-сессия активна; дождитесь выхода игры")
	}
	return nil
}

func activePowerGUID() (string, error) {
	output, err := runCommand(10*time.Second, systemTool("powercfg.exe"), "/getactivescheme")
	if err != nil {
		return "", err
	}
	guid := guidPattern.FindString(string(output))
	if guid == "" {
		return "", errors.New("powercfg не вернул GUID активной схемы")
	}
	return strings.ToLower(guid), nil
}

func newPowerGUID() (string, error) {
	guid, err := windows.GenerateGUID()
	if err != nil {
		return "", err
	}
	return strings.ToLower(strings.Trim(guid.String(), "{}")), nil
}

func createPerformancePlan(guid string) ([]PowerSetting, error) {
	powercfg := systemTool("powercfg.exe")
	const ultimate = "e9a42b02-d5df-448d-aa00-03f14749eb61"
	const highPerformance = "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c"
	_, err := runCommand(15*time.Second, powercfg, "/duplicatescheme", ultimate, guid)
	if err != nil {
		if exists, _ := powerSchemeExists(guid); !exists {
			_, err = runCommand(15*time.Second, powercfg, "/duplicatescheme", highPerformance, guid)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("создание отдельной схемы производительности: %w", err)
	}
	settings := availableMaximumPowerSettings(guid)
	commands := [][]string{{"/changename", guid, "GofMan3 Max Performance", "Обратимая игровая схема"}}
	for _, setting := range settings {
		commands = append(commands, []string{"/setacvalueindex", guid, setting.Subgroup, setting.Setting, fmt.Sprint(setting.Value)})
	}
	commands = append(commands, []string{"/setactive", guid})
	for _, args := range commands {
		if _, err := runCommand(15*time.Second, powercfg, args...); err != nil {
			return nil, err
		}
	}
	active, err := activePowerGUID()
	if err != nil || !strings.EqualFold(active, guid) {
		if err == nil {
			err = fmt.Errorf("активна схема %s вместо %s", active, guid)
		}
		return nil, err
	}
	return settings, nil
}

func availableMaximumPowerSettings(scheme string) []PowerSetting {
	settings := append([]PowerSetting(nil), maximumPowerSettings...)
	for _, setting := range optionalMaximumPowerSettings {
		if _, err := powerACValue(scheme, setting.Subgroup, setting.Setting); err == nil {
			settings = append(settings, setting)
		}
	}
	return settings
}

func powerACValue(scheme, subgroup, setting string) (uint32, error) {
	parseGUID := func(value string) (windows.GUID, error) {
		return windows.GUIDFromString("{" + strings.Trim(value, "{}") + "}")
	}
	schemeGUID, err := parseGUID(scheme)
	if err != nil {
		return 0, err
	}
	subgroupGUID, err := parseGUID(subgroup)
	if err != nil {
		return 0, err
	}
	settingGUID, err := parseGUID(setting)
	if err != nil {
		return 0, err
	}
	var value uint32
	result, _, _ := windows.NewLazySystemDLL("powrprof.dll").NewProc("PowerReadACValueIndex").Call(
		0,
		uintptr(unsafe.Pointer(&schemeGUID)),
		uintptr(unsafe.Pointer(&subgroupGUID)),
		uintptr(unsafe.Pointer(&settingGUID)),
		uintptr(unsafe.Pointer(&value)),
	)
	if result != 0 {
		return 0, windows.Errno(result)
	}
	return value, nil
}

func powerSchemeExists(guid string) (bool, error) {
	output, err := runCommand(10*time.Second, systemTool("powercfg.exe"), "/list")
	if err != nil {
		return false, err
	}
	for _, found := range guidPattern.FindAllString(string(output), -1) {
		if strings.EqualFold(found, guid) {
			return true, nil
		}
	}
	return false, nil
}

func restorePowerPlan(snapshot PowerSnapshot) error {
	if snapshot.PreviousGUID != "" {
		if _, err := runCommand(15*time.Second, systemTool("powercfg.exe"), "/setactive", snapshot.PreviousGUID); err != nil {
			return err
		}
	}
	if snapshot.CreatedGUID != "" {
		exists, err := powerSchemeExists(snapshot.CreatedGUID)
		if err != nil {
			return err
		}
		if exists {
			if _, err := runCommand(15*time.Second, systemTool("powercfg.exe"), "/delete", snapshot.CreatedGUID); err != nil {
				return err
			}
		}
	}
	return nil
}

func queryNetworkProperties() ([]NetProperty, error) {
	script := `$ErrorActionPreference = 'Stop'
$PSModuleAutoLoadingPreference = 'None'
Import-Module -Name "$PSHOME\Modules\Microsoft.PowerShell.Utility\Microsoft.PowerShell.Utility.psd1" -ErrorAction Stop
Import-Module -Name "$PSHOME\Modules\NetAdapter\NetAdapter.psd1" -ErrorAction Stop
[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false)
$wanted = @('*InterruptModeration','*EEE','EEE','EnergyEfficientEthernet','AdvancedEEE','ULPMode','GigaLite')
$result = @()
NetAdapter\Get-NetAdapter -Physical -ErrorAction Stop | Where-Object { $_.NdisPhysicalMedium -eq 14 } | ForEach-Object {
  $adapter = $_
  NetAdapter\Get-NetAdapterAdvancedProperty -Name $adapter.Name -AllProperties -ErrorAction SilentlyContinue | ForEach-Object {
    if ($wanted -contains [string]$_.RegistryKeyword -and $null -ne $_.RegistryValue) {
      $values = @($_.RegistryValue | ForEach-Object { [string]$_ })
      $result += [pscustomobject]@{interface_guid=$adapter.InterfaceGuid.ToString();adapter_name=$adapter.Name;keyword=[string]$_.RegistryKeyword;values=$values}
    }
  }
}
Microsoft.PowerShell.Utility\ConvertTo-Json -InputObject @($result) -Compress -Depth 4`
	output, err := encodedPowerShell(script, 25*time.Second)
	if err != nil {
		return nil, err
	}
	properties, err := decodeNetworkProperties(output)
	if err != nil {
		return nil, fmt.Errorf("разбор параметров Ethernet: %w", err)
	}
	return properties, nil
}

func decodeNetworkProperties(output []byte) ([]NetProperty, error) {
	var properties []NetProperty
	if len(bytes.TrimSpace(output)) == 0 {
		return properties, nil
	}
	if err := json.Unmarshal(bytes.TrimSpace(output), &properties); err != nil {
		return nil, err
	}
	return properties, nil
}

func networkChanges(properties []NetProperty) []NetProperty {
	result := make([]NetProperty, 0, len(properties))
	for _, property := range properties {
		if networkPropertyNeedsChange(property.Values) {
			result = append(result, property)
		}
	}
	return result
}

func networkPropertyNeedsChange(values []string) bool {
	for _, value := range values {
		if value != "0" {
			return true
		}
	}
	return false
}

func setNetworkProperties(properties []NetProperty, restore bool) error {
	if len(properties) == 0 {
		return nil
	}
	payload, err := json.Marshal(properties)
	if err != nil {
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(payload)
	script := `$ErrorActionPreference = 'Stop'
$PSModuleAutoLoadingPreference = 'None'
Import-Module -Name "$PSHOME\Modules\Microsoft.PowerShell.Utility\Microsoft.PowerShell.Utility.psd1" -ErrorAction Stop
Import-Module -Name "$PSHOME\Modules\NetAdapter\NetAdapter.psd1" -ErrorAction Stop
$json = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($env:GOFMAN_NET_PROPERTIES))
$items = @($json | Microsoft.PowerShell.Utility\ConvertFrom-Json)
foreach ($item in $items) {
  $adapter = NetAdapter\Get-NetAdapter -IncludeHidden | Where-Object { $_.InterfaceGuid -eq [Guid]$item.interface_guid } | Select-Object -First 1
  if ($null -eq $adapter) { throw "Network adapter not found: $($item.interface_guid)" }
  $values = if ($env:GOFMAN_NET_RESTORE -eq '1') { @($item.values) } else { @('0') }
  NetAdapter\Set-NetAdapterAdvancedProperty -Name $adapter.Name -RegistryKeyword $item.keyword -RegistryValue $values -NoRestart -ErrorAction Stop
}`
	runes := utf16.Encode([]rune(script))
	data := make([]byte, len(runes)*2)
	for i, value := range runes {
		binary.LittleEndian.PutUint16(data[i*2:], value)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, powershellPath(), "-NoLogo", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(data))
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	mode := "0"
	if restore {
		mode = "1"
	}
	environment, err := trustedPowerShellEnvironment()
	if err != nil {
		return err
	}
	command.Env = append(environment, "GOFMAN_NET_PROPERTIES="+encoded, "GOFMAN_NET_RESTORE="+mode)
	output, err := command.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errors.New("таймаут применения параметров Ethernet")
	}
	if err != nil {
		return fmt.Errorf("параметры Ethernet: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func appDataDir() (string, error) {
	base, err := programDataDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "GofMan3 Optimizer"), nil
}
