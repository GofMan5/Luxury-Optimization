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
	return audit
}

func detectOptimizationFindings(audit *Audit) []Finding {
	var findings []Finding
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
