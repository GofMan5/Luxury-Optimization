package optimizer

import (
	"errors"
	"os"
	"os/exec"
	"time"
)

func profileByID(id string) (Profile, error) {
	switch canonicalProfileID(id) {
	case profileLite:
		return Profile{ID: id, Name: "Lite Linux session", Description: "GameMode when available; no persistent system mutation"}, nil
	case profileMedium:
		return Profile{ID: id, Name: "Medium Linux session", Description: "GameMode plus explicit per-game process controls; no persistent system mutation"}, nil
	case profileMaximum:
		return Profile{ID: id, Name: "Max Linux session", Description: "Maximum session controls supported by GameMode and explicit process settings; no persistent system mutation"}, nil
	default:
		return Profile{}, errors.New("неизвестный профиль: используйте lite, medium или max")
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
		PlanItem{ID: "linux-gamemode", Category: "Session", Name: "Feral GameMode", Current: availableDetail(gameModeErr == nil, "available", "not installed"), Desired: availableDetail(gameModeErr == nil, "enabled for game process", "skipped"), Changed: gameModeErr == nil},
		PlanItem{ID: "linux-process", Category: "Process", Name: "Priority and affinity", Current: "unchanged", Desired: "only explicit per-game values", Changed: false},
		PlanItem{ID: "linux-windows-skip", Category: "System", Name: "Windows registry/power/NIC tweaks", Current: "not applicable", Desired: "skipped", Changed: false},
	)
	if gameModeErr != nil {
		plan.Warnings = append(plan.Warnings, "gamemoderun недоступен: boost продолжит работу без него.")
	}
	if canonicalProfileID(profileID) == profileMaximum && hardware.HasBattery {
		plan.Warnings = append(plan.Warnings, "Обнаружена батарея: используйте максимальный профиль только при питании от сети.")
	}
	return plan, nil
}
