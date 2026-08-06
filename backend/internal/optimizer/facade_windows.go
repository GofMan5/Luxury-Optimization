//go:build windows

package optimizer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func (s *Service) ApplyProfile(profile string) (MutationResult, error) {
	if err := applyCommand([]string{"--profile", profile, "--yes", "--quiet", "--require-checkpoint"}); err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Changed: true, Message: "Профиль применён и проверен."}, nil
}

func (s *Service) ApplyTweak(id string) (MutationResult, error) {
	s.tweakMu.Lock()
	defer s.tweakMu.Unlock()
	if !tweakIDPattern.MatchString(id) {
		return MutationResult{}, errors.New("неверный ID твика")
	}
	plan, err := s.Plan(profileMaximum)
	if err != nil {
		return MutationResult{}, err
	}
	found, changed := false, false
	for _, item := range plan.Items {
		if item.ID == id {
			found, changed = true, item.Changed
			break
		}
	}
	if !found {
		return MutationResult{}, fmt.Errorf("твик %q недоступен на этой системе", id)
	}
	if !changed {
		return MutationResult{Message: "Твик уже применён; изменений не требуется."}, nil
	}
	backupID := backupIDNow()
	args := []string{"tweak", "apply", "--id", id, "--backup-id", backupID, "--parent-pid", parentPIDArg()}
	if err := runElevatedAndWait(args); err != nil {
		return MutationResult{}, err
	}
	if err := setTweakMarker(id, backupID); err != nil {
		rollbackErr := runElevatedAndWait([]string{"tweak", "restore", "--id", id, "--backup-id", backupID, "--parent-pid", parentPIDArg()})
		if rollbackErr != nil {
			return MutationResult{Changed: true, Artifact: backupID}, fmt.Errorf("твик применён, но быстрый откат не сохранён: %w; аварийный откат: %v", err, rollbackErr)
		}
		return MutationResult{}, fmt.Errorf("не удалось сохранить быстрый откат; твик автоматически отменён: %w", err)
	}
	return MutationResult{Changed: true, Message: "Твик применён и проверен. Отдельная резервная копия готова к откату.", Artifact: backupID}, nil
}

func (s *Service) RestoreTweak(id string) (MutationResult, error) {
	s.tweakMu.Lock()
	defer s.tweakMu.Unlock()
	if !tweakIDPattern.MatchString(id) {
		return MutationResult{}, errors.New("неверный ID твика")
	}
	markers, err := loadTweakMarkers()
	if err != nil {
		return MutationResult{}, err
	}
	marker, ok := markers.Backups[id]
	if !ok || !backupIDPattern.MatchString(marker.BackupID) {
		return MutationResult{}, errors.New("для этого твика нет отдельной точки отката")
	}
	if err := runElevatedAndWait([]string{"tweak", "restore", "--id", id, "--backup-id", marker.BackupID, "--parent-pid", parentPIDArg()}); err != nil {
		return MutationResult{}, err
	}
	if err := removeTweakMarker(id); err != nil {
		return MutationResult{Changed: true, Message: "Твик восстановлен, но локальная кнопка отката не обновилась: " + err.Error(), Artifact: marker.BackupID}, nil
	}
	return MutationResult{Changed: true, Message: "Твик возвращён в исходное состояние и проверен.", Artifact: marker.BackupID}, nil
}

func (s *Service) CheckpointStatus(profile string) (CheckpointStatus, error) {
	if _, err := profileByID(profile); err != nil {
		return CheckpointStatus{}, err
	}
	return loadCheckpointMarker(profile)
}

func (s *Service) CreateCheckpoint(profile string) (CheckpointStatus, error) {
	if _, err := profileByID(profile); err != nil {
		return CheckpointStatus{}, err
	}
	id := time.Now().UTC().Format("20060102T150405.000000000Z")
	if err := runElevatedAndWait([]string{"checkpoint", "--profile", profile, "--id", id, "--quiet", "--parent-pid", strconv.Itoa(os.Getpid())}); err != nil {
		return CheckpointStatus{}, err
	}
	status := CheckpointStatus{Ready: true, Profile: profile, CreatedAt: time.Now().UTC()}
	status.ExpiresAt = status.CreatedAt.Add(24 * time.Hour)
	if err := saveCheckpointMarker(id, status); err != nil {
		return CheckpointStatus{}, err
	}
	return status, nil
}

