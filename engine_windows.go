package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var backupFilePattern = regexp.MustCompile(`^\d{8}T\d{6}\.\d{9}Z\.json$`)

func buildPlan(profileID string) (Plan, error) {
	profile, err := profileByID(profileID)
	if err != nil {
		return Plan{}, err
	}
	if err := validateProfile(profile); err != nil {
		return Plan{}, err
	}
	hardware, hardwareErr := detectHardware()
	plan := Plan{Profile: profile, Hardware: hardware}
	if hardwareErr != nil {
		plan.Warnings = append(plan.Warnings, "Не удалось полностью прочитать железо: "+hardwareErr.Error())
	}
	if profile.ID == profileMaximum && hardware.HasBattery {
		plan.Warnings = append(plan.Warnings, "Обнаружена батарея: максимальный профиль увеличит нагрев и расход энергии от сети.")
	}
	for _, change := range profile.Changes {
		matches, current, err := registryMatches(change)
		if err != nil {
			return plan, fmt.Errorf("план %s: %w", change.ID, err)
		}
		plan.Items = append(plan.Items, PlanItem{Category: change.Category, Name: change.Description, Current: current, Desired: formatDesired(change), Changed: !matches})
	}
	if profile.PerformancePlan {
		current, err := activePowerGUID()
		if err != nil {
			current = "не удалось прочитать: " + err.Error()
		}
		plan.Items = append(plan.Items, PlanItem{Category: "Питание", Name: "Создать отдельную обратимую схему максимальной производительности", Current: current, Desired: "новая схема GofMan3 Max Performance", Changed: true})
		for _, setting := range optionalMaximumPowerSettings {
			if value, readErr := powerACValue(current, setting.Subgroup, setting.Setting); readErr == nil {
				plan.Items = append(plan.Items, PlanItem{Category: "Питание", Name: setting.Name, Current: fmt.Sprint(value), Desired: fmt.Sprint(setting.Value), Changed: value != setting.Value})
			}
		}
	}
	if profile.NetworkLatency {
		properties, err := queryNetworkProperties()
		if err != nil {
			plan.Warnings = append(plan.Warnings, "Ethernet low-latency недоступен и будет пропущен: "+err.Error())
		} else {
			for _, property := range networkChanges(properties) {
				plan.Items = append(plan.Items, PlanItem{Category: "Ethernet", Name: property.AdapterName + ": " + property.Keyword, Current: strings.Join(property.Values, ","), Desired: "0 (отключено)", Changed: true})
			}
		}
	}
	return plan, nil
}

