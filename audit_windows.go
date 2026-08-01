package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func collectAudit() Audit {
	audit := Audit{GeneratedAt: time.Now(), Version: version, Administrator: isAdministrator()}
	hardware, err := detectHardware()
	audit.Hardware = hardware
	if err != nil {
		audit.Warnings = append(audit.Warnings, "Неполные сведения о железе: "+err.Error())
	}
	audit.ActivePowerGUID, err = activePowerGUID()
	if err != nil {
		audit.Warnings = append(audit.Warnings, "Схема питания не прочитана: "+err.Error())
	}
	audit.Findings = detectLegacyFindings(&audit)
	return audit
}

func detectLegacyFindings(audit *Audit) []Finding {
	var findings []Finding
	addValueFinding := func(severity, id, title, hive, path, name, repair string) {
		if snapshot, ok := valueExists(hive, path, name); ok {
			findings = append(findings, Finding{Severity: severity, ID: id, Title: title, Evidence: hive + `\` + path + `\` + name + " = " + formatSnapshot(snapshot), Repair: repair})
		}
	}
	addDWordFinding := func(severity, id, title, hive, path, name string, dangerous uint32) {
		if value, ok := readDWord(hive, path, name); ok && value == dangerous {
			findings = append(findings, Finding{Severity: severity, ID: id, Title: title, Evidence: fmt.Sprintf("%s\\%s\\%s = %d", hive, path, name, value), Repair: "Кнопка «Ремонт старого BAT» удалит значение."})
		}
	}
	const memory = `SYSTEM\CurrentControlSet\Control\Session Manager\Memory Management`
	const csrss = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Image File Execution Options\csrss.exe\PerfOptions`
	addValueFinding("critical", "csrss-cpu-priority", "Задан опасный приоритет CPU для CSRSS", "HKLM", csrss, "CpuPriorityClass", "Кнопка «Ремонт старого BAT» удалит значение.")
	addValueFinding("critical", "csrss-io-priority", "Задан опасный I/O-приоритет для CSRSS", "HKLM", csrss, "IoPriority", "Кнопка «Ремонт старого BAT» удалит значение.")
	addValueFinding("critical", "cpu-mitigation-override", "Переопределены аппаратные защиты CPU", "HKLM", memory, "FeatureSettingsOverride", "Вернуть управление mitigations Windows и перезагрузить ПК.")
	addValueFinding("critical", "cpu-mitigation-mask", "Задана маска отключения защит CPU", "HKLM", memory, "FeatureSettingsOverrideMask", "Вернуть управление mitigations Windows и перезагрузить ПК.")
	addDWordFinding("high", "security-toast-disabled", "Отключены уведомления безопасности и обслуживания", "HKCU", `Software\Microsoft\Windows\CurrentVersion\Notifications\Settings\Windows.SystemToast.SecurityAndMaintenance`, "Enabled", 0)
	addDWordFinding("high", "security-notifications-disabled", "Отключены уведомления Windows Security", "HKLM", `SOFTWARE\Microsoft\Windows Defender Security Center\Notifications`, "DisableNotifications", 1)
	addDWordFinding("high", "security-policy-notifications-disabled", "Политика отключает уведомления Windows Security", "HKLM", `SOFTWARE\Policies\Microsoft\Windows Defender Security Center\Notifications`, "DisableNotifications", 1)
	addDWordFinding("high", "security-policy-enhanced-disabled", "Политика отключает расширенные уведомления Windows Security", "HKLM", `SOFTWARE\Policies\Microsoft\Windows Defender Security Center\Notifications`, "DisableEnhancedNotifications", 1)
	addDWordFinding("high", "defender-reporting-notifications-disabled", "Отключены расширенные уведомления Defender", "HKLM", `SOFTWARE\Policies\Microsoft\Windows Defender\Reporting`, "DisableEnhancedNotifications", 1)
	addDWordFinding("high", "global-toasts-disabled", "Глобально отключены системные уведомления", "HKCU", `SOFTWARE\Microsoft\Windows\CurrentVersion\Notifications\Settings`, "NOC_GLOBAL_SETTING_TOASTS_ENABLED", 0)
	if value, ok := readDWord("HKLM", `SOFTWARE\Policies\Microsoft\Windows Defender`, "DisableAntiSpyware"); ok && value != 0 {
		findings = append(findings, Finding{Severity: "critical", ID: "defender-policy", Title: "Старый BAT пытался отключить Microsoft Defender", Evidence: fmt.Sprintf("DisableAntiSpyware = %d", value), Repair: "Удалить устаревшую политику через «Ремонт старого BAT»."})
	}
	profiles := []struct{ name, path string }{
		{"доменный", `SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\DomainProfile`},
		{"частный", `SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\StandardProfile`},
		{"публичный", `SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\PublicProfile`},
	}
	for _, profile := range profiles {
		if value, ok := readDWord("HKLM", profile.path, "EnableFirewall"); ok && value == 0 {
			findings = append(findings, Finding{Severity: "critical", ID: "firewall-" + profile.name, Title: "Firewall отключён: " + profile.name + " профиль", Evidence: "EnableFirewall = 0", Repair: "Включить через «Ремонт старого BAT»."})
		}
	}
	if value, ok := readDWord("HKLM", `SYSTEM\CurrentControlSet\Services\MpsSvc`, "Start"); ok && value == 4 {
		findings = append(findings, Finding{Severity: "critical", ID: "firewall-service", Title: "Служба Windows Firewall отключена", Evidence: "MpsSvc Start = 4", Repair: "Вернуть автоматический запуск через «Ремонт старого BAT»."})
	}
	for _, item := range []struct{ id, title, name string }{
		{"nvidia-preemption", "Присутствует недокументированный NVIDIA DisablePreemption", "DisablePreemption"},
		{"nvidia-cuda-preemption", "Присутствует недокументированный NVIDIA DisableCudaContextPreemption", "DisableCudaContextPreemption"},
		{"nvidia-write-combining", "Присутствует недокументированный NVIDIA DisableWriteCombining", "DisableWriteCombining"},
	} {
		addValueFinding("high", item.id, item.title, "HKLM", `SYSTEM\CurrentControlSet\Services\nvlddmkm`, item.name, "Удалить через «Ремонт старого BAT» и перезагрузить ПК.")
	}
	systemDirectory, rootErr := trustedSystemDirectory()
	if rootErr != nil {
		audit.Warnings = append(audit.Warnings, "Каталог Windows не определён: "+rootErr.Error())
		systemDirectory = `C:\__gofman3_invalid_windows_directory__`
	}
	runtimeBroker := filepath.Join(systemDirectory, "RuntimeBroker.exe")
	if _, err := os.Stat(runtimeBroker); err != nil {
		findings = append(findings, Finding{Severity: "critical", ID: "runtimebroker-missing", Title: "RuntimeBroker.exe отсутствует или недоступен", Evidence: runtimeBroker + ": " + err.Error(), Repair: "Не восстанавливается автоматически: запустить DISM /RestoreHealth и SFC /scannow от администратора."})
	}
	if _, err := os.Stat(runtimeBroker + ".disabled"); err == nil {
		findings = append(findings, Finding{Severity: "critical", ID: "runtimebroker-disabled", Title: "Найден RuntimeBroker.exe.disabled", Evidence: runtimeBroker + ".disabled", Repair: "Проверить системные файлы через DISM и SFC; не копировать .bak вручную."})
	}
	opengl := filepath.Join(systemDirectory, "opengl32.dll")
	if info, err := os.Stat(opengl); err == nil && info.Size() > 100*1024*1024 {
		findings = append(findings, Finding{Severity: "critical", ID: "opengl-size", Title: "opengl32.dll имеет аномальный размер", Evidence: fmt.Sprintf("%s: %d байт", opengl, info.Size()), Repair: "Запустить DISM /RestoreHealth и SFC /scanfile для opengl32.dll."})
	}
	if audit.Administrator {
		if output, err := runCommand(10*time.Second, systemTool("bcdedit.exe"), "/enum", "{current}"); err == nil {
			text := strings.ToLower(string(output))
			for _, option := range []string{"disabledynamictick", "useplatformtick", "useplatformclock", "isolatedcontext", "allowedinmemorysettings"} {
				if strings.Contains(text, option) {
					findings = append(findings, Finding{Severity: "high", ID: "bcd-" + option, Title: "Обнаружен нестандартный BCD timer/security параметр", Evidence: option + " присутствует в {current}", Repair: "Проверить значение вручную; автоматический BCD-ремонт намеренно не выполняется."})
				}
			}
		}
	} else {
		audit.Warnings = append(audit.Warnings, "BCD не проверен без прав администратора.")
	}
	return findings
}
