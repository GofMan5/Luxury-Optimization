package optimizer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Service is the stable application boundary used by the desktop sidecar.
// The existing platform transaction code stays behind it so legacy backup,
// registry and mutex contracts remain unchanged.
type Service struct{ tweakMu sync.Mutex }

type MutationResult struct {
	Changed  bool   `json:"changed"`
	Message  string `json:"message"`
	Artifact string `json:"artifact,omitempty"`
}

type SaveGameRequest struct {
	Path     string   `json:"path"`
	Name     string   `json:"name"`
	Profile  string   `json:"profile"`
	Priority string   `json:"priority"`
	Affinity string   `json:"affinity,omitempty"`
	Args     []string `json:"args,omitempty"`
}

type GameLaunchResult struct {
	PID       int        `json:"pid"`
	Name      string     `json:"name"`
	LaunchID  string     `json:"launch_id,omitempty"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	Warning   string     `json:"warning,omitempty"`
}

type SystemRestorePoint struct {
	SequenceNumber uint32    `json:"sequence_number"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
	RestoreType    uint32    `json:"restore_point_type"`
}

type CheckpointStatus struct {
	Ready     bool      `json:"ready"`
	Profile   string    `json:"profile"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

type UpdateStatus struct {
	LastCheck   time.Time `json:"last_check,omitempty"`
	Channel     string    `json:"channel"`
	Current     string    `json:"current"`
	Latest      string    `json:"latest,omitempty"`
	UpdateReady bool      `json:"update_ready"`
}

func NewService() *Service { return &Service{} }

func (s *Service) Audit() Audit { return collectAudit() }

func (s *Service) Plan(profile string) (Plan, error) {
	plan, err := buildPlan(profile)
	if err == nil {
		describePlan(&plan)
		decorateTweakRestoreState(&plan)
	}
	return plan, err
}

func (s *Service) ScanGames() GamesReport { return discoverGames() }

func (s *Service) SavedGames() (SavedGames, error) { return loadSavedGamesDefault() }

func (s *Service) SaveGame(request SaveGameRequest) (SavedGame, error) {
	path, err := validateGameExecutable(request.Path)
	if err != nil {
		return SavedGame{}, err
	}
	if request.Name == "" {
		request.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	if request.Profile == "" {
		request.Profile = profileMaximum
	}
	if request.Priority == "" {
		request.Priority = "normal"
	}
	affinity, err := parseAffinity(request.Affinity)
	if err != nil {
		return SavedGame{}, err
	}
	hashInput := path
	if runtime.GOOS == "windows" {
		hashInput = strings.ToLower(path)
	}
	hash := sha256.Sum256([]byte(hashInput))
	game := SavedGame{
		ID:       hex.EncodeToString(hash[:])[:12],
		Name:     request.Name,
		Path:     path,
		Profile:  request.Profile,
		Priority: request.Priority,
		Affinity: uint64(affinity),
		Args:     append([]string(nil), request.Args...),
	}
	if err := validateSavedGame(game, true); err != nil {
		return SavedGame{}, err
	}
	release, err := acquireGameProfilesLock()
	if err != nil {
		return SavedGame{}, err
	}
	defer release()
	store, err := loadSavedGamesDefault()
	if err != nil {
		return SavedGame{}, err
	}
	replaced := false
	for index := range store.Games {
		if store.Games[index].ID != game.ID {
			continue
		}
		if !sameGamePath(store.Games[index].Path, game.Path) {
			return SavedGame{}, errors.New("обнаружена коллизия ID игрового профиля")
		}
		store.Games[index], replaced = game, true
		break
	}
	if !replaced {
		store.Games = append(store.Games, game)
	}
	if err := saveSavedGamesDefault(store); err != nil {
		return SavedGame{}, err
	}
	return game, nil
}

func (s *Service) RemoveGame(id string) error {
	if !savedGameIDPattern.MatchString(id) {
		return errors.New("укажите корректный ID игрового профиля")
	}
	release, err := acquireGameProfilesLock()
	if err != nil {
		return err
	}
	defer release()
	store, err := loadSavedGamesDefault()
	if err != nil {
		return err
	}
	found := false
	games := store.Games[:0]
	for _, game := range store.Games {
		if game.ID == id {
			found = true
			continue
		}
		games = append(games, game)
	}
	if !found {
		return fmt.Errorf("игровой профиль %q не найден", id)
	}
	store.Games = games
	return saveSavedGamesDefault(store)
}

func (s *Service) LaunchGame(id string) (GameLaunchResult, error) {
	if !savedGameIDPattern.MatchString(id) {
		return GameLaunchResult{}, errors.New("укажите корректный ID игрового профиля")
	}
	store, err := loadSavedGamesDefault()
	if err != nil {
		return GameLaunchResult{}, err
	}
	for _, game := range store.Games {
		if game.ID != id {
			continue
		}
		if err := validateSavedGame(game, true); err != nil {
			return GameLaunchResult{}, err
		}
		args := []string{"boost", "--game", game.Path, "--profile", game.Profile, "--priority", game.Priority}
		if game.Affinity != 0 {
			args = append(args, "--affinity", fmt.Sprintf("0x%X", game.Affinity))
		}
		args = append(args, "--")
		args = append(args, game.Args...)
		pid, err := launchRecoveryCLI(args)
		result := GameLaunchResult{PID: pid, Name: game.Name}
		if err != nil {
			return result, err
		}
		record, historyErr := recordGameLaunch(game, pid)
		if historyErr != nil {
			result.Warning = "Игра запущена, но история запуска не сохранена: " + displayText(historyErr.Error())
		} else {
			result.LaunchID, result.StartedAt = record.ID, &record.StartedAt
		}
		return result, nil
	}
	return GameLaunchResult{}, fmt.Errorf("игровой профиль %q не найден", id)
}

func (s *Service) Startup() StartupReport { return listStartupEntries() }

func (s *Service) SetStartup(name string, enabled bool) error {
	if err := validateStartupName(name); err != nil {
		return err
	}
	release, err := acquireOperationLock()
	if err != nil {
		return err
	}
	defer release()
	if enabled {
		return enableStartup(name)
	}
	return disableStartup(name)
}

func (s *Service) Services(state, match string) (ServicesReport, error) {
	return listServices(state, match)
}

func (s *Service) NetworkInterfaces() ([]NetworkInterface, error) {
	return listNetworkInterfaces()
}

func (s *Service) NetworkTest(address string, count, timeoutMS int) (LatencyReport, error) {
	return measureTCPLatency(address, count, time.Duration(timeoutMS)*time.Millisecond)
}

func (s *Service) CompareBenchmarks(before, after BenchmarkSet) (BenchmarkComparison, error) {
	if err := validateBenchmarkSet(before); err != nil {
		return BenchmarkComparison{}, fmt.Errorf("before: %w", err)
	}
	if err := validateBenchmarkSet(after); err != nil {
		return BenchmarkComparison{}, fmt.Errorf("after: %w", err)
	}
	return compareBenchmarks(before, after), nil
}

func (s *Service) Clean(days int) (CleanResult, error) { return cleanTemporaryFiles(days) }

func (s *Service) UpdateStatus() (UpdateStatus, error) {
	config, err := loadUpdateConfig()
	if err != nil {
		config = updateConfig{}
	}
	return UpdateStatus{LastCheck: config.LastCheck, Channel: releaseChannel, Current: version}, nil
}

func (s *Service) CheckUpdate(ctx context.Context) (UpdateStatus, error) {
	status, err := s.UpdateStatus()
	if err != nil {
		return status, err
	}
	release, newer, err := checkForUpdate(ctx)
	if err != nil {
		return status, err
	}
	status.Latest, status.UpdateReady = release.TagName, newer
	config, err := loadUpdateConfig()
	if err == nil {
		config.LastCheck = time.Now().UTC()
		if saveErr := saveUpdateConfig(config); saveErr == nil {
			status.LastCheck = config.LastCheck
		}
	}
	return status, nil
}

func (s *Service) InstallUpdate(ctx context.Context) (MutationResult, error) {
	release, newer, err := checkForUpdate(ctx)
	if err != nil {
		return MutationResult{}, err
	}
	if !newer {
		return MutationResult{Message: "Установлена актуальная версия."}, nil
	}
	message, err := installRelease(ctx, release)
	return MutationResult{Changed: err == nil, Message: message}, err
}

func sameGamePath(left, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
