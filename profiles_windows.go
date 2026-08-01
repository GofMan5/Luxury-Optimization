package main

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func profileByID(id string) (Profile, error) {
	switch id {
	case profileRecommended:
		return recommendedProfile(), nil
	case profileMaximum:
		p := recommendedProfile()
		p.ID = profileMaximum
		p.Name = "Максимальная производительность"
		p.Description = "Безопасная база плюс отключение системного power throttling, отдельная схема питания и поддерживаемые low-latency параметры Ethernet."
		p.PerformancePlan = true
		p.NetworkLatency = true
		p.RebootRequired = true
		p.Changes = append(p.Changes,
			dword("power-throttling", "Питание", "Отключить системный power throttling для всех процессов", "HKLM", `SYSTEM\CurrentControlSet\Control\Power\PowerThrottling`, "PowerThrottlingOff", 1),
			dword("startup-delay", "Интерфейс", "Убрать задержку запуска автозагрузки после входа", "HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\Serialize`, "StartupDelayInMSec", 0),
			stringValue("menu-show-delay", "Интерфейс", "Убрать задержку открытия меню", "HKCU", `Control Panel\Desktop`, "MenuShowDelay", "0"),
		)
		return p, nil
	case profileRepair:
		return repairProfile(), nil
	default:
		return Profile{}, fmt.Errorf("неизвестный профиль %q", id)
	}
}

func recommendedProfile() Profile {
	return Profile{
		ID:          profileRecommended,
		Name:        "Рекомендуемая оптимизация",
		Description: "Обратимые игровые и интерфейсные настройки без отключения защит Windows.",
		Changes: []RegChange{
			dword("game-mode-allow", "Игры", "Разрешить автоматический Game Mode", "HKCU", `SOFTWARE\Microsoft\GameBar`, "AllowAutoGameMode", 1),
			dword("game-mode-enable", "Игры", "Включить Game Mode", "HKCU", `SOFTWARE\Microsoft\GameBar`, "AutoGameModeEnabled", 1),
			dword("capture-game-dvr", "Игры", "Отключить фоновую запись Game DVR", "HKCU", `System\GameConfigStore`, "GameDVR_Enabled", 0),
			dword("capture-app", "Игры", "Отключить фоновый захват экрана", "HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\GameDVR`, "AppCaptureEnabled", 0),
			stringValue("mouse-speed", "Ввод", "Отключить ускорение указателя", "HKCU", `Control Panel\Mouse`, "MouseSpeed", "0"),
			stringValue("mouse-threshold-1", "Ввод", "Убрать первый порог ускорения мыши", "HKCU", `Control Panel\Mouse`, "MouseThreshold1", "0"),
			stringValue("mouse-threshold-2", "Ввод", "Убрать второй порог ускорения мыши", "HKCU", `Control Panel\Mouse`, "MouseThreshold2", "0"),
			dword("ui-transparency", "Интерфейс", "Отключить прозрачность Windows", "HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Themes\Personalize`, "EnableTransparency", 0),
			dword("ui-taskbar-animation", "Интерфейс", "Отключить анимацию панели задач", "HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\Advanced`, "TaskbarAnimations", 0),
			stringValue("ui-window-animation", "Интерфейс", "Отключить анимацию сворачивания окон", "HKCU", `Control Panel\Desktop\WindowMetrics`, "MinAnimate", "0"),
		},
	}
}

func repairProfile() Profile {
	changes := []RegChange{
		deleteValue("repair-csrss-cpu", "Безопасность", "Удалить принудительный realtime-приоритет CSRSS", "HKLM", `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Image File Execution Options\csrss.exe\PerfOptions`, "CpuPriorityClass"),
		deleteValue("repair-csrss-io", "Безопасность", "Удалить принудительный I/O-приоритет CSRSS", "HKLM", `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Image File Execution Options\csrss.exe\PerfOptions`, "IoPriority"),
		deleteValue("repair-spectre-value", "Безопасность", "Вернуть управление CPU mitigations системе", "HKLM", `SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management`, "FeatureSettings"),
		deleteValue("repair-spectre-override", "Безопасность", "Удалить отключение Spectre/Meltdown", "HKLM", `SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management`, "FeatureSettingsOverride"),
		deleteValue("repair-spectre-mask", "Безопасность", "Удалить маску отключения Spectre/Meltdown", "HKLM", `SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management`, "FeatureSettingsOverrideMask"),
		deleteValue("repair-sehop", "Безопасность", "Вернуть системное управление SEHOP", "HKLM", `SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management`, "KernelSEHOPEnabled"),
		deleteValue("repair-exception-chain", "Безопасность", "Вернуть проверку exception chain", "HKLM", `SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management`, "DisableExceptionChainValidation"),
		deleteValue("repair-cfg", "Безопасность", "Вернуть системное управление CFG", "HKLM", `SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management`, "EnableCfg"),
		deleteValue("repair-security-toast", "Безопасность", "Вернуть уведомления безопасности и обслуживания", "HKCU", `Software\Microsoft\Windows\CurrentVersion\Notifications\Settings\Windows.SystemToast.SecurityAndMaintenance`, "Enabled"),
		deleteValue("repair-security-notifications", "Безопасность", "Удалить отключение уведомлений Windows Security", "HKLM", `SOFTWARE\Microsoft\Windows Defender Security Center\Notifications`, "DisableNotifications"),
		deleteValue("repair-security-policy-notifications", "Безопасность", "Удалить политику отключения уведомлений Windows Security", "HKLM", `SOFTWARE\Policies\Microsoft\Windows Defender Security Center\Notifications`, "DisableNotifications"),
		deleteValue("repair-security-policy-enhanced", "Безопасность", "Удалить политику отключения расширенных уведомлений Windows Security", "HKLM", `SOFTWARE\Policies\Microsoft\Windows Defender Security Center\Notifications`, "DisableEnhancedNotifications"),
		deleteValue("repair-defender-reporting-notifications", "Безопасность", "Вернуть расширенные уведомления Defender", "HKLM", `SOFTWARE\Policies\Microsoft\Windows Defender\Reporting`, "DisableEnhancedNotifications"),
		deleteValue("repair-global-toasts", "Безопасность", "Удалить глобальное отключение системных уведомлений", "HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Notifications\Settings`, "NOC_GLOBAL_SETTING_TOASTS_ENABLED"),
		deleteValue("repair-defender", "Безопасность", "Удалить устаревшую политику отключения Defender", "HKLM", `SOFTWARE\Policies\Microsoft\Windows Defender`, "DisableAntiSpyware"),
		deleteValue("repair-nvidia-preemption", "GPU", "Удалить недокументированное отключение GPU preemption", "HKLM", `SYSTEM\CurrentControlSet\Services\nvlddmkm`, "DisablePreemption"),
		deleteValue("repair-nvidia-cuda-preemption", "GPU", "Удалить недокументированное отключение CUDA preemption", "HKLM", `SYSTEM\CurrentControlSet\Services\nvlddmkm`, "DisableCudaContextPreemption"),
		deleteValue("repair-nvidia-write-combining", "GPU", "Удалить недокументированное отключение write combining", "HKLM", `SYSTEM\CurrentControlSet\Services\nvlddmkm`, "DisableWriteCombining"),
		dword("repair-firewall-domain", "Безопасность", "Включить Firewall для доменного профиля", "HKLM", `SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\DomainProfile`, "EnableFirewall", 1),
		dword("repair-firewall-private", "Безопасность", "Включить Firewall для частного профиля", "HKLM", `SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\StandardProfile`, "EnableFirewall", 1),
		dword("repair-firewall-public", "Безопасность", "Включить Firewall для публичного профиля", "HKLM", `SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\PublicProfile`, "EnableFirewall", 1),
	}
	return Profile{
		ID:             profileRepair,
		Name:           "Ремонт опасных твиков старого BAT",
		Description:    "Удаляет известные опасные значения старого скрипта, возвращает уведомления Windows Security и включает Firewall. Перед изменениями создаётся точная резервная копия.",
		Changes:        changes,
		RepairFirewall: true,
		RebootRequired: true,
	}
}

func dword(id, category, description, hive, path, name string, value uint32) RegChange {
	return RegChange{ID: id, Category: category, Description: description, Hive: hive, Path: path, Name: name, Kind: registry.DWORD, DWord: value}
}

func stringValue(id, category, description, hive, path, name, value string) RegChange {
	return RegChange{ID: id, Category: category, Description: description, Hive: hive, Path: path, Name: name, Kind: registry.SZ, String: value}
}

func deleteValue(id, category, description, hive, path, name string) RegChange {
	return RegChange{ID: id, Category: category, Description: description, Hive: hive, Path: path, Name: name, Delete: true}
}

func validateProfile(p Profile) error {
	seen := make(map[string]string, len(p.Changes))
	for _, change := range p.Changes {
		key := strings.ToUpper(change.Hive + `\` + change.Path + `\` + change.Name)
		if previous, ok := seen[key]; ok {
			return fmt.Errorf("конфликт целей %s и %s: %s", previous, change.ID, key)
		}
		seen[key] = change.ID
	}
	return nil
}

func gpuVendor(pnpID, name string) string {
	value := strings.ToUpper(pnpID + " " + name)
	switch {
	case strings.Contains(value, "VEN_10DE") || strings.Contains(value, "NVIDIA"):
		return "NVIDIA"
	case strings.Contains(value, "VEN_1002") || strings.Contains(value, "AMD") || strings.Contains(value, "RADEON"):
		return "AMD"
	case strings.Contains(value, "VEN_8086") || strings.Contains(value, "INTEL"):
		return "Intel"
	case strings.Contains(value, "VEN_1414") || strings.Contains(value, "MICROSOFT"):
		return "Microsoft"
	case strings.Contains(value, "QUALCOMM") || strings.Contains(value, "SNAPDRAGON"):
		return "Qualcomm"
	default:
		return "Другой"
	}
}
