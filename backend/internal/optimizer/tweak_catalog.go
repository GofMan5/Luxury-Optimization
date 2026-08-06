package optimizer

import "strings"

type tweakMetadata struct {
	effect, benefit, risk, level string
}

var mediumPowerSettingIDs = map[string]bool{
	"bc5038f7-23e0-4960-96da-33abaf5935ec": true,
	"bc5038f7-23e0-4960-96da-33abaf5935ed": true,
	"bc5038f7-23e0-4960-96da-33abaf5935ee": true,
	"893dee8e-2bef-41e0-89c6-b55d0929964c": true,
	"893dee8e-2bef-41e0-89c6-b55d0929964d": true,
	"893dee8e-2bef-41e0-89c6-b55d0929964e": true,
	"36687f9e-e3a5-4dbf-b1dc-15eb381c6863": true,
	"36687f9e-e3a5-4dbf-b1dc-15eb381c6864": true,
	"36687f9e-e3a5-4dbf-b1dc-15eb381c6865": true,
	"be337238-0d82-4146-a960-4f3749d470c7": true,
	"94d3a615-a899-4ac5-ae2b-e4d8f634367f": true,
}

func isMediumPowerSetting(id string) bool {
	return mediumPowerSettingIDs[strings.TrimPrefix(strings.ToLower(id), "power-")]
}

var tweakCatalog = map[string]tweakMetadata{
	"game-mode-allow":      {"Allows Windows to engage Game Mode automatically for detected games.", "May reduce background scheduling interference while a game is active.", "Low: Windows remains in control and may show no measurable change on some systems.", "low"},
	"game-mode-enable":     {"Enables the current-user Windows Game Mode preference.", "Can prioritize the active game workload over background activity.", "Low: may change background-app responsiveness; FPS gain is workload-dependent.", "low"},
	"capture-game-dvr":     {"Stops Game DVR from continuously preparing background recording.", "Can reduce capture-related CPU, GPU and storage activity when recording is unused.", "Medium: background recording and instant replay become unavailable.", "medium"},
	"capture-app":          {"Disables current-user background app capture.", "Avoids capture overhead and surprise recording during games.", "Medium: Windows capture features must be re-enabled before use.", "medium"},
	"mouse-speed":          {"Disables enhanced pointer acceleration for the current user.", "Produces more consistent physical mouse-to-pointer movement for aiming.", "Low: users accustomed to acceleration may dislike the new feel.", "low"},
	"mouse-threshold-1":    {"Removes the first legacy mouse acceleration threshold.", "Keeps input scaling consistent with acceleration disabled.", "Low: changes pointer feel only; no FPS effect.", "low"},
	"mouse-threshold-2":    {"Removes the second legacy mouse acceleration threshold.", "Keeps input scaling consistent with acceleration disabled.", "Low: changes pointer feel only; no FPS effect.", "low"},
	"ui-transparency":      {"Disables Windows transparency effects for the current user.", "Slightly reduces desktop composition work outside the game.", "Low: visual transparency is removed; gaming gains are usually small.", "low"},
	"ui-taskbar-animation": {"Disables taskbar animations for the current user.", "Makes shell interactions immediate and removes minor animation work.", "Low: purely visual trade-off with no guaranteed in-game gain.", "low"},
	"ui-window-animation":  {"Disables minimize and restore window animations.", "Reduces shell animation delay when switching in and out of games.", "Low: window transitions become abrupt.", "low"},
	"startup-delay":        {"Removes the post-sign-in delay before startup applications launch.", "Starts user applications sooner after sign-in.", "Medium: creates a stronger CPU and storage burst during login.", "medium"},
	"menu-show-delay":      {"Removes the configured menu opening delay.", "Makes classic Windows menus respond immediately.", "Low: menus may feel too abrupt; no direct FPS benefit.", "low"},
	"power-plan":           {"Creates a separate reversible AC performance plan without editing the original plan.", "Can reduce frequency and device power-state transition latency during sustained play.", "High: increases power use, heat and fan noise; use on AC power with adequate cooling.", "high"},
	"linux-gamemode":       {"Runs the game through Feral GameMode when it is installed.", "Lets the distribution apply temporary game-scoped performance policy.", "Low: unavailable systems safely launch without it.", "low"},
	"linux-process":        {"Keeps priority and CPU affinity opt-in per saved launch.", "Allows targeted process tuning without global scheduler changes.", "Medium: bad affinity masks can reduce performance.", "medium"},
	"linux-windows-skip":   {"Explicitly skips Windows-only registry, power and adapter settings.", "Prevents fake compatibility and partial mutation on Linux.", "Low: no system change is performed.", "low"},
}

func describePlan(plan *Plan) {
	for index := range plan.Items {
		item := &plan.Items[index]
		metadata, ok := tweakCatalog[item.ID]
		if item.ID == "power-plan" && canonicalProfileID(plan.Profile.ID) == profileMedium {
			metadata, ok = tweakMetadata{"Creates a separate reversible Medium AC plan without editing the current plan.", "Applies only the reviewed CPU limits, EPP, boost and active-cooling policies.", "Medium: may increase heat and power use under load; the workload may show no gain.", "medium"}, true
		}
		if !ok && strings.HasPrefix(item.ID, "power-") {
			level := "high"
			risk := "High: can increase heat and power draw, and the workload may show no gain; unsupported settings are skipped."
			if isMediumPowerSetting(item.ID) {
				level, risk = "medium", "Medium: changes a reviewed CPU performance policy only inside a cloned AC plan; heat and power use may increase."
			}
			metadata = tweakMetadata{"Copies one supported native Windows High Performance AC policy into a separate Luxury plan.", "May reduce a CPU, storage or device power-state transition during play.", risk, level}
			ok = true
		}
		if !ok && strings.HasPrefix(item.ID, "ethernet-") {
			metadata = tweakMetadata{"Disables one low-power or interrupt-moderation property exposed by the physical Ethernet adapter.", "May reduce packet batching or wake-up latency on a supported wired adapter.", "Medium: can increase CPU use or power draw and may reduce throughput on some adapters.", "medium"}
		}
		if !ok {
			metadata = tweakMetadata{item.Name, "No generic performance claim; measure this target on the actual workload.", "Unknown until measured; the original value remains restorable.", "medium"}
		}
		item.Effect, item.Benefit, item.Risk, item.RiskLevel, item.Reversible = metadata.effect, metadata.benefit, metadata.risk, metadata.level, true
	}
}