func applyProfile(profileID string, boostSession bool) (string, error) {
	if !isAdministrator() {
		return "", errors.New("для применения нужны права администратора")
	}
	releaseLock, err := acquireOperationLock()
	if err != nil {
		return "", err
	}
	defer releaseLock()
	if err := checkBoostSession(boostSession); err != nil {
		return "", err
	}
	profile, err := profileByID(profileID)
	if err != nil {
		return "", err
	}
	if err := validateProfile(profile); err != nil {
		return "", err
	}
	if registryUserSID == "" {
		sid, sidErr := currentUserSID()
		if sidErr != nil {
			return "", sidErr
		}
		if err := setRegistryUserSID(sid); err != nil {
			return "", err
		}
	}
	defer func() { _ = setRegistryUserSID("") }()
	stateDir, err := ensureStateDir()
	if err != nil {
		return "", err
	}
	backup := Backup{
		FormatVersion:  1,
		CatalogVersion: 1,
		ID:             time.Now().UTC().Format("20060102T150405.000000000Z"),
		CreatedAt:      time.Now().UTC(),
		Profile:        profile.ID,
		TargetUserSID:  registryUserSID,
		Status:         "prepared",
	}
	backup.Path = filepath.Join(stateDir, backup.ID+".json")
	for _, change := range profile.Changes {
		snapshot, err := snapshotRegistry(change)
		if err != nil {
			return "", fmt.Errorf("резервная копия %s: %w", change.ID, err)
		}
		backup.Registry = append(backup.Registry, snapshot)
	}
	backup.CatalogDigest = backupTargetDigest(backup.Registry)
	if profileChangesMouse(profile.Changes) {
		backup.Mouse, err = captureMouseParameters()
		if err != nil {
			return "", err
		}
	}
	if profile.PerformancePlan {
		backup.Power.PreviousGUID, err = activePowerGUID()
		if err != nil {
			return "", err
		}
		backup.Power.CreatedGUID, err = newPowerGUID()
		if err != nil {
			return "", err
		}
		// Journal the destination GUID before powercfg can create or activate it.
		backup.PowerApplied = true
	}
	if profile.NetworkLatency {
		properties, netErr := queryNetworkProperties()
		if netErr == nil {
			backup.Network = networkChanges(properties)
		}
	}
	if err := saveBackup(&backup); err != nil {
		return "", err
	}

	fail := func(cause error) (string, error) {
		backup.Error = cause.Error()
		backup.Status = "failed"
		_ = saveBackup(&backup)
		rollbackErr := rollbackBackup(&backup)
		if rollbackErr != nil {
			return backup.Path, fmt.Errorf("%w; автоматический откат: %v", cause, rollbackErr)
		}
		return backup.Path, fmt.Errorf("%w; применённые изменения автоматически отменены", cause)
	}

	if profile.PerformancePlan {
		backup.Power.Settings, err = createPerformancePlan(backup.Power.CreatedGUID)
		if err != nil {
			return fail(err)
		}
		if err := saveBackup(&backup); err != nil {
			return fail(err)
		}
	}
	for i, change := range profile.Changes {
		backup.AppliedRegistry = i + 1
		if err := saveBackup(&backup); err != nil {
			return fail(err)
		}
		if err := applyRegistry(change); err != nil {
			return fail(fmt.Errorf("%s: %w", change.ID, err))
		}
	}
	if backup.Mouse.Captured {
		backup.Mouse.Applied = true
		if err := saveBackup(&backup); err != nil {
			return fail(err)
		}
		if err := applyMouseParameters(MouseSnapshot{Threshold1: 0, Threshold2: 0, Speed: 0}); err != nil {
			return fail(err)
		}
	}
	if len(backup.Network) > 0 {
		backup.NetworkApplied = true
		if err := saveBackup(&backup); err != nil {
			return fail(err)
		}
		if err := setNetworkProperties(backup.Network, false); err != nil {
			return fail(err)
		}
	}
	if err := verifyApplied(profile, backup); err != nil {
		return fail(err)
	}
	backup.Status = "applied"
	backup.Error = ""
	if err := saveBackup(&backup); err != nil {
		return fail(err)
	}
	return backup.Path, nil
}

func verifyApplied(profile Profile, backup Backup) error {
	for _, change := range profile.Changes {
		matches, _, err := registryMatches(change)
		if err != nil {
			return err
		}
		if !matches {
			return fmt.Errorf("проверка %s не прошла", change.ID)
		}
	}
	if backup.Mouse.Applied {
		current, err := captureMouseParameters()
		if err != nil {
			return err
		}
		if current.Threshold1 != 0 || current.Threshold2 != 0 || current.Speed != 0 {
			return errors.New("live-параметры ускорения мыши не применены")
		}
	}
	if backup.PowerApplied {
		active, err := activePowerGUID()
		if err != nil {
			return err
		}
		if !strings.EqualFold(active, backup.Power.CreatedGUID) {
			return fmt.Errorf("новая схема питания не активна")
		}
		for _, setting := range backup.Power.Settings {
			value, err := powerACValue(active, setting.Subgroup, setting.Setting)
			if err != nil {
				return fmt.Errorf("проверка AC-настройки питания %s: %w", setting.Setting, err)
			}
			if value != setting.Value {
				return fmt.Errorf("AC-настройка питания %s: получено %d, ожидалось %d", setting.Setting, value, setting.Value)
			}
		}
	}
	if backup.NetworkApplied {
		current, err := queryNetworkProperties()
		if err != nil {
			return err
		}
		lookup := make(map[string][]string, len(current))
		for _, property := range current {
			lookup[strings.ToLower(property.InterfaceGUID)+"|"+strings.ToLower(property.Keyword)] = property.Values
		}
		for _, property := range backup.Network {
			values, ok := lookup[strings.ToLower(property.InterfaceGUID)+"|"+strings.ToLower(property.Keyword)]
			if !ok || len(values) == 0 || networkPropertyNeedsChange(values) {
				return fmt.Errorf("не подтверждён Ethernet параметр %s/%s", property.AdapterName, property.Keyword)
			}
		}
	}
	return nil
}

