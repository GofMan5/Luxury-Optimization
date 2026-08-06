package optimizer

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func profileByID(id string) (Profile, error) {
	switch id {
	case profileLite:
		return windowsProfile(id, "Lite", "Безопасные игровые и интерфейсные улучшения без изменения схемы питания.", []string{
			"game-mode-allow", "game-mode-enable", "ui-transparency", "ui-taskbar-animation", "ui-window-animation", "menu-show-delay",
		}, false, false), nil
	case profileMedium:
		return windowsProfile(id, "Medium", "Lite плюс отключение фонового захвата, предсказуемый ввод и ограниченный набор нативных CPU AC-политик.", []string{
			"game-mode-allow", "game-mode-enable", "capture-game-dvr", "capture-app", "mouse-speed", "mouse-threshold-1", "mouse-threshold-2",
			"ui-transparency", "ui-taskbar-animation", "ui-window-animation", "menu-show-delay",
		}, true, false), nil
	case profileMaximum:
		return windowsProfile(id, "Max", "Все поддерживаемые нативные CPU/storage AC-политики, игровые настройки и физический Ethernet для максимальной производительности.", nil, true, true), nil
	case profileLegacyRecommended:
		return windowsProfile(id, "Legacy recommended", "Совместимость с резервными копиями и игровыми профилями ранних 1.0.x.", []string{
			"game-mode-allow", "game-mode-enable", "capture-game-dvr", "capture-app", "mouse-speed", "mouse-threshold-1", "mouse-threshold-2",
			"ui-transparency", "ui-taskbar-animation", "ui-window-animation",
		}, false, false), nil
	case profileLegacyMaximum:
		return windowsProfile(id, "Legacy maximum", "Совместимость с резервными копиями и игровыми профилями ранних 1.0.x.", nil, true, true), nil
	default:
		return Profile{}, fmt.Errorf("неизвестный профиль %q", id)
	}
}

func windowsProfile(id, name, description string, included []string, performancePlan, networkLatency bool) Profile {
	changes := windowsProfileChanges()
	if included != nil {
		wanted := make(map[string]bool, len(included))
		for _, tweakID := range included {
			wanted[tweakID] = true
		}
		selected := make([]RegChange, 0, len(included))
		for _, change := range changes {
			if wanted[change.ID] {
				selected = append(selected, change)
			}
		}
		changes = selected
	}
	return Profile{ID: id, Name: name, Description: description, Changes: changes, PerformancePlan: performancePlan, NetworkLatency: networkLatency}
}

func windowsProfileChanges() []RegChange {
	return []RegChange{
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
		dword("startup-delay", "Интерфейс", "Убрать задержку запуска автозагрузки после входа", "HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\Serialize`, "StartupDelayInMSec", 0),
		stringValue("menu-show-delay", "Интерфейс", "Убрать задержку открытия меню", "HKCU", `Control Panel\Desktop`, "MenuShowDelay", "0"),
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
