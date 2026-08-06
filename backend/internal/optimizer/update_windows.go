package optimizer

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf16"
)

func acquireUpdateLock() (func(), error) {
	return acquireNamedMutex(`Local\LuxuryOptimization-Update-v1`, "другое обновление уже выполняется")
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
	if !pendingInfo.Mode().IsRegular() || isReparsePoint(pendingInfo) || !executableInfo.Mode().IsRegular() || isReparsePoint(executableInfo) {
		return "", errors.New("self-update требует обычные файлы без reparse point")
	}
	if !filepath.IsAbs(pendingPath) || !filepath.IsAbs(executablePath) || !filepathEqual(filepath.Dir(pendingPath), filepath.Dir(executablePath)) {
		return "", errors.New("pending update должен находиться рядом с executable")
	}
	script := `$ErrorActionPreference = 'Stop'
$pidToWait = [int]$env:LUXURY_UPDATE_PID
try {
  $process = [Diagnostics.Process]::GetProcessById($pidToWait)
  if (-not $process.WaitForExit(43200000)) { throw 'parent process did not exit within 12 hours' }
} catch [ArgumentException] {}
$current = [IO.Path]::GetFullPath($env:LUXURY_UPDATE_CURRENT)
$pending = [IO.Path]::GetFullPath($env:LUXURY_UPDATE_PENDING)
if ([IO.Path]::GetDirectoryName($current) -ne [IO.Path]::GetDirectoryName($pending)) { throw 'directory mismatch' }
$backup = $current + '.update-old-' + [Guid]::NewGuid().ToString('N')
[IO.File]::Move($current, $backup)
try {
  [IO.File]::Move($pending, $current)
  [IO.File]::Delete($backup)
} catch {
  if ([IO.File]::Exists($current)) { [IO.File]::Delete($current) }
  if ([IO.File]::Exists($backup)) { [IO.File]::Move($backup, $current) }
  throw
}`
	runes := utf16.Encode([]rune(script))
	data := make([]byte, len(runes)*2)
	for index, value := range runes {
		binary.LittleEndian.PutUint16(data[index*2:], value)
	}
	environment, err := trustedPowerShellEnvironment()
	if err != nil {
		return "", err
	}
	command := exec.Command(powershellPath(), "-NoLogo", "-NoProfile", "-NonInteractive", "-EncodedCommand", base64.StdEncoding.EncodeToString(data))
	command.Env = append(environment,
		"LUXURY_UPDATE_PID="+strconv.Itoa(os.Getpid()),
		"LUXURY_UPDATE_CURRENT="+executablePath,
		"LUXURY_UPDATE_PENDING="+pendingPath,
	)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	if err := command.Start(); err != nil {
		return "", err
	}
	if err := command.Process.Release(); err != nil {
		return "", err
	}
	return "Обновление проверено и запланировано; файл будет заменён после завершения текущего процесса.", nil
}

func filepathEqual(left, right string) bool {
	return filepath.Clean(left) == filepath.Clean(right) || strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}
