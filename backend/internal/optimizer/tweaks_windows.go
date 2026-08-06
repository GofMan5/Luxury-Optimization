package optimizer

import (
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var tweakIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,127}$`)

type tweakTarget struct {
	profile   string
	change    *RegChange
	power     *PowerSetting
	network   *NetProperty
	fullPower bool
}

func networkTweakID(property NetProperty) string {
	key := strings.ToLower(strings.Trim(property.InterfaceGUID, "{}")) + "\x00" + strings.ToLower(property.Keyword)
	sum := sha256.Sum256([]byte(key))
	return fmt.Sprintf("ethernet-%x", sum[:10])
}

func powerTweakID(setting PowerSetting) string { return "power-" + strings.ToLower(setting.Setting) }

func resolveTweak(id string) (tweakTarget, error) {
	if !tweakIDPattern.MatchString(id) {
		return tweakTarget{}, errors.New("неверный ID твика")
	}
	for _, profileID := range []string{profileLite, profileMedium, profileMaximum} {
		profile, _ := profileByID(profileID)
		for _, change := range profile.Changes {
			if change.ID == id {
				copy := change
				return tweakTarget{profile: profileID, change: &copy}, nil
			}
		}
	}
	if id == "power-plan" {
		return tweakTarget{profile: profileMaximum, fullPower: true}, nil
	}
	if strings.HasPrefix(id, "power-") {
		active, err := activePowerGUID()
		if err != nil {
			return tweakTarget{}, err
		}
		settings, err := availableMaximumPowerSettings(active)
		if err != nil {
			return tweakTarget{}, err
		}
		for _, setting := range settings {
			if powerTweakID(setting) != id {
				continue
			}
			copy := setting
			return tweakTarget{profile: profileMaximum, power: &copy}, nil
		}
	}
	if strings.HasPrefix(id, "ethernet-") {
		properties, err := queryNetworkProperties()
		if err != nil {
			return tweakTarget{}, err
		}
		var found *NetProperty
		for _, property := range properties {
			if networkTweakID(property) != id {
				continue
			}
			if found != nil {
				return tweakTarget{}, errors.New("неоднозначный ID Ethernet-твика")
			}
			copy := property
			found = &copy
		}
		if found != nil {
			return tweakTarget{profile: profileMaximum, network: found}, nil
		}
	}
	return tweakTarget{}, fmt.Errorf("твик %q недоступен на этой системе", id)
}

func applyTweak(id, backupID string) (string, error) {
	if !isAdministrator() {
		return "", errors.New("для применения твика нужны права администратора")
	}
	if !backupIDPattern.MatchString(backupID) {
		return "", errors.New("неверный ID резервной копии твика")
	}
	releaseLock, err := acquireOperationLock()
	if err != nil {
		return "", err
	}
	defer releaseLock()
	if err := checkBoostSession(false); err != nil {
		return "", err
	}
	if registryUserSID == "" {
		return "", errors.New("не задан пользователь твика")
	}
	target, err := resolveTweak(id)
	if err != nil {
		return "", err
	}
	stateDir, err := ensureStateDir()
	if err != nil {
		return "", err
	}
	backup := Backup{
		FormatVersion: 1, CatalogVersion: 1, ID: backupID, CreatedAt: time.Now().UTC(),
		Profile: target.profile, TweakID: id, TargetUserSID: registryUserSID, Status: "prepared",
	}
	backup.Path = filepath.Join(stateDir, backup.ID+".json")
	if _, err := os.Lstat(backup.Path); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return "", errors.New("резервная копия твика уже существует")
		}
		return "", err
	}
	if target.change != nil {
		snapshot, err := snapshotRegistry(*target.change)
		if err != nil {
			return "", fmt.Errorf("резервная копия %s: %w", id, err)
		}
		backup.Registry = []RegSnapshot{snapshot}
		if isMouseTweak(id) {
			backup.Mouse, err = captureMouseParameters()
			if err != nil {
				return "", err
			}
		}
	}
	backup.CatalogDigest = backupTargetDigest(backup.Registry)
	if target.fullPower || target.power != nil {
		backup.Power.PreviousGUID, err = activePowerGUID()
		if err != nil {
			return "", err
		}
		backup.Power.CreatedGUID, err = newPowerGUID()
		if err != nil {
			return "", err
		}
		backup.PowerApplied = true // Journal before powercfg can create the scheme.
	}
	if target.network != nil {
		backup.Network = []NetProperty{*target.network}
	}
	if err := saveBackup(&backup); err != nil {
		return "", err
	}

	fail := func(cause error) (string, error) {
		backup.Error, backup.Status = cause.Error(), "failed"
		_ = saveBackup(&backup)
		if rollbackErr := rollbackBackup(&backup); rollbackErr != nil {
			return backup.Path, fmt.Errorf("%w; автоматический откат: %v", cause, rollbackErr)
		}
		return backup.Path, fmt.Errorf("%w; изменение автоматически отменено", cause)
	}

	if target.fullPower {
		backup.Power.Settings, err = createPerformancePlan(backup.Power.CreatedGUID)
	} else if target.power != nil {
		backup.Power.Settings, err = createPerformancePlanWithSettings(backup.Power.CreatedGUID, []PowerSetting{*target.power})
	}
	if err != nil {
		return fail(err)
	}
	if target.fullPower || target.power != nil {
		if err := saveBackup(&backup); err != nil {
			return fail(err)
		}
	}
	if target.change != nil {
		backup.AppliedRegistry = 1
		if err := saveBackup(&backup); err != nil {
			return fail(err)
		}
		if err := applyRegistry(*target.change); err != nil {
			return fail(fmt.Errorf("%s: %w", id, err))
		}
		if isMouseTweak(id) {
			backup.Mouse.Applied = true
			if err := saveBackup(&backup); err != nil {
				return fail(err)
			}
			desired := backup.Mouse
			setMouseTweakValue(&desired, id, 0)
			if err := applyMouseParameters(desired); err != nil {
				return fail(err)
			}
		}
	}
	if target.network != nil {
		backup.NetworkApplied = true
		if err := saveBackup(&backup); err != nil {
			return fail(err)
		}
		if err := setNetworkProperties(backup.Network, false); err != nil {
			return fail(err)
		}
	}
	profile := Profile{ID: target.profile}
	if target.change != nil {
		profile.Changes = []RegChange{*target.change}
	}
	if err := verifyApplied(profile, backup); err != nil {
		return fail(err)
	}
	backup.Status, backup.Error = "applied", ""
	if err := saveBackup(&backup); err != nil {
		return fail(err)
	}
	return backup.Path, nil
}

func restoreTweakBackup(requestSID, tweakID, backupID string) (string, error) {
	if !isAdministrator() {
		return "", errors.New("для отката твика нужны права администратора")
	}
	if !sidPattern.MatchString(requestSID) || !tweakIDPattern.MatchString(tweakID) || !backupIDPattern.MatchString(backupID) {
		return "", errors.New("не удалось подтвердить запрос отката твика")
	}
	releaseLock, err := acquireOperationLock()
	if err != nil {
		return "", err
	}
	defer releaseLock()
	if err := checkBoostSession(false); err != nil {
		return "", err
	}
	backup, err := loadBackupByID(requestSID, backupID)
	if err != nil {
		return "", err
	}
	if backup.TweakID != tweakID || !backupRestorable(backup.Status) {
		return backup.Path, errors.New("резервная копия не относится к выбранному твика или уже восстановлена")
	}
	if err := validateBackup(backup); err != nil {
		return backup.Path, err
	}
	if err := validatePowerRestoreOrder(backup); err != nil {
		return backup.Path, err
	}
	if err := setRegistryUserSID(backup.TargetUserSID); err != nil {
		return backup.Path, err
	}
	defer func() { _ = setRegistryUserSID("") }()
	return backup.Path, rollbackBackup(&backup)
}

func tweakCommand(args []string) error {
	if len(args) == 0 || (args[0] != "apply" && args[0] != "restore") {
		return errors.New("tweak доступен только внутреннему elevated-процессу")
	}
	action := args[0]
	set := flag.NewFlagSet("tweak "+action, flag.ContinueOnError)
	id := set.String("id", "", "internal: tweak ID")
	backupID := set.String("backup-id", "", "internal: backup ID")
	parentPID := set.Uint("parent-pid", 0, "internal: PID исходного процесса")
	if err := set.Parse(args[1:]); err != nil {
		return err
	}
	if set.NArg() != 0 || *parentPID == 0 || !isAdministrator() || !tweakIDPattern.MatchString(*id) || !backupIDPattern.MatchString(*backupID) {
		return errors.New("tweak доступен только внутреннему elevated-процессу")
	}
	sid, err := userSIDFromOptimizerProcess(uint32(*parentPID))
	if err != nil {
		return err
	}
	if action == "restore" {
		_, err = restoreTweakBackup(sid, *id, *backupID)
		return err
	}
	if err := setRegistryUserSID(sid); err != nil {
		return err
	}
	defer func() { _ = setRegistryUserSID("") }()
	_, err = applyTweak(*id, *backupID)
	return err
}

func backupRestorable(status string) bool {
	switch status {
	case "applied", "prepared", "checkpoint", "failed", "rolling_back", "rollback_failed":
		return true
	default:
		return false
	}
}

func isMouseTweak(id string) bool {
	return id == "mouse-speed" || id == "mouse-threshold-1" || id == "mouse-threshold-2"
}

func mouseTweakValue(snapshot MouseSnapshot, id string) int32 {
	switch id {
	case "mouse-speed":
		return snapshot.Speed
	case "mouse-threshold-1":
		return snapshot.Threshold1
	case "mouse-threshold-2":
		return snapshot.Threshold2
	default:
		return -1
	}
}

func setMouseTweakValue(snapshot *MouseSnapshot, id string, value int32) {
	switch id {
	case "mouse-speed":
		snapshot.Speed = value
	case "mouse-threshold-1":
		snapshot.Threshold1 = value
	case "mouse-threshold-2":
		snapshot.Threshold2 = value
	}
}

func validatePowerRestoreOrder(backup Backup) error {
	if !backup.PowerApplied || backup.Power.CreatedGUID == "" || backup.Status != "applied" {
		return nil
	}
	active, err := activePowerGUID()
	if err != nil {
		return err
	}
	if !strings.EqualFold(active, backup.Power.CreatedGUID) {
		return errors.New("после этой операции была активирована другая схема питания; сначала откатите более новый power-твик")
	}
	return nil
}

func validateTweakBackupShape(backup Backup, allowed map[string]RegChange) error {
	if !tweakIDPattern.MatchString(backup.TweakID) {
		return errors.New("backup содержит неверный ID твика")
	}
	if expected, ok := allowed[backup.TweakID]; ok {
		if len(backup.Registry) != 1 || backup.Registry[0].Change.ID != expected.ID || len(backup.Network) != 0 || hasPowerState(backup) || backup.NetworkApplied {
			return errors.New("backup твика содержит лишние цели")
		}
		if backup.Mouse.Captured != isMouseTweak(backup.TweakID) || (!isMouseTweak(backup.TweakID) && hasMouseState(backup.Mouse)) {
			return errors.New("backup твика содержит несогласованное состояние мыши")
		}
		return nil
	}
	if backup.TweakID == "power-plan" {
		if len(backup.Registry) != 0 || len(backup.Network) != 0 || backup.Power.CreatedGUID == "" || hasMouseState(backup.Mouse) || backup.NetworkApplied {
			return errors.New("backup power-твика имеет неверную форму")
		}
		return nil
	}
	if strings.HasPrefix(backup.TweakID, "power-") && len(backup.Power.Settings) == 1 {
		setting := backup.Power.Settings[0]
		if len(backup.Registry) != 0 || len(backup.Network) != 0 || backup.Power.CreatedGUID == "" || powerTweakID(setting) != backup.TweakID || !allowedPowerSetting(setting) || hasMouseState(backup.Mouse) || backup.NetworkApplied {
			return errors.New("backup power-твика содержит лишние настройки")
		}
		return nil
	}
	if len(backup.Registry) == 0 && len(backup.Network) == 1 && !hasPowerState(backup) && !hasMouseState(backup.Mouse) && networkTweakID(backup.Network[0]) == backup.TweakID {
		return nil
	}
	return errors.New("backup содержит неизвестный твик")
}

func hasPowerState(backup Backup) bool {
	return backup.Power.PreviousGUID != "" || backup.Power.CreatedGUID != "" || len(backup.Power.Settings) != 0 || backup.PowerApplied
}

func hasMouseState(snapshot MouseSnapshot) bool {
	return snapshot.Captured || snapshot.Applied || snapshot.Threshold1 != 0 || snapshot.Threshold2 != 0 || snapshot.Speed != 0
}

func validatePowerSettings(settings []PowerSetting) error {
	seen := make(map[string]bool, len(settings))
	for _, setting := range settings {
		key := powerSettingKey(setting)
		if seen[key] || !allowedPowerSetting(setting) {
			return errors.New("backup содержит недопустимую настройку питания")
		}
		seen[key] = true
	}
	return nil
}

func allowedPowerSetting(setting PowerSetting) bool {
	if strings.EqualFold(setting.Subgroup, processorPowerSubgroup) || strings.EqualFold(setting.Subgroup, storagePowerSubgroup) {
		_, err := parsePowerGUID(setting.Setting)
		return len(strings.Trim(setting.Setting, "{}")) == 36 && err == nil
	}
	for _, expected := range append(append([]PowerSetting(nil), maximumPowerSettings...), optionalMaximumPowerSettings...) {
		if powerSettingKey(setting) == powerSettingKey(expected) {
			return setting.Value == expected.Value
		}
	}
	return false
}

func backupIDNow() string { return time.Now().UTC().Format("20060102T150405.000000000Z") }

func parentPIDArg() string { return strconv.Itoa(os.Getpid()) }
