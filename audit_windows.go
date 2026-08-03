package main

import (
	"fmt"
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
	audit.Findings = detectOptimizationFindings(&audit)
	audit.Capabilities = windowsCapabilities()
	return audit
}

func windowsCapabilities() []Capability {
	return []Capability{
		{ID: "persistent-profile", Available: true, Mode: "reversible", Detail: "registry, mouse, power and supported Ethernet values use backup/apply/read-back/rollback"},
		{ID: "game-boost", Available: true, Mode: "session", Detail: "game stays non-elevated; system profile is restored after exit"},
		{ID: "game-discovery", Available: true, Mode: "read-only", Detail: "Steam, Epic and fixed-drive Xbox discovery"},
		{ID: "startup", Available: true, Mode: "reversible", Detail: "HKCU startup values are backed up and restored by exact type/value"},
		{ID: "services", Available: true, Mode: "read-only", Detail: "Windows SCM inventory does not change service configuration"},
		{ID: "self-update", Available: true, Mode: "opt-in", Detail: "GitHub Release asset is selected by OS/arch and verified against SHA256SUMS.txt"},
	}
}

func detectOptimizationFindings(audit *Audit) []Finding {
	findings := []Finding{}
	profile := recommendedProfile()
	changed := 0
	for _, change := range profile.Changes {
		matches, _, err := registryMatches(change)
		if err != nil {
			audit.Warnings = append(audit.Warnings, fmt.Sprintf("Настройка %s не прочитана: %v", change.ID, err))
			continue
		}
		if !matches {
			changed++
		}
	}
	if changed > 0 {
		findings = append(findings, Finding{
			ID:       "recommended-profile-drift",
			Title:    "Рекомендуемый игровой профиль применён не полностью",
			Evidence: fmt.Sprintf("Отличаются %d из %d настроек", changed, len(profile.Changes)),
			Action:   "Открыть точный план рекомендуемого профиля.",
		})
	}
	startup := listStartupEntries()
	present := 0
	for _, entry := range startup.Entries {
		if entry.State == "present" {
			present++
		}
	}
	if present >= 10 {
		findings = append(findings, Finding{
			ID:       "startup-load",
			Title:    "Много программ зарегистрировано в автозагрузке",
			Evidence: fmt.Sprintf("Найдено %d registry startup-команд", present),
			Action:   "Проверить startup list и отключить только ненужные HKCU-команды.",
		})
	}
	return findings
}
