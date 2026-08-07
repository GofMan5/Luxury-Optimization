package optimizer

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	gameHistoryVersion          = 1
	maxGameHistorySize          = 4 << 20
	maxGameLaunches             = 512
	maxGameBenchmarks           = 64
	maxGameLaunchesPerProfile   = 24
	maxGameBenchmarksPerProfile = 8
)

var gameHistoryIDPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

type GameLaunchRecord struct {
	ID          string    `json:"id"`
	GameID      string    `json:"game_id"`
	GameName    string    `json:"game_name"`
	StartedAt   time.Time `json:"started_at"`
	LauncherPID int       `json:"launcher_pid"`
	Profile     string    `json:"profile"`
	Priority    string    `json:"priority"`
	Affinity    uint64    `json:"affinity,omitempty"`
}

type GameBenchmarkAttachment struct {
	ID         string              `json:"id"`
	GameID     string              `json:"game_id"`
	CreatedAt  time.Time           `json:"created_at"`
	Before     BenchmarkSet        `json:"before"`
	After      BenchmarkSet        `json:"after"`
	Comparison BenchmarkComparison `json:"comparison"`
}

type GameHistoryStore struct {
	Version    int                       `json:"version"`
	Launches   []GameLaunchRecord        `json:"launches"`
	Benchmarks []GameBenchmarkAttachment `json:"benchmarks"`
}

type GameHistoryReport struct {
	GameID     string                    `json:"game_id"`
	Launches   []GameLaunchRecord        `json:"launches"`
	Benchmarks []GameBenchmarkAttachment `json:"benchmarks"`
}

func (s *Service) GameHistory(gameID string) (GameHistoryReport, error) {
	if !savedGameIDPattern.MatchString(gameID) {
		return GameHistoryReport{}, errors.New("invalid game profile ID")
	}
	games, err := loadSavedGamesDefault()
	if err != nil {
		return GameHistoryReport{}, err
	}
	if _, found := findSavedGame(games, gameID); !found {
		return GameHistoryReport{}, fmt.Errorf("game profile %q not found", gameID)
	}
	history, err := loadGameHistoryDefault()
	if err != nil {
		return GameHistoryReport{}, err
	}
	report := GameHistoryReport{GameID: gameID, Launches: []GameLaunchRecord{}, Benchmarks: []GameBenchmarkAttachment{}}
	for _, launch := range history.Launches {
		if launch.GameID == gameID {
			report.Launches = append(report.Launches, launch)
		}
	}
	for _, benchmark := range history.Benchmarks {
		if benchmark.GameID == gameID {
			report.Benchmarks = append(report.Benchmarks, benchmark)
		}
	}
	return report, nil
}

func (s *Service) AttachGameBenchmark(gameID string, before, after BenchmarkSet) (GameBenchmarkAttachment, error) {
	if !savedGameIDPattern.MatchString(gameID) {
		return GameBenchmarkAttachment{}, errors.New("invalid game profile ID")
	}
	if err := validateAttachedBenchmarkSet(before); err != nil {
		return GameBenchmarkAttachment{}, fmt.Errorf("before: %w", err)
	}
	if err := validateAttachedBenchmarkSet(after); err != nil {
		return GameBenchmarkAttachment{}, fmt.Errorf("after: %w", err)
	}
	release, err := acquireGameProfilesLock()
	if err != nil {
		return GameBenchmarkAttachment{}, err
	}
	defer release()
	games, err := loadSavedGamesDefault()
	if err != nil {
		return GameBenchmarkAttachment{}, err
	}
	if _, found := findSavedGame(games, gameID); !found {
		return GameBenchmarkAttachment{}, fmt.Errorf("game profile %q not found", gameID)
	}
	history, err := loadGameHistoryDefault()
	if err != nil {
		return GameBenchmarkAttachment{}, err
	}
	id, err := newGameHistoryID(history)
	if err != nil {
		return GameBenchmarkAttachment{}, err
	}
	attachment := GameBenchmarkAttachment{ID: id, GameID: gameID, CreatedAt: time.Now().UTC(), Before: before, After: after, Comparison: compareBenchmarks(before, after)}
	history.Benchmarks = append(history.Benchmarks, attachment)
	if err := saveGameHistoryDefault(history); err != nil {
		return GameBenchmarkAttachment{}, err
	}
	return attachment, nil
}