func (s *Service) Restore(backupID string) (MutationResult, error) {
	args := []string{"--yes", "--quiet"}
	if backupID != "" {
		args = append(args, "--id", backupID)
	}
	if err := restoreCommand(args); err != nil {
		return MutationResult{}, err
	}
	if backupID != "" {
		_ = removeTweakMarkerByBackup(backupID)
	}
	return MutationResult{Changed: true, Message: "Исходные настройки восстановлены и проверены."}, nil
}

func (s *Service) Backups() ([]BackupSummary, error) {
	if !isAdministrator() {
		return nil, errors.New("для чтения sealed backups нужны права администратора")
	}
	sid, err := currentUserSID()
	if err != nil {
		return nil, err
	}
	return listBackupSummaries(sid)
}

func (s *Service) SetService(name string, enabled bool) (MutationResult, error) {
	return setServiceEnabledViaElevation(name, enabled)
}

func setServiceEnabledViaElevation(name string, enabled bool) (MutationResult, error) {
	if err := servicesCommand([]string{"set", "--name", name, "--enabled=" + strconv.FormatBool(enabled), "--yes", "--quiet"}); err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Changed: true, Message: "Настройка запуска службы сохранена и проверена."}, nil
}

func (s *Service) SystemRestorePoints() ([]SystemRestorePoint, error) {
	const script = `$ErrorActionPreference = 'Stop'
Import-Module -Name "$PSHOME\Modules\CimCmdlets\CimCmdlets.psd1" -ErrorAction Stop
$items = @(
  Get-CimInstance -Namespace 'root/default' -ClassName 'SystemRestore' -ErrorAction Stop |
    Sort-Object -Property SequenceNumber -Descending |
    ForEach-Object {
      $created = [Management.ManagementDateTimeConverter]::ToDateTime([string]$_.CreationTime).ToUniversalTime().ToString('o')
      [pscustomobject]@{
        sequence_number = [uint32]$_.SequenceNumber
        description = [string]$_.Description
        created_at = $created
        restore_point_type = [uint32]$_.RestorePointType
      }
    }
)
[Console]::Out.Write((ConvertTo-Json -Compress -InputObject $items))`
	output, err := encodedPowerShell(script, 15*time.Second)
	if err != nil {
		return nil, err
	}
	return decodeSystemRestorePoints(output)
}

func (s *Service) OpenSystemRestore() (MutationResult, error) {
	command := exec.Command(systemTool("rstrui.exe"))
	if err := command.Start(); err != nil {
		return MutationResult{}, err
	}
	go func() { _ = command.Wait() }()
	return MutationResult{Message: "Открыто штатное восстановление Windows."}, nil
}

func decodeSystemRestorePoints(output []byte) ([]SystemRestorePoint, error) {
	if value := strings.TrimSpace(string(output)); value == "" || value == "null" {
		return []SystemRestorePoint{}, nil
	}
	var points []SystemRestorePoint
	if err := json.Unmarshal(output, &points); err != nil {
		return nil, err
	}
	return points, nil
}

type checkpointMarker struct {
	ID        string    `json:"id"`
	Profile   string    `json:"profile"`
	CreatedAt time.Time `json:"created_at"`
}

type tweakMarker struct {
	BackupID string    `json:"backup_id"`
	Created  time.Time `json:"created_at"`
}

type tweakMarkers struct {
	Version int                    `json:"version"`
	Backups map[string]tweakMarker `json:"backups"`
}

func checkpointMarkerPath() (string, error) {
	base, err := localAppDataDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Luxury Optimization", "checkpoint.json"), nil
}

func tweakMarkerPath() (string, error) {
	base, err := localAppDataDirectory()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Luxury Optimization", "tweak-backups.json"), nil
}

