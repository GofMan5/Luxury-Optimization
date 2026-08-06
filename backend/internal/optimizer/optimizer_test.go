//go:build windows

package optimizer

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
)

func TestShippedProfilesHaveNoConflictsOrDangerousTweaks(t *testing.T) {
	profileIDs := []string{profileLite, profileMedium, profileMaximum, profileLegacyRecommended, profileLegacyMaximum}
	for _, id := range profileIDs {
		profile, err := profileByID(id)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateProfile(profile); err != nil {
			t.Fatalf("%s: %v", id, err)
		}
	}
	for _, id := range profileIDs {
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

func TestProfilesAreStrictlyTieredAndLegacyCompatible(t *testing.T) {
	lite, _ := profileByID(profileLite)
	medium, _ := profileByID(profileMedium)
	maximum, _ := profileByID(profileMaximum)
	if len(lite.Changes) != 6 || len(medium.Changes) != 11 || len(maximum.Changes) != 12 {
		t.Fatalf("unexpected tier sizes: lite=%d medium=%d max=%d", len(lite.Changes), len(medium.Changes), len(maximum.Changes))
	}
	if lite.PerformancePlan || lite.NetworkLatency || !medium.PerformancePlan || medium.NetworkLatency || !maximum.PerformancePlan || !maximum.NetworkLatency {
		t.Fatalf("unexpected tier capabilities: lite=%+v medium=%+v max=%+v", lite, medium, maximum)
	}
	for _, pair := range [][2]Profile{{lite, medium}, {medium, maximum}} {
		superset := make(map[string]bool, len(pair[1].Changes))
		for _, change := range pair[1].Changes {
			superset[change.ID] = true
		}
		for _, change := range pair[0].Changes {
			if !superset[change.ID] {
				t.Fatalf("%s is not contained in %s", change.ID, pair[1].ID)
			}
		}
	}
	legacyRecommended, err := profileByID(profileLegacyRecommended)
	if err != nil || len(legacyRecommended.Changes) != 10 {
		t.Fatalf("legacy recommended profile changed: %+v %v", legacyRecommended, err)
	}
	legacyMaximum, err := profileByID(profileLegacyMaximum)
	if err != nil || len(legacyMaximum.Changes) != 12 || canonicalProfileID(legacyMaximum.ID) != profileMaximum {
		t.Fatalf("legacy maximum profile changed: %+v %v", legacyMaximum, err)
	}
	legacyBackup := Backup{
		FormatVersion: 1, CatalogVersion: 1, Profile: profileLegacyMaximum, TargetUserSID: "S-1-5-21-1",
		Power: PowerSnapshot{
			PreviousGUID: "381b4222-f694-41f0-9685-ff5bb260df2e",
			CreatedGUID:  "11111111-2222-3333-4444-555555555555",
			Settings: []PowerSetting{
				{Subgroup: processorPowerSubgroup, Setting: "bc5038f7-23e0-4960-96da-33abaf5935ec", Value: 100},
				{Subgroup: processorPowerSubgroup, Setting: "893dee8e-2bef-41e0-89c6-b55d0929964c", Value: 5},
				{Subgroup: "501a4d13-42af-4429-9fd1-a8218c268e20", Setting: "ee12f906-d277-404b-b6da-e5fa1a576df5", Value: 0},
				{Subgroup: "2a737441-1930-4402-8d77-b2bebba308a3", Setting: "48e6b7a6-50f5-4782-a5d4-53bb8f07e226", Value: 0},
				{Subgroup: processorPowerSubgroup, Setting: "36687f9e-e3a5-4dbf-b1dc-15eb381c6863", Value: 0},
				{Subgroup: processorPowerSubgroup, Setting: "be337238-0d82-4146-a960-4f3749d470c7", Value: 2},
			},
		},
		PowerApplied: true,
	}
	for _, change := range legacyMaximum.Changes {
		legacyBackup.Registry = append(legacyBackup.Registry, RegSnapshot{Change: change})
	}
	legacyBackup.CatalogDigest = backupTargetDigest(legacyBackup.Registry)
	if err := validateBackup(legacyBackup); err != nil {
		t.Fatalf("legacy 1.0.x maximum backup rejected: %v", err)
	}
}

func TestLiteAndMediumContainNoHigherRiskItems(t *testing.T) {
	for _, tier := range []struct {
		id      string
		allowed map[string]bool
	}{
		{profileLite, map[string]bool{"low": true}},
		{profileMedium, map[string]bool{"low": true, "medium": true}},
	} {
		plan, err := buildPlan(tier.id)
		if err != nil {
			t.Fatal(err)
		}
		describePlan(&plan)
		for _, item := range plan.Items {
			if !tier.allowed[item.RiskLevel] {
				t.Fatalf("%s contains %s-risk item %s", tier.id, item.RiskLevel, item.ID)
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

func TestNetworkTweakIDIsStableAndPerAdapter(t *testing.T) {
	left := NetProperty{InterfaceGUID: "00000000-0000-0000-0000-000000000001", Keyword: "*EEE"}
	right := NetProperty{InterfaceGUID: "00000000-0000-0000-0000-000000000002", Keyword: "*EEE"}
	first, repeated, other := networkTweakID(left), networkTweakID(left), networkTweakID(right)
	if first != repeated || first == other {
		t.Fatalf("network tweak IDs are not stable and adapter-scoped: %q %q", first, other)
	}
	if !tweakIDPattern.MatchString(networkTweakID(left)) {
		t.Fatalf("generated tweak ID is invalid: %q", networkTweakID(left))
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
	profile, _ := profileByID(profileLite)
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

func TestPerTweakBackupRejectsCrossTarget(t *testing.T) {
	profile, _ := profileByID(profileLite)
	change := profile.Changes[0]
	backup := Backup{
		FormatVersion: 1, CatalogVersion: 1, Profile: profile.ID, TweakID: change.ID,
		TargetUserSID: "S-1-5-21-1", Registry: []RegSnapshot{{Change: change}},
	}
	backup.CatalogDigest = backupTargetDigest(backup.Registry)
	if err := validateBackup(backup); err != nil {
		t.Fatalf("valid per-tweak backup rejected: %v", err)
	}
	backup.TweakID = profile.Changes[1].ID
	if err := validateBackup(backup); err == nil {
		t.Fatal("cross-target per-tweak backup was accepted")
	}
}

func TestMouseTweakChangesOnlySelectedLiveField(t *testing.T) {
	snapshot := MouseSnapshot{Threshold1: 6, Threshold2: 10, Speed: 1, Captured: true}
	setMouseTweakValue(&snapshot, "mouse-speed", 0)
	if snapshot.Speed != 0 || snapshot.Threshold1 != 6 || snapshot.Threshold2 != 10 {
		t.Fatalf("mouse tweak changed sibling fields: %#v", snapshot)
	}
}

func TestRegistryRollbackReadback(t *testing.T) {
	path := fmt.Sprintf(`Software\GofMan3\Optimizer\Tests\%d`, time.Now().UnixNano())
	change := dword("rollback-readback", "test", "test", "HKCU", path, "Value", 1)
	t.Cleanup(func() { _ = registry.DeleteKey(registry.CURRENT_USER, path) })
	snapshot, err := snapshotRegistry(change)
	if err != nil || snapshot.Existed {
		t.Fatalf("bad initial snapshot: %#v err=%v", snapshot, err)
	}
	if err := applyRegistry(change); err != nil {
		t.Fatal(err)
	}
	if err := restoreRegistry(snapshot); err != nil {
		t.Fatal(err)
	}
	matches, err := registrySnapshotMatches(snapshot)
	if err != nil || !matches {
		t.Fatalf("registry rollback read-back failed: matches=%t err=%v", matches, err)
	}
}

func TestBackupIDValidation(t *testing.T) {
	if !backupIDPattern.MatchString("20260801T010203.123456789Z") {
		t.Fatal("valid backup ID rejected")
	}
	for _, value := range []string{"", "../backup", "20260801T010203Z", "20260801T010203.123456789Z.json"} {
		if backupIDPattern.MatchString(value) {
			t.Fatalf("invalid backup ID accepted: %q", value)
		}
	}
	if err := run([]string{"restore", "--id", "../backup", "--yes"}); err == nil {
		t.Fatal("invalid restore ID reached the restore path")
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
	settings, err := availableMaximumPowerSettings(active)
	if err != nil {
		t.Fatal(err)
	}
	if len(settings) < len(maximumPowerSettings) {
		t.Fatalf("required power settings missing: %#v", settings)
	}
	seen := make(map[string]bool, len(settings))
	ids := make(map[string]bool, len(settings))
	for _, setting := range settings {
		key := powerSettingKey(setting)
		id := powerTweakID(setting)
		if seen[key] || ids[id] || setting.Name == "" || !allowedPowerSetting(setting) {
			t.Fatalf("invalid or duplicate native power setting: %#v", setting)
		}
		seen[key] = true
		ids[id] = true
	}
	medium, err := powerSettingsForProfile(profileMedium, active)
	if err != nil || len(medium) == 0 || len(medium) > len(mediumPowerSettingIDs) {
		t.Fatalf("invalid Medium power tier: count=%d err=%v", len(medium), err)
	}
	for _, setting := range medium {
		if !isMediumPowerSetting(powerTweakID(setting)) {
			t.Fatalf("unreviewed power setting entered Medium: %#v", setting)
		}
	}
	t.Logf("supported native High Performance settings: %d", len(settings))
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

func TestCommandsRejectUnexpectedPositionals(t *testing.T) {
	cases := [][]string{
		{"audit", "extra"},
		{"plan", "extra"},
		{"apply", "extra", "--yes"},
		{"restore", "extra", "--yes"},
		{"clean", "extra", "--yes"},
		{"startup", "list", "extra"},
		{"startup", "disable", "--name", "test", "--yes", "extra"},
		{"games", "scan", "extra"},
		{"games", "remove", "--id", "0123456789ab", "--yes", "extra"},
		{"services", "list", "extra"},
		{"network", "interfaces", "extra"},
		{"backups", "list", "extra"},
	}
	for _, args := range cases {
		if err := run(args); err == nil || !strings.Contains(err.Error(), "аргумент") {
			t.Fatalf("unexpected positional reached command %v: %v", args, err)
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

func TestProcessTuningInputValidation(t *testing.T) {
	for _, value := range []string{"normal", "above-normal", "high"} {
		if _, err := processPriorityClass(value); err != nil {
			t.Fatalf("valid priority %q rejected: %v", value, err)
		}
	}
	if _, err := processPriorityClass("realtime"); err == nil {
		t.Fatal("unsafe realtime priority accepted")
	}
	if _, err := parseAffinity("0"); err == nil {
		t.Fatal("zero affinity accepted")
	}
	if mask, err := parseAffinity("0x1"); err != nil || mask != 1 {
		t.Fatalf("valid affinity rejected: mask=%X err=%v", mask, err)
	}
}

func TestTuneChildProcessReadback(t *testing.T) {
	command := exec.Command(systemTool("ping.exe"), "-n", "30", "127.0.0.1")
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = command.Process.Kill()
		_ = command.Wait()
	}()
	if err := tuneGameProcess(uint32(command.Process.Pid), "above-normal", 1); err != nil {
		t.Fatal(err)
	}
}

func TestStartupBackupRoundTrip(t *testing.T) {
	for _, kind := range []uint32{registry.SZ, registry.EXPAND_SZ} {
		encoded := encodeStartupBackup(kind, `%LOCALAPPDATA%\Game Launcher.exe --silent`)
		actualKind, command, err := decodeStartupBackup(encoded)
		if err != nil || actualKind != kind || command != encoded[1] {
			t.Fatalf("startup backup round-trip failed: kind=%d command=%q err=%v", actualKind, command, err)
		}
	}
	if _, _, err := decodeStartupBackup([]string{"4", "binary"}); err == nil {
		t.Fatal("unsupported startup type accepted")
	}
	if err := validateStartupName("bad\nname"); err == nil {
		t.Fatal("control character accepted in startup name")
	}
}

func TestSteamLibraryDiscovery(t *testing.T) {
	root := t.TempDir()
	steamApps := filepath.Join(root, "steamapps")
	gameDir := filepath.Join(steamApps, "common", "Test Game", "Binaries", "Win64")
	if err := os.MkdirAll(gameDir, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `"AppState"
{
  "appid" "42"
  "name" "Test Game"
  "installdir" "Test Game"
}`
	if err := os.WriteFile(filepath.Join(steamApps, "appmanifest_42.acf"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(gameDir, "TestGame.exe")
	if err := os.WriteFile(executable, []byte("MZfixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	games, warnings := scanSteamRoot(root)
	if len(warnings) != 0 || len(games) != 1 || games[0].ID != "42" || len(games[0].Executables) != 1 {
		t.Fatalf("bad Steam discovery: games=%#v warnings=%#v", games, warnings)
	}
}

func TestSteamManifestCannotEscapeLibrary(t *testing.T) {
	root := t.TempDir()
	steamApps := filepath.Join(root, "steamapps")
	if err := os.MkdirAll(steamApps, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `"AppState"
{
  "appid" "666"
  "name" "Escape"
  "installdir" "..\\escape"
}`
	if err := os.WriteFile(filepath.Join(steamApps, "appmanifest_666.acf"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	games, warnings := scanSteamRoot(root)
	if len(games) != 0 || len(warnings) != 1 {
		t.Fatalf("traversal manifest accepted: games=%#v warnings=%#v", games, warnings)
	}
	if pathWithin(filepath.Join(root, "game"), filepath.Join(root, "game-escape")) {
		t.Fatal("sibling path treated as child")
	}
}

func TestSavedGamesAreAtomicAndValidated(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "game.exe")
	if err := os.WriteFile(gamePath, []byte("MZfixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	storePath := filepath.Join(root, "state", "games.json")
	store := SavedGames{Version: gameProfilesVersion, Games: []SavedGame{{
		ID: "0123456789ab", Name: "Test Game", Path: gamePath, Profile: profileMaximum, Priority: "above-normal", Affinity: 1, Args: []string{"-windowed"},
	}}}
	if err := saveSavedGames(storePath, store); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadSavedGames(storePath)
	if err != nil || len(loaded.Games) != 1 || loaded.Games[0].Args[0] != "-windowed" {
		t.Fatalf("saved game was not restored: %#v err=%v", loaded, err)
	}
	loaded.Games[0].Name = "Updated"
	if err := saveSavedGames(storePath, loaded); err != nil {
		t.Fatal(err)
	}
	loaded, err = loadSavedGames(storePath)
	if err != nil || loaded.Games[0].Name != "Updated" {
		t.Fatalf("atomic replacement failed: %#v err=%v", loaded, err)
	}
	loaded.Games[0].Priority = "realtime"
	if err := saveSavedGames(storePath, loaded); err == nil {
		t.Fatal("unsafe saved priority accepted")
	}
}

func TestTCPLatencyUsesSuccessfulSamples(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 5; i++ {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			connection.Close()
		}
	}()
	report, err := measureTCPLatency(listener.Addr().String(), 5, time.Second)
	if err != nil || report.Succeeded != 5 || report.Failed != 0 || report.P95MS < report.MedianMS {
		t.Fatalf("bad local latency report: %#v err=%v", report, err)
	}
	<-done
}

func TestServiceLabelsAreStable(t *testing.T) {
	if serviceState(svc.Running) != "running" || serviceState(svc.Stopped) != "stopped" {
		t.Fatal("service states are mislabeled")
	}
	if serviceStartType(windows.SERVICE_AUTO_START) != "automatic" || serviceStartType(windows.SERVICE_DISABLED) != "disabled" {
		t.Fatal("service start types are mislabeled")
	}
	if !isWindowsSystemService(`%SystemRoot%\System32\svchost.exe -k LocalService`) || isWindowsSystemService(`C:\Program Files\Vendor\agent.exe`) {
		t.Fatal("system service origin is mislabeled")
	}
	if !criticalWindowsService("RpcSs") || !criticalWindowsService("mpssvc") || criticalWindowsService("VendorAgent") {
		t.Fatal("critical service protection is mislabeled")
	}
	startType, delayed, err := decodeServiceBackup([]string{"v1", strconv.FormatUint(windows.SERVICE_AUTO_START, 10), "true"})
	if err != nil || startType != windows.SERVICE_AUTO_START || !delayed {
		t.Fatalf("service backup decode failed: type=%d delayed=%t err=%v", startType, delayed, err)
	}
}

func TestDecodeSystemRestorePointsKeepsArrayShape(t *testing.T) {
	points, err := decodeSystemRestorePoints([]byte(`[{"sequence_number":7,"description":"Before update","created_at":"2026-08-04T00:00:00Z","restore_point_type":0}]`))
	if err != nil || len(points) != 1 || points[0].SequenceNumber != 7 {
		t.Fatalf("points=%#v err=%v", points, err)
	}
	empty, err := decodeSystemRestorePoints([]byte(`[]`))
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty=%#v err=%v", empty, err)
	}
}

func TestBenchmarkComparisonRejectsNoiseAndFindsGain(t *testing.T) {
	before := BenchmarkSet{Label: "before", Runs: []BenchmarkRun{
		{AverageFPS: 100, Low1FPS: 70, P95FrameMS: 12},
		{AverageFPS: 101, Low1FPS: 71, P95FrameMS: 11.8},
		{AverageFPS: 99, Low1FPS: 69, P95FrameMS: 12.2},
	}}
	after := BenchmarkSet{Label: "after", Runs: []BenchmarkRun{
		{AverageFPS: 110, Low1FPS: 80, P95FrameMS: 10},
		{AverageFPS: 111, Low1FPS: 81, P95FrameMS: 9.8},
		{AverageFPS: 109, Low1FPS: 79, P95FrameMS: 10.2},
	}}
	comparison := compareBenchmarks(before, after)
	if comparison.Verdict != "measurably_improved" || !comparison.AverageFPS.Meaningful || !comparison.Low1FPS.Meaningful || !comparison.P95FrameMS.Meaningful {
		t.Fatalf("measurable gain missed: %#v", comparison)
	}
	nearNoise := compareBenchmarks(before, BenchmarkSet{Label: "same", Runs: []BenchmarkRun{
		{AverageFPS: 101, Low1FPS: 70.5, P95FrameMS: 11.9},
		{AverageFPS: 100, Low1FPS: 70, P95FrameMS: 12},
		{AverageFPS: 100.5, Low1FPS: 69.5, P95FrameMS: 12.1},
	}})
	if nearNoise.Verdict != "within_run_to_run_variance" {
		t.Fatalf("noise reported as gain: %#v", nearNoise)
	}
	if err := validateBenchmarkSet(BenchmarkSet{Runs: before.Runs[:2]}); err == nil {
		t.Fatal("two-run benchmark accepted")
	}
	tooLarge := before
	tooLarge.Runs = append([]BenchmarkRun(nil), before.Runs...)
	tooLarge.Runs[0].AverageFPS = maxBenchmarkMetric + 1
	if err := validateBenchmarkSet(tooLarge); err == nil {
		t.Fatal("unbounded benchmark metric accepted")
	}
}