func rollbackBackup(backup *Backup) error {
	var problems []error
	backup.Status = "rolling_back"
	if err := saveBackup(backup); err != nil {
		problems = append(problems, err)
	}
	if backup.NetworkApplied {
		if err := setNetworkProperties(backup.Network, true); err != nil {
			problems = append(problems, err)
		} else {
			backup.NetworkApplied = false
			if err := saveBackup(backup); err != nil {
				problems = append(problems, err)
			}
		}
	}
	limit := backup.AppliedRegistry
	if limit > len(backup.Registry) {
		limit = len(backup.Registry)
	}
	registryFailed := false
	for i := limit - 1; i >= 0; i-- {
		if err := restoreRegistry(backup.Registry[i]); err != nil {
			registryFailed = true
			problems = append(problems, fmt.Errorf("откат %s: %w", backup.Registry[i].Change.ID, err))
		}
	}
	if !registryFailed {
		backup.AppliedRegistry = 0
		if err := saveBackup(backup); err != nil {
			problems = append(problems, err)
		}
	}
	if backup.Mouse.Applied && !registryFailed {
		if err := applyMouseParameters(backup.Mouse); err != nil {
			problems = append(problems, err)
		} else {
			backup.Mouse.Applied = false
			if err := saveBackup(backup); err != nil {
				problems = append(problems, err)
			}
		}
	}
	if backup.PowerApplied {
		if err := restorePowerPlan(backup.Power); err != nil {
			problems = append(problems, err)
		} else {
			backup.PowerApplied = false
			if err := saveBackup(backup); err != nil {
				problems = append(problems, err)
			}
		}
	}
	if len(problems) == 0 {
		backup.Status = "rolled_back"
	} else {
		backup.Status = "rollback_failed"
	}
	if err := saveBackup(backup); err != nil {
		problems = append(problems, err)
	}
	return errors.Join(problems...)
}

func profileChangesMouse(changes []RegChange) bool {
	wanted := map[string]bool{"mouse-threshold-1": false, "mouse-threshold-2": false, "mouse-speed": false}
	for _, change := range changes {
		if _, ok := wanted[change.ID]; ok {
			wanted[change.ID] = true
		}
	}
	return wanted["mouse-threshold-1"] && wanted["mouse-threshold-2"] && wanted["mouse-speed"]
}

func restoreLatest(requestSID string, boostSession bool) (string, error) {
	if !isAdministrator() {
		return "", errors.New("для восстановления нужны права администратора")
	}
	if !sidPattern.MatchString(requestSID) {
		return "", errors.New("не удалось подтвердить SID инициатора восстановления")
	}
	releaseLock, err := acquireOperationLock()
	if err != nil {
		return "", err
	}
	defer releaseLock()
	if err := checkBoostSession(boostSession); err != nil {
		return "", err
	}
	backup, err := loadLatestBackup(requestSID)
	if err != nil {
		return "", err
	}
	if backup.Status != "applied" && backup.Status != "prepared" && backup.Status != "failed" && backup.Status != "rolling_back" && backup.Status != "rollback_failed" {
		return backup.Path, fmt.Errorf("состояние %q нельзя восстанавливать", backup.Status)
	}
	if err := validateBackup(backup); err != nil {
		return backup.Path, err
	}
	if err := setRegistryUserSID(backup.TargetUserSID); err != nil {
		return backup.Path, err
	}
	defer func() { _ = setRegistryUserSID("") }()
	return backup.Path, rollbackBackup(&backup)
}

func ensureStateDir() (string, error) {
	base, err := appDataDir()
	if err != nil {
		return "", err
	}
	if err := ensureSecureDirectory(base); err != nil {
		return "", err
	}
	dir := filepath.Join(base, "state")
	if err := ensureSecureDirectory(dir); err != nil {
		return "", err
	}
	return dir, nil
}

func saveBackup(backup *Backup) error {
	if backup.Path == "" {
		return errors.New("путь резервной копии пуст")
	}
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(backup.Path), filepath.Base(backup.Path)+".*.tmp")
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
	production := isProductionBackupPath(backup.Path)
	digest := backupDigest(data)
	if production {
		if !isAdministrator() {
			return errors.New("production backup можно сохранить только elevated-процессом")
		}
		if err := prepareBackupSeal(backup.ID, digest); err != nil {
			return err
		}
	}
	if err := os.Rename(temporary, backup.Path); err != nil {
		return err
	}
	if production {
		if err := commitBackupSeal(backup.ID, digest); err != nil {
			return err
		}
	}
	return nil
}

