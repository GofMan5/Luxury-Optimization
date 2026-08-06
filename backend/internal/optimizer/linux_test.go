//go:build linux

package optimizer

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestLinuxExecutableAndSavedProfileRoundTrip(t *testing.T) {
	game := filepath.Join(t.TempDir(), "game")
	if err := os.WriteFile(game, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	resolved, err := validateGameExecutable(game)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "games.json")
	store := SavedGames{Version: gameProfilesVersion, Games: []SavedGame{{ID: "0123456789ab", Name: "Test", Path: resolved, Profile: profileMaximum, Priority: "normal"}}}
	if err := saveSavedGames(path, store); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadSavedGames(path)
	if err != nil || len(loaded.Games) != 1 || loaded.Games[0].Path != resolved {
		t.Fatalf("round trip: %+v, %v", loaded, err)
	}
}

func TestLinuxDesktopEntryParser(t *testing.T) {
	path := filepath.Join(t.TempDir(), "game.desktop")
	data := "[Desktop Entry]\nType=Application\nName=Game\nExec=/opt/game --start\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	values, err := readDesktopEntry(path)
	if err != nil || values["Exec"] != "/opt/game --start" {
		t.Fatalf("desktop entry: %+v, %v", values, err)
	}
}

func TestLinuxStartupDisableEnableRoundTrip(t *testing.T) {
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	dir := filepath.Join(config, "autostart")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "game.desktop")
	data := "[Desktop Entry]\nType=Application\nName=Game\nExec=/opt/game\n"
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := disableStartup("game.desktop"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("entry still present: %v", err)
	}
	if err := enableStartup("game.desktop"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestLinuxUnavailableCapabilitiesRemainNonFatal(t *testing.T) {
	if _, err := buildPlan(profileRecommended); err != nil {
		t.Fatal(err)
	}
	audit := collectAudit()
	if len(audit.Capabilities) == 0 || audit.Hardware.OS.Caption == "" {
		t.Fatalf("incomplete audit: %+v", audit)
	}
}

func TestLinuxAffinityHonorsCurrentCPUSet(t *testing.T) {
	var allowed unix.CPUSet
	if err := unix.SchedGetaffinity(0, &allowed); err != nil {
		t.Skip(err)
	}
	for cpu := 0; cpu < strconv.IntSize; cpu++ {
		if allowed.IsSet(cpu) {
			mask := fmt.Sprintf("0x%x", uint64(1)<<cpu)
			if _, err := parseAffinity(mask); err != nil {
				t.Fatalf("allowed CPU %d rejected: %v", cpu, err)
			}
			return
		}
	}
	t.Skip("no representable CPU in current cpuset")
}

func TestLinuxUpdateReplacementIsAtomicAndKeepsMode(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "Luxury-Optimization-linux-amd64")
	pending := filepath.Join(dir, ".update")
	if err := os.WriteFile(current, []byte("old"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pending, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := installDownloadedUpdate(pending, current); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(current)
	if err != nil || string(data) != "new" {
		t.Fatalf("replacement: %q, %v", data, err)
	}
	info, err := os.Stat(current)
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("mode: %v, %v", info.Mode().Perm(), err)
	}
}

func TestLinuxCleanNeverRecursesIntoNonEmptyDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("clean intentionally refuses root")
	}
	dir, err := os.MkdirTemp(os.TempDir(), "luxury-optimization-clean-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	child := filepath.Join(dir, "keep")
	if err := os.WriteFile(child, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-72 * time.Hour)
	if err := os.Chtimes(dir, old, old); err != nil {
		t.Fatal(err)
	}
	if _, err := cleanTemporaryFiles(1); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(child); err != nil {
		t.Fatalf("clean removed nested data: %v", err)
	}
}
