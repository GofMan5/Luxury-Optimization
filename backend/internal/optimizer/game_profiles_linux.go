package optimizer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/sys/unix"
)

const (
	gameProfilesVersion = 1
	maxGameProfilesSize = 1 << 20
)

var savedGameIDPattern = regexp.MustCompile(`^[0-9a-f]{12}$`)

type SavedGame struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Path     string   `json:"path"`
	Profile  string   `json:"profile"`
	Priority string   `json:"priority"`
	Affinity uint64   `json:"affinity,omitempty"`
	Args     []string `json:"args,omitempty"`
}

type SavedGames struct {
	Version int         `json:"version"`
	Games   []SavedGame `json:"games"`
}

func savedGamesCommand(args []string) error {
	switch args[0] {
	case "list":
		set := flag.NewFlagSet("games list", flag.ContinueOnError)
		jsonOnly := set.Bool("json", false, "вывести JSON")
		if err := set.Parse(args[1:]); err != nil {
			return err
		}
		if set.NArg() != 0 {
			return errors.New("лишние аргументы games list")
		}
		store, err := loadSavedGamesDefault()
		if err != nil {
			return err
		}
		if *jsonOnly {
			data, err := json.MarshalIndent(store, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}
		for _, game := range store.Games {
			fmt.Printf("%s  %s  [%s, %s, affinity=0x%X]\n  %s\n", game.ID, displayText(game.Name), game.Profile, game.Priority, game.Affinity, displayText(game.Path))
		}
		return nil
	case "add":
		return addSavedGame(args[1:])
	case "run":
		return runSavedGame(args[1:])
	case "remove":
		return removeSavedGame(args[1:])
	default:
		return errors.New("games поддерживает scan, add, list, run и remove")
	}
}

func addSavedGame(args []string) error {
	set := flag.NewFlagSet("games add", flag.ContinueOnError)
	path := set.String("path", "", "абсолютный путь к игре")
	name := set.String("name", "", "отображаемое имя")
	profile := set.String("profile", profileMaximum, "recommended или maximum")
	priority := set.String("priority", "normal", "normal, above-normal или high")
	affinityText := set.String("affinity", "", "необязательная CPU mask")
	if err := set.Parse(args); err != nil {
		return err
	}
	resolved, err := validateGameExecutable(*path)
	if err != nil {
		return err
	}
	if *name == "" {
		*name = filepath.Base(resolved)
	}
	affinity, err := parseAffinity(*affinityText)
	if err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(resolved))
	game := SavedGame{ID: hex.EncodeToString(hash[:])[:12], Name: *name, Path: resolved, Profile: *profile, Priority: *priority, Affinity: uint64(affinity), Args: append([]string(nil), set.Args()...)}
	if err := validateSavedGame(game, true); err != nil {
		return err
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
	replaced := false
	for index := range store.Games {
		if store.Games[index].ID == game.ID {
			if store.Games[index].Path != game.Path {
				return errors.New("обнаружена коллизия ID игрового профиля")
			}
			store.Games[index], replaced = game, true
			break
		}
	}
	if !replaced {
		store.Games = append(store.Games, game)
	}
	if err := saveSavedGamesDefault(store); err != nil {
		return err
	}
	fmt.Printf("Игровой профиль сохранён: %s (%s).\n", displayText(game.Name), game.ID)
	return nil
}

func runSavedGame(args []string) error {
	set := flag.NewFlagSet("games run", flag.ContinueOnError)
	id := set.String("id", "", "ID из games list")
	if err := set.Parse(args); err != nil {
		return err
	}
	if !savedGameIDPattern.MatchString(*id) {
		return errors.New("укажите корректный --id из games list")
	}
	store, err := loadSavedGamesDefault()
	if err != nil {
		return err
	}
	for _, game := range store.Games {
		if game.ID == *id {
			if err := validateSavedGame(game, true); err != nil {
				return err
			}
			return runBoostSession(game.Path, game.Profile, game.Priority, uintptr(game.Affinity), append(append([]string(nil), game.Args...), set.Args()...))
		}
	}
	return fmt.Errorf("игровой профиль %q не найден", *id)
}

func removeSavedGame(args []string) error {
	set := flag.NewFlagSet("games remove", flag.ContinueOnError)
	id := set.String("id", "", "ID из games list")
	yes := set.Bool("yes", false, "подтвердить")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 || !savedGameIDPattern.MatchString(*id) {
		return errors.New("укажите корректный --id из games list")
	}
	if !*yes && !confirm("Удалить игровой профиль "+*id+"?") {
		return errors.New("операция отменена")
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
		if game.ID == *id {
			found = true
			continue
		}
		games = append(games, game)
	}
	if !found {
		return fmt.Errorf("игровой профиль %q не найден", *id)
	}
	store.Games = games
	if err := saveSavedGamesDefault(store); err != nil {
		return err
	}
	fmt.Println("Игровой профиль удалён:", *id)
	return nil
}

func gameProfilesPath() (string, error) {
	base, err := xdgDirectory("XDG_CONFIG_HOME", ".config")
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "luxury-optimization", "games.json"), nil
}