func recordGameLaunch(game SavedGame, pid int) (GameLaunchRecord, error) {
	release, err := acquireGameProfilesLock()
	if err != nil {
		return GameLaunchRecord{}, err
	}
	defer release()
	history, err := loadGameHistoryDefault()
	if err != nil {
		return GameLaunchRecord{}, err
	}
	id, err := newGameHistoryID(history)
	if err != nil {
		return GameLaunchRecord{}, err
	}
	record := GameLaunchRecord{ID: id, GameID: game.ID, GameName: game.Name, StartedAt: time.Now().UTC(), LauncherPID: pid, Profile: game.Profile, Priority: game.Priority, Affinity: game.Affinity}
	history.Launches = append(history.Launches, record)
	if err := saveGameHistoryDefault(history); err != nil {
		return GameLaunchRecord{}, err
	}
	return record, nil
}

func findSavedGame(store SavedGames, id string) (SavedGame, bool) {
	for _, game := range store.Games {
		if game.ID == id {
			return game, true
		}
	}
	return SavedGame{}, false
}

func gameHistoryPath() (string, error) {
	gamesPath, err := gameProfilesPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(gamesPath), "game-history.json"), nil
}

func loadGameHistoryDefault() (GameHistoryStore, error) {
	path, err := gameHistoryPath()
	if err != nil {
		return GameHistoryStore{}, err
	}
	return loadGameHistory(path)
}

func saveGameHistoryDefault(store GameHistoryStore) error {
	path, err := gameHistoryPath()
	if err != nil {
		return err
	}
	return saveGameHistory(path, store)
}

func loadGameHistory(path string) (GameHistoryStore, error) {
	store := GameHistoryStore{Version: gameHistoryVersion, Launches: []GameLaunchRecord{}, Benchmarks: []GameBenchmarkAttachment{}}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return store, err
	}
	if !validGameHistoryFile(info) || info.Size() <= 0 || info.Size() > maxGameHistorySize {
		return store, errors.New("game-history.json is not a valid regular file")
	}
	data, err := readSmallFile(path, maxGameHistorySize)
	if err != nil {
		return store, err
	}
	if err := json.Unmarshal(data, &store); err != nil {
		return store, err
	}
	if err := normalizeGameHistory(&store); err != nil {
		return store, err
	}
	return store, nil
}