func loadLatestBackup(targetSID string) (Backup, error) {
	base, err := appDataDir()
	if err != nil {
		return Backup{}, err
	}
	dir := filepath.Join(base, "state")
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || isReparsePoint(info) {
		return Backup{}, errors.New("каталог резервных копий отсутствует или небезопасен")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Backup{}, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		entryInfo, infoErr := entry.Info()
		if infoErr == nil && !entry.IsDir() && !isReparsePoint(entryInfo) && backupFilePattern.MatchString(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	if len(names) == 0 {
		return Backup{}, errors.New("резервные копии не найдены")
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	for _, name := range names {
		path := filepath.Join(dir, name)
		id := strings.TrimSuffix(name, ".json")
		if err := verifyBackupSeal(path, id); err != nil {
			return Backup{}, fmt.Errorf("недоверенный backup %s: %w", name, err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return Backup{}, err
		}
		var backup Backup
		if err := json.Unmarshal(data, &backup); err != nil {
			return Backup{}, err
		}
		if backup.TargetUserSID != targetSID || backup.Status == "rolled_back" {
			continue
		}
		if backup.ID != id {
			return Backup{}, errors.New("ID backup не совпадает с именем файла")
		}
		backup.Path = path
		return backup, nil
	}
	return Backup{}, errors.New("восстанавливаемые резервные копии этого пользователя не найдены")
}

func validateBackup(backup Backup) error {
	if backup.FormatVersion != 1 {
		return fmt.Errorf("неподдерживаемый формат backup %d", backup.FormatVersion)
	}
	if backup.CatalogVersion != 1 {
		return fmt.Errorf("неподдерживаемая версия каталога backup %d", backup.CatalogVersion)
	}
	if backup.CatalogDigest == "" || backup.CatalogDigest != backupTargetDigest(backup.Registry) {
		return errors.New("каталог целей backup повреждён")
	}
	if !sidPattern.MatchString(backup.TargetUserSID) {
		return errors.New("backup содержит неверный SID пользователя")
	}
	profile, err := profileByID(backup.Profile)
	if err != nil {
		return err
	}
	allowed := make(map[string]RegChange, len(profile.Changes))
	for _, change := range profile.Changes {
		allowed[change.ID] = change
	}
	// Keep any removed v1 target in this map in future releases so old backups remain restorable.
	seen := make(map[string]bool, len(backup.Registry))
	for i, snapshot := range backup.Registry {
		expected, ok := allowed[snapshot.Change.ID]
		if !ok || seen[snapshot.Change.ID] ||
			!strings.EqualFold(snapshot.Change.Hive, expected.Hive) ||
			!strings.EqualFold(snapshot.Change.Path, expected.Path) ||
			!strings.EqualFold(snapshot.Change.Name, expected.Name) {
			return fmt.Errorf("backup содержит недопустимую цель на позиции %d", i)
		}
		seen[snapshot.Change.ID] = true
	}
	if backup.Power.PreviousGUID != "" && !guidPattern.MatchString(backup.Power.PreviousGUID) {
		return errors.New("backup содержит неверный GUID питания")
	}
	if backup.Power.CreatedGUID != "" && !guidPattern.MatchString(backup.Power.CreatedGUID) {
		return errors.New("backup содержит неверный GUID созданной схемы")
	}
	allowedKeywords := map[string]bool{"*interruptmoderation": true, "*eee": true, "eee": true, "energyefficientethernet": true, "advancedeee": true, "ulpmode": true, "gigalite": true}
	for _, property := range backup.Network {
		if !guidPattern.MatchString(property.InterfaceGUID) || !allowedKeywords[strings.ToLower(property.Keyword)] {
			return errors.New("backup содержит недопустимый Ethernet параметр")
		}
	}
	return nil
}

func backupTargetDigest(snapshots []RegSnapshot) string {
	hash := sha256.New()
	for _, snapshot := range snapshots {
		change := snapshot.Change
		_, _ = fmt.Fprintf(hash, "%s\x00%s\x00%s\x00%s\n", change.ID, strings.ToUpper(change.Hive), strings.ToLower(change.Path), strings.ToLower(change.Name))
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil))
}
