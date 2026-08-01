package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestShippedProfilesHaveNoConflictsOrDangerousTweaks(t *testing.T) {
	for _, id := range []string{profileRecommended, profileMaximum} {
		profile, err := profileByID(id)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateProfile(profile); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
	}
	for _, id := range []string{profileRecommended, profileMaximum} {
		profile, _ := profileByID(id)
		for _, change := range profile.Changes {
			target := strings.ToLower(change.Path + `\` + change.Name)
			for _, forbidden := range []string{"csrss.exe", "disableantispyware", "featuresettingsoverride", "runtimebroker", "disablepreemption", "disablewritecombining"} {
				if strings.Contains(target, forbidden) {
					t.Fatalf("%s contains forbidden target %s", id, target)
				}
			}
		}
	}
}

func TestGPUVendorDetection(t *testing.T) {
	tests := map[string]string{
		`PCI\VEN_10DE&DEV_2504`: "NVIDIA",
		`PCI\VEN_1002&DEV_73BF`: "AMD",
		`PCI\VEN_8086&DEV_56A0`: "Intel",
		`PCI\VEN_1414&DEV_008C`: "Microsoft",
		`PCI\VEN_ABCD&DEV_0001`: "Другой",
	}
	for pnp, expected := range tests {
		if actual := gpuVendor(pnp, ""); actual != expected {
			t.Fatalf("%s: got %s, want %s", pnp, actual, expected)
		}
	}
}

func TestNetworkChangesOnlySelectEnabledProperties(t *testing.T) {
	properties := []NetProperty{
		{Keyword: "*EEE", Values: []string{"0"}},
		{Keyword: "*InterruptModeration", Values: []string{"1"}},
		{Keyword: "AdvancedEEE", Values: []string{"2"}},
		{Keyword: "GigaLite", Values: []string{"0", "1"}},
	}
	changes := networkChanges(properties)
	if len(changes) != 3 || changes[0].Keyword != "*InterruptModeration" || changes[1].Keyword != "AdvancedEEE" || changes[2].Keyword != "GigaLite" {
		t.Fatalf("unexpected changes: %#v", changes)
	}
}

func TestDecodeNetworkPropertiesKeepsArrayShape(t *testing.T) {
	fixtures := []struct {
		json  string
		count int
	}{
		{`[]`, 0},
		{`[{"interface_guid":"00000000-0000-0000-0000-000000000001","keyword":"*EEE","values":["1"]}]`, 1},
		{`[{"keyword":"*EEE","values":["1"]},{"keyword":"*InterruptModeration","values":["0"]}]`, 2},
	}
	for _, fixture := range fixtures {
		properties, err := decodeNetworkProperties([]byte(fixture.json))
		if err != nil || len(properties) != fixture.count {
			t.Fatalf("%s: count=%d err=%v", fixture.json, len(properties), err)
		}
	}
	if _, err := decodeNetworkProperties([]byte(`{"keyword":"*EEE"}`)); err == nil {
		t.Fatal("PowerShell 5.1 singleton object must be rejected")
	}
}

func TestBackupValidationRejectsChangedTarget(t *testing.T) {
	profile, _ := profileByID(profileRecommended)
	backup := Backup{FormatVersion: 1, CatalogVersion: 1, Profile: profile.ID, TargetUserSID: "S-1-5-21-1"}
	for _, change := range profile.Changes {
		backup.Registry = append(backup.Registry, RegSnapshot{Change: change})
	}
	backup.CatalogDigest = backupTargetDigest(backup.Registry)
	if err := validateBackup(backup); err != nil {
		t.Fatalf("valid backup rejected: %v", err)
	}
	backup.Registry[0].Change.Path = `SOFTWARE\Unexpected`
	if err := validateBackup(backup); err == nil {
		t.Fatal("tampered backup was accepted")
	}
}

func TestBackupSaveIsReplaceable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	backup := Backup{Path: path, FormatVersion: 1, ID: "one"}
	if err := saveBackup(&backup); err != nil {
		t.Fatal(err)
	}
	backup.ID = "two"
	if err := saveBackup(&backup); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), `"two"`) {
		t.Fatalf("backup was not replaced: %v %s", err, data)
	}
}

func TestCleanTreeDeletesOnlyOldFiles(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "old.tmp")
	newPath := filepath.Join(root, "new.tmp")
	if err := os.WriteFile(oldPath, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-72 * time.Hour)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	result, err := cleanTree(root, time.Now().Add(-48*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if result.Files != 1 {
		t.Fatalf("deleted %d files, want 1", result.Files)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old file still exists: %v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("new file was removed: %v", err)
	}
}

func TestTUIHomeActionsAreClickable(t *testing.T) {
	model := &tuiModel{screen: screenHome, width: 100, height: 40, audit: Audit{Hardware: Hardware{CPUs: []CPUInfo{{Name: "CPU"}}, GPUs: []GPUInfo{{Name: "GPU", Vendor: "Test"}}}}}
	view := model.View()
	if view == "" {
		t.Fatal("empty view")
	}
	wanted := map[string]bool{"audit": false, "plan-recommended": false, "plan-maximum": false, "confirm-clean": false, "confirm-restore": false, "help": false, "exit": false}
	for _, box := range model.hitboxes {
		if _, ok := wanted[box.action]; ok {
			wanted[box.action] = true
			if action := model.actionAt((box.x1+box.x2)/2, box.y1); action != box.action {
				t.Fatalf("button %s hit-test returned %s", box.action, action)
			}
		}
	}
	for action, found := range wanted {
		if !found {
			t.Errorf("missing clickable action %s", action)
		}
	}
}

func TestTUIHeaderDoesNotWrapAtMinimumWidth(t *testing.T) {
	model := &tuiModel{screen: screenHome, width: 54, height: 24}
	view := model.View()
	for index, line := range strings.Split(view, "\n") {
		if width := lipgloss.Width(line); width > model.width {
			t.Fatalf("line %d wraps: width=%d > %d", index, width, model.width)
		}
	}
}

func TestSmallTUIStillHasMouseExit(t *testing.T) {
	model := &tuiModel{screen: screenHome, width: 40, height: 10}
	_ = model.View()
	if action := model.actionAt(5, 2); action != "exit" {
		t.Fatalf("small-screen exit hitbox returned %q", action)
	}
}

func TestAuditRefreshReturnsToAudit(t *testing.T) {
	model := &tuiModel{screen: screenAudit}
	updated, _ := model.handleAction("refresh-audit")
	current := updated.(*tuiModel)
	if current.screen != screenLoading {
		t.Fatalf("bad refresh state: %#v", current)
	}
	updated, _ = current.Update(auditMsg{audit: Audit{Version: "test"}, target: screenAudit, navigate: true})
	if updated.(*tuiModel).screen != screenAudit {
		t.Fatalf("refresh returned to %v", updated.(*tuiModel).screen)
	}
}

func TestBackgroundAuditNeverInterruptsPlanLoading(t *testing.T) {
	model := &tuiModel{screen: screenLoading, loadingText: "plan"}
	updated, _ := model.Update(auditMsg{audit: Audit{Version: "fresh"}, navigate: false})
	current := updated.(*tuiModel)
	if current.screen != screenLoading || current.audit.Version != "fresh" {
		t.Fatalf("background audit changed navigation: %#v", current)
	}
}

func TestFitTextUsesExactCellWidth(t *testing.T) {
	value := fitText("Профиль", 20)
	if width := lipgloss.Width(value); width != 20 {
		t.Fatalf("width=%d, want 20", width)
	}
}

func TestSystemToolIgnoresSpoofedEnvironment(t *testing.T) {
	old := os.Getenv("SystemRoot")
	t.Cleanup(func() { _ = os.Setenv("SystemRoot", old) })
	spoofed := t.TempDir()
	if err := os.Setenv("SystemRoot", spoofed); err != nil {
		t.Fatal(err)
	}
	path := systemTool("powercfg.exe")
	if strings.Contains(strings.ToLower(path), strings.ToLower(spoofed)) || !filepath.IsAbs(path) {
		t.Fatalf("system tool used untrusted environment: %s", path)
	}
}

func TestPowerSettingReadback(t *testing.T) {
	active, err := activePowerGUID()
	if err != nil {
		t.Fatal(err)
	}
	value, err := powerACValue(active, maximumPowerSettings[0].Subgroup, maximumPowerSettings[0].Setting)
	if err != nil {
		t.Fatal(err)
	}
	if value > 100 {
		t.Fatalf("invalid maximum processor state: %d", value)
	}
	settings := availableMaximumPowerSettings(active)
	if len(settings) < len(maximumPowerSettings) {
		t.Fatalf("required power settings missing: %#v", settings)
	}
	for _, setting := range settings[len(maximumPowerSettings):] {
		t.Logf("supported optional power setting: %s", setting.Name)
	}
}

func TestMouseParametersReadback(t *testing.T) {
	snapshot, err := captureMouseParameters()
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Captured || snapshot.Speed < 0 || snapshot.Speed > 2 {
		t.Fatalf("invalid live mouse parameters: %#v", snapshot)
	}
}

func TestTrustedPowerShellEnvironmentDropsUserModulePath(t *testing.T) {
	old := os.Getenv("PSModulePath")
	t.Cleanup(func() { _ = os.Setenv("PSModulePath", old) })
	spoofed := filepath.Join(t.TempDir(), "Modules")
	if err := os.Setenv("PSModulePath", spoofed); err != nil {
		t.Fatal(err)
	}
	environment, err := trustedPowerShellEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range environment {
		if strings.HasPrefix(strings.ToUpper(item), "PSMODULEPATH=") {
			if strings.Contains(strings.ToLower(item), strings.ToLower(spoofed)) || !strings.Contains(strings.ToLower(item), `\windows\system32\windowspowershell\v1.0\modules`) {
				t.Fatalf("untrusted PSModulePath survived: %s", item)
			}
			return
		}
	}
	t.Fatal("PSModulePath missing")
}

func TestTrustedPowerShellEnvironmentIsAllowlisted(t *testing.T) {
	for _, name := range []string{"COR_ENABLE_PROFILING", "COR_PROFILER", "COR_PROFILER_PATH_64", "UNRELATED_ATTACKER_VALUE"} {
		t.Setenv(name, "attacker-controlled")
	}
	environment, err := trustedPowerShellEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range environment {
		name, _, _ := strings.Cut(item, "=")
		if strings.HasPrefix(strings.ToUpper(name), "COR_") || strings.EqualFold(name, "UNRELATED_ATTACKER_VALUE") {
			t.Fatalf("untrusted environment survived: %s", item)
		}
	}
}

func TestInternalResultArgumentsAreRemoved(t *testing.T) {
	args, path := stripResultFile([]string{"apply", "--parent-pid", "42", "--result-file", `C:\Temp\result.json`, "--yes"})
	if path != `C:\Temp\result.json` || internalParentPID(args) != 42 {
		t.Fatalf("bad internal parsing: %#v %q", args, path)
	}
	for _, arg := range args {
		if arg == "--result-file" {
			t.Fatal("result flag leaked to command parser")
		}
	}
}

func TestParentProcessSIDIsBoundToSameExecutable(t *testing.T) {
	expected, err := currentUserSID()
	if err != nil {
		t.Fatal(err)
	}
	actual, err := userSIDFromOptimizerProcess(uint32(os.Getpid()))
	if err != nil {
		t.Fatal(err)
	}
	if actual != expected {
		t.Fatalf("parent SID=%s, want %s", actual, expected)
	}
}

func TestFailedRegistryRollbackRemainsRetryable(t *testing.T) {
	backup := Backup{
		Path:            filepath.Join(t.TempDir(), "state.json"),
		AppliedRegistry: 1,
		Registry: []RegSnapshot{{
			Change:  RegChange{ID: "invalid", Hive: "INVALID", Path: "x", Name: "y"},
			Existed: true,
		}},
	}
	if err := rollbackBackup(&backup); err == nil {
		t.Fatal("rollback unexpectedly succeeded")
	}
	if backup.AppliedRegistry != 1 || backup.Status != "rollback_failed" {
		t.Fatalf("failed rollback lost retry state: %#v", backup)
	}
}

func TestBoostSessionLockGuardsOperations(t *testing.T) {
	if err := checkBoostSession(true); err == nil {
		t.Fatal("boost bypass worked without an active session")
	}
	release, err := acquireBoostSessionLock()
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if err := checkBoostSession(true); err != nil {
		t.Fatalf("active boost session was not recognized: %v", err)
	}
	if err := checkBoostSession(false); err == nil {
		t.Fatal("normal operation was allowed during boost session")
	}
}

func TestBoostSessionFlagIsInternalOnly(t *testing.T) {
	for _, args := range [][]string{
		{"apply", "--boost-session", "--yes"},
		{"restore", "--boost-session", "--yes"},
	} {
		if err := run(args); err == nil || !strings.Contains(err.Error(), "parent-pid") {
			t.Fatalf("public boost bypass was not rejected for %v: %v", args, err)
		}
	}
}

func TestValidateGameExecutable(t *testing.T) {
	root := t.TempDir()
	valid := filepath.Join(root, "game.exe")
	if err := os.WriteFile(valid, []byte("MZfixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if path, err := validateGameExecutable(valid); err != nil || !filepath.IsAbs(path) {
		t.Fatalf("valid MZ executable rejected: path=%q err=%v", path, err)
	}

	nonEXE := filepath.Join(root, "game.bin")
	invalidPE := filepath.Join(root, "invalid.exe")
	directory := filepath.Join(root, "directory.exe")
	if err := os.WriteFile(nonEXE, []byte("MZ"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalidPE, []byte("NO"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		"relative":   "game.exe",
		"non-exe":    nonEXE,
		"invalid-pe": invalidPE,
		"directory":  directory,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := validateGameExecutable(value); err == nil {
				t.Fatalf("invalid game path accepted: %s", value)
			}
		})
	}
}
