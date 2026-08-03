package main

import (
	"errors"
	"os"
	"os/exec"
	"time"
)

func profileByID(id string) (Profile, error) {
	switch id {
	case profileRecommended:
		return Profile{ID: id, Name: "Recommended Linux session", Description: "GameMode when available; no persistent system mutation"}, nil
	case profileMaximum:
		return Profile{ID: id, Name: "Maximum Linux session", Description: "GameMode plus explicit process priority/affinity; no persistent system mutation"}, nil
	default:
		return Profile{}, errors.New("неизвестный профиль: используйте recommended или maximum")
	}
}

func collectAudit() Audit {
	hardware, err := detectHardware()
	audit := Audit{GeneratedAt: time.Now(), Version: version, Hardware: hardware, Administrator: os.Geteuid() == 0, Capabilities: linuxCapabilities(), Findings: []Finding{}}
	if err != nil {
		audit.Warnings = append(audit.Warnings, "Неполные сведения о системе: "+err.Error())
	}
	if _, err := exec.LookPath("gamemoderun"); err != nil {
		audit.Findings = append(audit.Findings, Finding{ID: "linux-gamemode", Title: "Feral GameMode не найден", Evidence: "boost запустит игру напрямую и не сломается", Action: "Установите пакет gamemode, если он доступен в дистрибутиве"})
	}
	if governor := currentGovernor(); governor != "" && governor != "performance" && governor != "schedutil" {
		audit.Findings = append(audit.Findings, Finding{ID: "linux-governor", Title: "CPU governor: " + governor, Evidence: "не меняется глобально", Action: "Используйте GameMode или штатный профиль питания дистрибутива"})
	}
	return audit
}

func buildPlan(profileID string) (Plan, error) {
	profile, err := profileByID(profileID)
	if err != nil {
		return Plan{}, err
	}
	hardware, hardwareErr := detectHardware()
	plan := Plan{Profile: profile, Hardware: hardware}
	if hardwareErr != nil {
		plan.Warnings = append(plan.Warnings, "Не удалось полностью прочитать систему: "+hardwareErr.Error())
	}
	_, gameModeErr := exec.LookPath("gamemoderun")
	plan.Items = append(plan.Items,
		PlanItem{Category: "Session", Name: "Feral GameMode", Current: availableDetail(gameModeErr == nil, "available", "not installed"), Desired: availableDetail(gameModeErr == nil, "enabled for game process", "skipped"), Changed: gameModeErr == nil},
		PlanItem{Category: "Process", Name: "Priority and affinity", Current: "unchanged", Desired: "only explicit per-game values", Changed: false},
		PlanItem{Category: "System", Name: "Windows registry/power/NIC tweaks", Current: "not applicable", Desired: "skipped", Changed: false},
	)
	if gameModeErr != nil {
		plan.Warnings = append(plan.Warnings, "gamemoderun недоступен: boost продолжит работу без него.")
	}
	if profileID == profileMaximum && hardware.HasBattery {
		plan.Warnings = append(plan.Warnings, "Обнаружена батарея: используйте максимальный профиль только при питании от сети.")
	}
	return plan, nil
}
