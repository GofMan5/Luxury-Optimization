package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type GameInstall struct {
	Source      string   `json:"source"`
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	InstallDir  string   `json:"install_dir"`
	Executables []string `json:"executables,omitempty"`
}

type GamesReport struct {
	Games    []GameInstall `json:"games"`
	Warnings []string      `json:"warnings,omitempty"`
}

var (
	vdfLinePattern = regexp.MustCompile(`(?m)^\s*"([^"]+)"\s*"((?:\\.|[^"])*)"`)
	numericVDFKey  = regexp.MustCompile(`^\d+$`)
)

func gamesCommand(args []string) error {
	if len(args) > 0 && args[0] != "scan" && !strings.HasPrefix(args[0], "-") {
		return savedGamesCommand(args)
	}
	if len(args) > 0 && args[0] == "scan" {
		args = args[1:]
	}
	set := flag.NewFlagSet("games scan", flag.ContinueOnError)
	jsonOnly := set.Bool("json", false, "вывести JSON")
	if err := set.Parse(args); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("лишние аргументы games scan")
	}
	report := discoverGames()
	if *jsonOnly {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}
	for _, game := range report.Games {
		fmt.Printf("[%s] %s — %s\n", displayText(game.Source), displayText(game.Name), displayText(game.InstallDir))
		for _, executable := range game.Executables {
			fmt.Println("  BIN:", displayText(executable))
		}
	}
	for _, warning := range report.Warnings {
		fmt.Println("Предупреждение:", displayText(warning))
	}
	return nil
}

func discoverGames() GamesReport {
	report := GamesReport{Games: []GameInstall{}}
	for _, root := range steamRoots() {
		games, warnings := scanSteamRoot(root)
		report.Games = append(report.Games, games...)
		report.Warnings = append(report.Warnings, warnings...)
	}
	seen := make(map[string]bool)
	unique := report.Games[:0]
	for _, game := range report.Games {
		key := game.Source + "\x00" + game.ID + "\x00" + filepath.Clean(game.InstallDir)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, game)
		}
	}
	report.Games = unique
	sort.Slice(report.Games, func(i, j int) bool {
		return strings.ToLower(report.Games[i].Name+report.Games[i].Source) < strings.ToLower(report.Games[j].Name+report.Games[j].Source)
	})
	return report
}

func steamRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	roots := []string{
		filepath.Join(home, ".steam", "steam"),
		filepath.Join(home, ".local", "share", "Steam"),
		filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam"),
	}
	return uniqueDirectories(roots)
}

