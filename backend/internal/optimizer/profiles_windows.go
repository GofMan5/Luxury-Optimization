package optimizer

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
		p.Description = "Игровая база плюс поддерживаемые CPU, AC power-plan и Ethernet low-latency настройки."
		p.PerformancePlan = true
		p.NetworkLatency = true
		p.Changes = append(p.Changes,
			dword("startup-delay", "Интерфейс", "Убрать задержку запуска автозагрузки после входа", "HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\Serialize`, "StartupDelayInMSec", 0),
			stringValue("menu-show-delay", "Интерфейс", "Убрать задержку открытия меню", "HKCU", `Control Panel\Desktop`, "MenuShowDelay", "0"),
		)
		return p, nil
	default:
		return Profile{}, fmt.Errorf("неизвестный профиль %q", id)
	}
}

func recommendedProfile() Profile {
	return Profile{
		ID:          profileRecommended,
		Name:        "Рекомендуемая игровая оптимизация",
		Description: "Обратимые игровые и интерфейсные настройки без отключения системной защиты.",
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

func dword(id, category, description, hive, path, name string, value uint32) RegChange {
	return RegChange{ID: id, Category: category, Description: description, Hive: hive, Path: path, Name: name, Kind: registry.DWORD, DWord: value}
}

func stringValue(id, category, description, hive, path, name, value string) RegChange {
	return RegChange{ID: id, Category: category, Description: description, Hive: hive, Path: path, Name: name, Kind: registry.SZ, String: value}
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