func saveCheckpointMarker(id string, status CheckpointStatus) error {
	path, err := checkpointMarkerPath()
	if err != nil {
		return err
	}
	data, err := json.Marshal(checkpointMarker{ID: id, Profile: status.Profile, CreatedAt: status.CreatedAt})
	if err != nil {
		return err
	}
	return writeUserState(path, "checkpoint-*.tmp", data)
}

func writeUserState(path, pattern string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), pattern)
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err = file.Chmod(0o600); err == nil {
		_, err = file.Write(data)
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func loadTweakMarkers() (tweakMarkers, error) {
	store := tweakMarkers{Version: 1, Backups: make(map[string]tweakMarker)}
	path, err := tweakMarkerPath()
	if err != nil {
		return store, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return store, err
	}
	if len(data) == 0 || len(data) > 64<<10 || json.Unmarshal(data, &store) != nil || store.Version != 1 || store.Backups == nil {
		return tweakMarkers{}, errors.New("локальное состояние быстрых откатов повреждено")
	}
	for id, marker := range store.Backups {
		if !tweakIDPattern.MatchString(id) || !backupIDPattern.MatchString(marker.BackupID) {
			return tweakMarkers{}, errors.New("локальное состояние быстрых откатов содержит неверный ID")
		}
	}
	return store, nil
}

func saveTweakMarkers(store tweakMarkers) error {
	path, err := tweakMarkerPath()
	if err != nil {
		return err
	}
	data, err := json.Marshal(store)
	if err != nil {
		return err
	}
	return writeUserState(path, "tweak-backups-*.tmp", data)
}

func setTweakMarker(id, backupID string) error {
	store, err := loadTweakMarkers()
	if err != nil {
		return err
	}
	store.Backups[id] = tweakMarker{BackupID: backupID, Created: time.Now().UTC()}
	return saveTweakMarkers(store)
}

func removeTweakMarker(id string) error {
	store, err := loadTweakMarkers()
	if err != nil {
		return err
	}
	delete(store.Backups, id)
	return saveTweakMarkers(store)
}

func removeTweakMarkerByBackup(backupID string) error {
	store, err := loadTweakMarkers()
	if err != nil {
		return err
	}
	for id, marker := range store.Backups {
		if marker.BackupID == backupID {
			delete(store.Backups, id)
		}
	}
	return saveTweakMarkers(store)
}

func decorateTweakRestoreState(plan *Plan) {
	for index := range plan.Items {
		plan.Items[index].ManualAvailable = true
	}
	store, err := loadTweakMarkers()
	if err != nil {
		plan.Warnings = append(plan.Warnings, "Не удалось прочитать быстрые откаты твиков: "+err.Error())
		return
	}
	for index := range plan.Items {
		_, plan.Items[index].RestoreAvailable = store.Backups[plan.Items[index].ID]
	}
}

func loadCheckpointMarker(profile string) (CheckpointStatus, error) {
	path, err := checkpointMarkerPath()
	if err != nil {
		return CheckpointStatus{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return CheckpointStatus{Profile: profile}, nil
	}
	if err != nil || len(data) > 4096 {
		return CheckpointStatus{Profile: profile}, err
	}
	var marker checkpointMarker
	if json.Unmarshal(data, &marker) != nil || !backupIDPattern.MatchString(marker.ID) || marker.Profile != profile {
		return CheckpointStatus{Profile: profile}, nil
	}
	expires := marker.CreatedAt.Add(24 * time.Hour)
	return CheckpointStatus{Ready: time.Now().UTC().Before(expires), Profile: profile, CreatedAt: marker.CreatedAt, ExpiresAt: expires}, nil
}

func launchRecoveryCLI(args []string) (int, error) {
	executable, err := os.Executable()
	if err != nil {
		return 0, err
	}
	command := exec.Command(executable, args...)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := command.Start(); err != nil {
		return 0, err
	}
	go func() { _ = command.Wait() }()
	return command.Process.Pid, nil
}