func scanSteamRoot(root string) ([]GameInstall, []string) {
	libraries := []string{root}
	data, err := readSmallFile(filepath.Join(root, "steamapps", "libraryfolders.vdf"), 4<<20)
	if err == nil {
		for _, pair := range parseVDF(data) {
			if strings.EqualFold(pair[0], "path") || numericVDFKey.MatchString(pair[0]) {
				value := filepath.Clean(filepath.FromSlash(strings.ReplaceAll(pair[1], `\\`, `\`)))
				if filepath.IsAbs(value) {
					libraries = append(libraries, value)
				}
			}
		}
	}
	libraries = uniqueDirectories(libraries)
	var games []GameInstall
	var warnings []string
	for _, library := range libraries {
		steamApps := filepath.Join(library, "steamapps")
		manifests, globErr := filepath.Glob(filepath.Join(steamApps, "appmanifest_*.acf"))
		if globErr != nil {
			warnings = append(warnings, "Steam "+library+": "+globErr.Error())
			continue
		}
		for _, manifest := range manifests {
			data, readErr := readSmallFile(manifest, 1<<20)
			if readErr != nil {
				warnings = append(warnings, "Steam manifest: "+readErr.Error())
				continue
			}
			fields := vdfMap(data)
			if fields["appid"] == "" || fields["name"] == "" || fields["installdir"] == "" {
				continue
			}
			common := filepath.Join(steamApps, "common")
			installDir := filepath.Clean(filepath.Join(common, filepath.FromSlash(fields["installdir"])))
			if !pathWithin(common, installDir) {
				warnings = append(warnings, "Steam manifest вышел за common: "+manifest)
				continue
			}
			if fields["appid"] == "228980" || strings.Contains(strings.ToLower(fields["name"]), "redistributable") {
				continue
			}
			games = append(games, GameInstall{Source: "Steam", ID: fields["appid"], Name: fields["name"], InstallDir: installDir, Executables: findGameExecutables(installDir, fields["name"])})
		}
	}
	return games, warnings
}

func parseVDF(data []byte) [][2]string {
	matches := vdfLinePattern.FindAllSubmatch(data, -1)
	result := make([][2]string, 0, len(matches))
	for _, match := range matches {
		result = append(result, [2]string{string(match[1]), strings.ReplaceAll(string(match[2]), `\"`, `"`)})
	}
	return result
}

func vdfMap(data []byte) map[string]string {
	result := make(map[string]string)
	for _, pair := range parseVDF(data) {
		result[strings.ToLower(pair[0])] = pair[1]
	}
	return result
}

func findGameExecutables(root, gameName string) []string {
	root, err := filepath.EvalSymlinks(root)
	if err != nil || !filepath.IsAbs(root) {
		return nil
	}
	type candidate struct {
		path  string
		score int
	}
	var candidates []candidate
	visited := 0
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		visited++
		if visited > 20_000 {
			return fs.SkipAll
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil || !pathWithin(root, path) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if relative != "." && strings.Count(relative, string(filepath.Separator)) >= 4 {
				return filepath.SkipDir
			}
			name := strings.ToLower(entry.Name())
			for _, ignored := range []string{"compatdata", "shadercache", "redistribut", "proton", "crash", "installer"} {
				if strings.Contains(name, ignored) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 || ignoredGameExecutable(entry.Name()) {
			return nil
		}
		resolved, validateErr := validateGameExecutable(path)
		if validateErr == nil && pathWithin(root, resolved) {
			candidates = append(candidates, candidate{path: resolved, score: executableScore(root, resolved, gameName)})
		}
		return nil
	})
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].path < candidates[j].path
		}
		return candidates[i].score > candidates[j].score
	})
	result := make([]string, 0, min(len(candidates), 8))
	seen := make(map[string]bool)
	for _, candidate := range candidates {
		if !seen[candidate.path] {
			result = append(result, candidate.path)
			seen[candidate.path] = true
			if len(result) == 8 {
				break
			}
		}
	}
	return result
}

func executableScore(root, path, gameName string) int {
	relative, _ := filepath.Rel(root, path)
	base := filepath.Base(path)
	normalize := func(value string) string {
		value = strings.ToLower(value)
		return strings.NewReplacer(" ", "", "-", "", "_", "", ".", "").Replace(value)
	}
	score := 40 - strings.Count(relative, string(filepath.Separator))*4
	name, executable, directory := normalize(gameName), normalize(base), normalize(filepath.Base(root))
	if executable == name || executable == directory {
		score += 80
	} else if name != "" && (strings.Contains(executable, name) || strings.Contains(name, executable)) {
		score += 35
	}
	return score
}

func ignoredGameExecutable(name string) bool {
	name = strings.ToLower(name)
	for _, ignored := range []string{"crash", "report", "setup", "install", "uninstall", "helper", "launcher", "benchmark", "server", "editor", "proton", "wine", "easyanticheat", "battleye"} {
		if strings.Contains(name, ignored) {
			return true
		}
	}
	return false
}

func readSmallFile(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, errors.New("файл не является обычным или превышает лимит")
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("файл превышает лимит")
	}
	return data, nil
}

func uniqueDirectories(paths []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.Clean(path)
		info, err := os.Stat(path)
		if err == nil && filepath.IsAbs(path) && info.IsDir() && !seen[path] {
			seen[path] = true
			result = append(result, path)
		}
	}
	return result
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
