package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
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
			fmt.Println("  EXE:", displayText(executable))
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
	games, warnings := scanEpicGames()
	report.Games = append(report.Games, games...)
	report.Warnings = append(report.Warnings, warnings...)
	games, warnings = scanXboxGames()
	report.Games = append(report.Games, games...)
	report.Warnings = append(report.Warnings, warnings...)

	seen := make(map[string]bool)
	unique := report.Games[:0]
	for _, game := range report.Games {
		key := strings.ToLower(game.Source + "\x00" + game.ID + "\x00" + filepath.Clean(game.InstallDir))
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
	var roots []string
	locations := []struct {
		root registry.Key
		view uint32
	}{
		{registry.CURRENT_USER, 0},
		{registry.LOCAL_MACHINE, registry.WOW64_64KEY},
		{registry.LOCAL_MACHINE, registry.WOW64_32KEY},
	}
	for _, location := range locations {
		key, err := registry.OpenKey(location.root, `SOFTWARE\Valve\Steam`, registry.QUERY_VALUE|location.view)
		if err != nil {
			continue
		}
		for _, name := range []string{"SteamPath", "InstallPath"} {
			if value, _, err := key.GetStringValue(name); err == nil {
				roots = append(roots, filepath.Clean(filepath.FromSlash(value)))
			}
		}
		key.Close()
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
			installDir := filepath.Clean(filepath.Join(common, fields["installdir"]))
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

func scanEpicGames() ([]GameInstall, []string) {
	programData, err := programDataDirectory()
	if err != nil {
		return nil, []string{"Epic: " + err.Error()}
	}
	manifests, err := filepath.Glob(filepath.Join(programData, "Epic", "EpicGamesLauncher", "Data", "Manifests", "*.item"))
	if err != nil {
		return nil, []string{"Epic: " + err.Error()}
	}
	var games []GameInstall
	var warnings []string
	for _, manifest := range manifests {
		data, err := readSmallFile(manifest, 2<<20)
		if err != nil {
			warnings = append(warnings, "Epic manifest: "+err.Error())
			continue
		}
		var item struct {
			CatalogItemID    string `json:"CatalogItemId"`
			AppName          string `json:"AppName"`
			DisplayName      string `json:"DisplayName"`
			InstallLocation  string `json:"InstallLocation"`
			LaunchExecutable string `json:"LaunchExecutable"`
		}
		if err := json.Unmarshal(data, &item); err != nil || item.DisplayName == "" || !filepath.IsAbs(item.InstallLocation) {
			continue
		}
		installDir := filepath.Clean(item.InstallLocation)
		executables := []string{}
		if item.LaunchExecutable != "" {
			candidate := filepath.Clean(filepath.Join(installDir, filepath.FromSlash(item.LaunchExecutable)))
			if pathWithin(installDir, candidate) {
				if resolved, err := validateGameExecutable(candidate); err == nil && pathWithin(installDir, resolved) {
					executables = append(executables, resolved)
				}
			}
		}
		if len(executables) == 0 {
			executables = findGameExecutables(installDir, item.DisplayName)
		}
		id := item.CatalogItemID
		if id == "" {
			id = item.AppName
		}
		games = append(games, GameInstall{Source: "Epic", ID: id, Name: item.DisplayName, InstallDir: installDir, Executables: executables})
	}
	return games, warnings
}

func scanXboxGames() ([]GameInstall, []string) {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil, []string{"Xbox: " + err.Error()}
	}
	var games []GameInstall
	for index := 0; index < 26; index++ {
		if mask&(1<<index) == 0 {
			continue
		}
		root := fmt.Sprintf("%c:\\", 'A'+index)
		pointer, _ := windows.UTF16PtrFromString(root)
		if windows.GetDriveType(pointer) != windows.DRIVE_FIXED {
			continue
		}
		xboxRoot := filepath.Join(root, "XboxGames")
		entries, readErr := os.ReadDir(xboxRoot)
		if readErr != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			installDir := filepath.Join(xboxRoot, entry.Name(), "Content")
			if info, err := os.Stat(installDir); err != nil || !info.IsDir() {
				installDir = filepath.Join(xboxRoot, entry.Name())
			}
			executables := findGameExecutables(installDir, entry.Name())
			if len(executables) > 0 {
				games = append(games, GameInstall{Source: "Xbox", ID: entry.Name(), Name: entry.Name(), InstallDir: installDir, Executables: executables})
			}
		}
	}
	return games, nil
}

