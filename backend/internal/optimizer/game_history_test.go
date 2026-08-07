package optimizer

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestGameLaunchResultOmitsUnavailableHistoryMetadata(t *testing.T) {
	data, err := json.Marshal(GameLaunchResult{PID: 42, Name: "Test Game", Warning: "history unavailable"})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"pid":42,"name":"Test Game","warning":"history unavailable"}` {
		t.Fatalf("unexpected launch result: %s", data)
	}
}

func TestGameHistoryIsAtomicBoundedAndRecomputesComparisons(t *testing.T) {
	path := filepath.Join(t.TempDir(), "game-history.json")
	gameID := "0123456789ab"
	before := BenchmarkSet{Label: "before", Runs: []BenchmarkRun{{100, 70, 12}, {101, 71, 11.9}, {99, 69, 12.1}}}
	after := BenchmarkSet{Label: "after", Runs: []BenchmarkRun{{110, 80, 10}, {111, 81, 9.9}, {109, 79, 10.1}}}
	now := time.Now().UTC().Add(-time.Minute)
	store := GameHistoryStore{Version: gameHistoryVersion}
	for index := range 30 {
		store.Launches = append(store.Launches, GameLaunchRecord{ID: fmt.Sprintf("%016x", index+1), GameID: gameID, GameName: "Test Game", StartedAt: now.Add(-time.Duration(index) * time.Minute), LauncherPID: 1000 + index, Profile: profileMaximum, Priority: "normal"})
	}
	for index := range 10 {
		store.Benchmarks = append(store.Benchmarks, GameBenchmarkAttachment{ID: fmt.Sprintf("%016x", index+101), GameID: gameID, CreatedAt: now.Add(-time.Duration(index) * time.Hour), Before: before, After: after})
	}
	invalidExtra := store
	invalidExtra.Launches = append(append([]GameLaunchRecord(nil), store.Launches...), GameLaunchRecord{ID: "000000000000ffff", GameID: gameID, GameName: "Test Game", StartedAt: now.Add(-1000 * time.Hour), Profile: profileMaximum, Priority: "normal"})
	if err := saveGameHistory(filepath.Join(t.TempDir(), "invalid-history.json"), invalidExtra); err == nil {
		t.Fatal("invalid record hidden by history pruning")
	}
	if err := saveGameHistory(path, store); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadGameHistory(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Launches) != maxGameLaunchesPerProfile || len(loaded.Benchmarks) != maxGameBenchmarksPerProfile {
		t.Fatalf("history was not pruned per game: launches=%d benchmarks=%d", len(loaded.Launches), len(loaded.Benchmarks))
	}
	if loaded.Benchmarks[0].Comparison.Verdict != "measurably_improved" || loaded.Launches[0].ID != "0000000000000001" {
		t.Fatalf("history normalization failed: %+v", loaded)
	}
	loaded.Launches[0].GameName = "Updated Game"
	if err := saveGameHistory(path, loaded); err != nil {
		t.Fatal(err)
	}
	replaced, err := loadGameHistory(path)
	if err != nil || replaced.Launches[0].GameName != "Updated Game" {
		t.Fatalf("atomic replacement failed: %+v err=%v", replaced, err)
	}
	replaced.Benchmarks[0].Before.Runs[0].AverageFPS = maxBenchmarkMetric + 1
	if err := saveGameHistory(path, replaced); err == nil {
		t.Fatal("invalid attached benchmark accepted")
	}
	unchanged, err := loadGameHistory(path)
	if err != nil || unchanged.Launches[0].GameName != "Updated Game" || unchanged.Benchmarks[0].Before.Runs[0].AverageFPS > maxBenchmarkMetric {
		t.Fatalf("failed save changed the last valid history: %+v err=%v", unchanged, err)
	}
}