func saveGameHistory(path string, store GameHistoryStore) error {
	store.Version = gameHistoryVersion
	if err := validateGameHistoryEntries(&store); err != nil {
		return err
	}
	pruneGameHistory(&store)
	if err := normalizeGameHistory(&store); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > maxGameHistorySize {
		return errors.New("game-history.json exceeds the size limit")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := protectGameHistoryDirectory(dir); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, "game-history-*.tmp")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err = protectGameHistoryFile(file); err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func normalizeGameHistory(store *GameHistoryStore) error {
	if store.Version != gameHistoryVersion {
		return fmt.Errorf("unsupported game-history.json version: %d", store.Version)
	}
	if len(store.Launches) > maxGameLaunches || len(store.Benchmarks) > maxGameBenchmarks {
		return errors.New("game-history.json contains too many records")
	}
	if store.Launches == nil {
		store.Launches = []GameLaunchRecord{}
	}
	if store.Benchmarks == nil {
		store.Benchmarks = []GameBenchmarkAttachment{}
	}
	if err := validateGameHistoryEntries(store); err != nil {
		return err
	}
	sort.SliceStable(store.Launches, func(i, j int) bool { return store.Launches[i].StartedAt.After(store.Launches[j].StartedAt) })
	sort.SliceStable(store.Benchmarks, func(i, j int) bool { return store.Benchmarks[i].CreatedAt.After(store.Benchmarks[j].CreatedAt) })
	return nil
}

func validateGameHistoryEntries(store *GameHistoryStore) error {
	seen := make(map[string]bool, len(store.Launches)+len(store.Benchmarks))
	for _, launch := range store.Launches {
		if err := validateGameLaunchRecord(launch); err != nil {
			return fmt.Errorf("launch %q: %w", launch.ID, err)
		}
		if seen[launch.ID] {
			return fmt.Errorf("duplicate history record %q", launch.ID)
		}
		seen[launch.ID] = true
	}
	for index := range store.Benchmarks {
		benchmark := &store.Benchmarks[index]
		if err := validateGameBenchmarkAttachment(*benchmark); err != nil {
			return fmt.Errorf("benchmark %q: %w", benchmark.ID, err)
		}
		if seen[benchmark.ID] {
			return fmt.Errorf("duplicate history record %q", benchmark.ID)
		}
		seen[benchmark.ID] = true
		benchmark.Comparison = compareBenchmarks(benchmark.Before, benchmark.After)
	}
	return nil
}

func validateGameLaunchRecord(record GameLaunchRecord) error {
	if !gameHistoryIDPattern.MatchString(record.ID) || !savedGameIDPattern.MatchString(record.GameID) {
		return errors.New("invalid ID")
	}
	if strings.TrimSpace(record.GameName) == "" || len(record.GameName) > 128 || strings.IndexFunc(record.GameName, unicode.IsControl) >= 0 {
		return errors.New("invalid game name")
	}
	if !validHistoryTime(record.StartedAt) || record.LauncherPID <= 0 || record.LauncherPID > math.MaxInt32 {
		return errors.New("invalid launch time or PID")
	}
	if _, err := profileByID(record.Profile); err != nil {
		return err
	}
	if _, err := processPriorityClass(record.Priority); err != nil {
		return err
	}
	if record.Affinity > uint64(^uintptr(0)) {
		return errors.New("affinity does not fit this architecture")
	}
	return nil
}

func validateGameBenchmarkAttachment(attachment GameBenchmarkAttachment) error {
	if !gameHistoryIDPattern.MatchString(attachment.ID) || !savedGameIDPattern.MatchString(attachment.GameID) {
		return errors.New("invalid ID")
	}
	if !validHistoryTime(attachment.CreatedAt) {
		return errors.New("invalid benchmark time")
	}
	if err := validateAttachedBenchmarkSet(attachment.Before); err != nil {
		return fmt.Errorf("before: %w", err)
	}
	if err := validateAttachedBenchmarkSet(attachment.After); err != nil {
		return fmt.Errorf("after: %w", err)
	}
	return nil
}

func validateAttachedBenchmarkSet(set BenchmarkSet) error {
	if strings.TrimSpace(set.Label) == "" || len(set.Label) > 80 || strings.IndexFunc(set.Label, unicode.IsControl) >= 0 {
		return errors.New("invalid benchmark label")
	}
	return validateBenchmarkSet(set)
}

func validHistoryTime(value time.Time) bool {
	return !value.IsZero() && value.After(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) && value.Before(time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC))
}

func pruneGameHistory(store *GameHistoryStore) {
	sort.SliceStable(store.Launches, func(i, j int) bool { return store.Launches[i].StartedAt.After(store.Launches[j].StartedAt) })
	launchCounts := make(map[string]int)
	launches := make([]GameLaunchRecord, 0, min(len(store.Launches), maxGameLaunches))
	for _, launch := range store.Launches {
		if len(launches) == maxGameLaunches {
			break
		}
		if launchCounts[launch.GameID] < maxGameLaunchesPerProfile {
			launches = append(launches, launch)
			launchCounts[launch.GameID]++
		}
	}
	store.Launches = launches

	sort.SliceStable(store.Benchmarks, func(i, j int) bool { return store.Benchmarks[i].CreatedAt.After(store.Benchmarks[j].CreatedAt) })
	benchmarkCounts := make(map[string]int)
	benchmarks := make([]GameBenchmarkAttachment, 0, min(len(store.Benchmarks), maxGameBenchmarks))
	for _, benchmark := range store.Benchmarks {
		if len(benchmarks) == maxGameBenchmarks {
			break
		}
		if benchmarkCounts[benchmark.GameID] < maxGameBenchmarksPerProfile {
			benchmarks = append(benchmarks, benchmark)
			benchmarkCounts[benchmark.GameID]++
		}
	}
	store.Benchmarks = benchmarks
}

func newGameHistoryID(store GameHistoryStore) (string, error) {
	used := make(map[string]bool, len(store.Launches)+len(store.Benchmarks))
	for _, launch := range store.Launches {
		used[launch.ID] = true
	}
	for _, benchmark := range store.Benchmarks {
		used[benchmark.ID] = true
	}
	for range 4 {
		var value [8]byte
		if _, err := rand.Read(value[:]); err != nil {
			return "", err
		}
		id := hex.EncodeToString(value[:])
		if !used[id] {
			return id, nil
		}
	}
	return "", errors.New("failed to create a unique history ID")
}