func loadSavedGamesDefault() (SavedGames, error) {
	path, err := gameProfilesPath()
	if err != nil {
		return SavedGames{}, err
	}
	return loadSavedGames(path)
}

func saveSavedGamesDefault(store SavedGames) error {
	path, err := gameProfilesPath()
	if err != nil {
		return err
	}
	return saveSavedGames(path, store)
}

func loadSavedGames(path string) (SavedGames, error) {
	store := SavedGames{Version: gameProfilesVersion, Games: []SavedGame{}}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return store, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() <= 0 || info.Size() > maxGameProfilesSize {
		return store, errors.New("games.json не является допустимым обычным файлом")
	}
	data, err := readSmallFile(path, maxGameProfilesSize)
	if err != nil {
		return store, err
	}
	if err := json.Unmarshal(data, &store); err != nil {
		return store, err
	}
	if err := validateSavedGames(store, false); err != nil {
		return store, err
	}
	return store, nil
}

func saveSavedGames(path string, store SavedGames) error {
	store.Version = gameProfilesVersion
	if err := validateSavedGames(store, false); err != nil {
		return err
	}
	sort.Slice(store.Games, func(i, j int) bool {
		return strings.ToLower(store.Games[i].Name) < strings.ToLower(store.Games[j].Name)
	})
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > maxGameProfilesSize {
		return errors.New("games.json превышает допустимый размер")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, "games-*.tmp")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err = file.Chmod(0o600); err == nil {
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

func validateSavedGames(store SavedGames, requirePath bool) error {
	if store.Version != gameProfilesVersion {
		return fmt.Errorf("неподдерживаемая версия games.json: %d", store.Version)
	}
	if len(store.Games) > 256 {
		return errors.New("games.json содержит слишком много профилей")
	}
	seen := make(map[string]bool)
	for _, game := range store.Games {
		if err := validateSavedGame(game, requirePath); err != nil {
			return fmt.Errorf("профиль %q: %w", game.ID, err)
		}
		if seen[game.ID] {
			return fmt.Errorf("дублирующийся игровой профиль %q", game.ID)
		}
		seen[game.ID] = true
	}
	return nil
}

func validateSavedGame(game SavedGame, requirePath bool) error {
	if !savedGameIDPattern.MatchString(game.ID) {
		return errors.New("неверный ID")
	}
	if strings.TrimSpace(game.Name) == "" || len(game.Name) > 128 || strings.IndexFunc(game.Name, unicode.IsControl) >= 0 {
		return errors.New("неверное имя")
	}
	if _, err := profileByID(game.Profile); err != nil {
		return err
	}
	if _, err := processPriorityClass(game.Priority); err != nil {
		return err
	}
	if game.Affinity > uint64(^uintptr(0)) {
		return errors.New("affinity не помещается в архитектуру")
	}
	if game.Affinity != 0 {
		if _, err := parseAffinity("0x" + strconv.FormatUint(game.Affinity, 16)); err != nil {
			return err
		}
	}
	if !filepath.IsAbs(game.Path) {
		return errors.New("путь к игре должен быть абсолютным")
	}
	if requirePath {
		resolved, err := validateGameExecutable(game.Path)
		if err != nil {
			return err
		}
		if resolved != game.Path {
			return errors.New("канонический путь к игре изменился; добавьте профиль заново")
		}
	}
	if len(game.Args) > 64 {
		return errors.New("слишком много аргументов игры")
	}
	for _, argument := range game.Args {
		if len(argument) > 4096 || strings.IndexByte(argument, 0) >= 0 {
			return errors.New("недопустимый аргумент игры")
		}
	}
	return nil
}

func acquireGameProfilesLock() (func(), error) {
	path, err := gameProfilesPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		lock.Close()
		return nil, errors.New("игровые профили уже изменяются другим процессом")
	}
	return func() {
		_ = unix.Flock(int(lock.Fd()), unix.LOCK_UN)
		_ = lock.Close()
	}, nil
}