func findGameExecutables(root, gameName string) []string {
	if !filepath.IsAbs(root) {
		return nil
	}
	type candidate struct {
		path  string
		score int
	}
	var candidates []candidate
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, `..\`) {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			if relative != "." && strings.Count(relative, string(filepath.Separator)) >= 4 {
				return filepath.SkipDir
			}
			lower := strings.ToLower(entry.Name())
			for _, fragment := range []string{"redist", "thirdparty", "test-artifacts", "installer", "__installer", "sdk"} {
				if relative != "." && strings.Contains(lower, fragment) {
					return filepath.SkipDir
				}
			}
			if info, err := entry.Info(); err != nil || isReparsePoint(info) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".exe") || ignoredGameExecutable(entry.Name()) {
			return nil
		}
		resolved, err := validateGameExecutable(path)
		if err == nil && pathWithin(root, resolved) {
			candidates = append(candidates, candidate{path: resolved, score: executableScore(root, resolved, gameName)})
			if len(candidates) >= 512 {
				return fs.SkipAll
			}
		}
		return nil
	})
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return strings.ToLower(candidates[i].path) < strings.ToLower(candidates[j].path)
		}
		return candidates[i].score > candidates[j].score
	})
	limit := len(candidates)
	if limit > 8 {
		limit = 8
	}
	result := make([]string, limit)
	for i := 0; i < limit; i++ {
		result[i] = candidates[i].path
	}
	return result
}

func executableScore(root, path, gameName string) int {
	relative, _ := filepath.Rel(root, path)
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	normalize := func(value string) string {
		return strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, strings.ToLower(value))
	}
	score := 40 - strings.Count(relative, string(filepath.Separator))*4
	name, executable, directory := normalize(gameName), normalize(base), normalize(filepath.Base(root))
	if executable == name || executable == directory {
		score += 100
	} else if len(name) >= 4 && len(executable) >= 4 && (strings.Contains(name, executable) || strings.Contains(executable, name)) {
		score += 35
	}
	lower := strings.ToLower(relative)
	if strings.Contains(lower, `\binaries\win64\`) || strings.Contains(lower, `\bin\win64\`) {
		score += 25
	}
	for _, fragment := range []string{"helper", "service", "editor", "compiler", "diagnostic", "inject", "console", "report", "benchmark", "server"} {
		if strings.Contains(strings.ToLower(base), fragment) {
			score -= 60
		}
	}
	for _, fragment := range []string{`\tools\`, `\test\`, `\output\`, `\python\`, `\distribution\bin\`} {
		if strings.Contains(lower, fragment) {
			score -= 50
		}
	}
	return score
}

func ignoredGameExecutable(name string) bool {
	name = strings.ToLower(name)
	for _, fragment := range []string{"unins", "installer", "updater", "vcredist", "vc_redist", "dxsetup", "ueprereq", "unitycrashhandler", "crashreport", "sndrpt", "easyanticheat", "beservice", "dotnet", "helper", "service", "editor", "compiler", "diagnostic", "inject", "console", "worldbuilder", "yt-dlp", "python"} {
		if strings.Contains(name, fragment) {
			return true
		}
	}
	return false
}

func readSmallFile(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || isReparsePoint(info) || info.Size() > limit {
		return nil, errors.New("файл не является допустимым обычным файлом: " + path)
	}
	return os.ReadFile(path)
}

func uniqueDirectories(paths []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.Clean(path)
		if !filepath.IsAbs(path) {
			continue
		}
		key := strings.ToLower(path)
		if info, err := os.Stat(path); err == nil && info.IsDir() && !seen[key] {
			seen[key] = true
			result = append(result, path)
		}
	}
	return result
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	return err == nil && relative != ".." && !strings.HasPrefix(relative, `..\`)
}
