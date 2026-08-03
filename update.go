package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	maxReleaseMetadata = 2 << 20
	maxChecksumFile    = 1 << 20
	maxReleaseAsset    = 256 << 20
)

var (
	releaseAPIURL  = "https://api.github.com/repos/" + repositorySlug + "/releases/latest"
	versionPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(-([0-9A-Za-z.-]+))?$`)
	hashPattern    = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
	updateClient   = &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many release redirects")
			}
			return validateReleaseURL(request.URL.String())
		},
	}
	allowInsecureTestURLs = false
	userConfigDir         = os.UserConfigDir
)

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"browser_download_url"`
		Size        int64  `json:"size"`
	} `json:"assets"`
}

type updateConfig struct {
	Auto      bool      `json:"auto"`
	LastCheck time.Time `json:"last_check,omitempty"`
}

type parsedVersion struct {
	major, minor, patch int
	prerelease          bool
}

func updateCommand(args []string) error {
	action := "check"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		action, args = args[0], args[1:]
	}
	switch action {
	case "status":
		set := flag.NewFlagSet("update status", flag.ContinueOnError)
		if err := set.Parse(args); err != nil {
			return err
		}
		if set.NArg() != 0 {
			return errors.New("лишние аргументы update status")
		}
		config, err := loadUpdateConfig()
		if err != nil {
			return err
		}
		fmt.Printf("Auto update: %t\nRelease channel: %s.x\nLast check: %s\n", config.Auto, releaseChannel, formatTime(config.LastCheck))
		return nil
	case "enable", "disable":
		set := flag.NewFlagSet("update "+action, flag.ContinueOnError)
		if err := set.Parse(args); err != nil {
			return err
		}
		if set.NArg() != 0 {
			return errors.New("лишние аргументы update " + action)
		}
		config, err := loadUpdateConfig()
		if err != nil {
			config = updateConfig{}
		}
		config.Auto = action == "enable"
		if err := saveUpdateConfig(config); err != nil {
			return err
		}
		fmt.Println("Auto update:", map[bool]string{true: "enabled", false: "disabled"}[config.Auto])
		return nil
	case "check", "install":
		set := flag.NewFlagSet("update "+action, flag.ContinueOnError)
		yes := set.Bool("yes", false, "подтвердить установку")
		if err := set.Parse(args); err != nil {
			return err
		}
		if set.NArg() != 0 {
			return errors.New("лишние аргументы update " + action)
		}
		release, newer, err := checkForUpdate(context.Background())
		if err != nil {
			return err
		}
		if !newer {
			fmt.Printf("Установлена актуальная версия %s в канале %s.x.\n", version, releaseChannel)
			return nil
		}
		fmt.Printf("Доступна %s: https://github.com/%s/releases/tag/%s\n", release.TagName, repositorySlug, release.TagName)
		if action == "check" {
			return nil
		}
		if !*yes && !confirm("Скачать, проверить SHA-256 и установить "+release.TagName+"?") {
			return errors.New("операция отменена")
		}
		message, err := installRelease(context.Background(), release)
		if err != nil {
			return err
		}
		fmt.Println(message)
		return nil
	default:
		return errors.New("update поддерживает check, install, status, enable и disable")
	}
}

func maybeAutoUpdate() {
	if strings.Contains(version, "-dev") {
		return
	}
	config, err := loadUpdateConfig()
	if err != nil || !config.Auto || time.Since(config.LastCheck) < 24*time.Hour {
		return
	}
	config.LastCheck = time.Now().UTC()
	_ = saveUpdateConfig(config)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	release, newer, err := checkForUpdate(ctx)
	if err != nil || !newer {
		return
	}
	message, err := installRelease(ctx, release)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Auto update пропущен:", displayText(err.Error()))
		return
	}
	fmt.Fprintln(os.Stderr, message)
}

func checkForUpdate(ctx context.Context) (githubRelease, bool, error) {
	data, err := fetchLimited(ctx, releaseAPIURL, maxReleaseMetadata)
	if err != nil {
		return githubRelease{}, false, fmt.Errorf("GitHub Release: %w", err)
	}
	var release githubRelease
	if err := json.Unmarshal(data, &release); err != nil {
		return release, false, err
	}
	latest, err := parseVersion(release.TagName)
	if err != nil {
		return release, false, fmt.Errorf("неподдерживаемый release tag %q", release.TagName)
	}
	channel, err := parseChannel(releaseChannel)
	if err != nil {
		return release, false, err
	}
	if latest.major != channel.major || latest.minor != channel.minor {
		return release, false, fmt.Errorf("latest release %s вне разрешённого канала %s.x", release.TagName, releaseChannel)
	}
	current, err := parseVersion(version)
	if err != nil {
		return release, false, fmt.Errorf("текущая версия %q некорректна", version)
	}
	return release, compareVersion(latest, current) > 0, nil
}

func installRelease(ctx context.Context, release githubRelease) (string, error) {
	releaseLock, err := acquireUpdateLock()
	if err != nil {
		return "", err
	}
	defer releaseLock()
	assetName, err := releaseAssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", err
	}
	assetURL, checksumURL, assetSize := "", "", int64(0)
	for _, asset := range release.Assets {
		switch asset.Name {
		case assetName:
			assetURL, assetSize = asset.DownloadURL, asset.Size
		case "SHA256SUMS.txt":
			checksumURL = asset.DownloadURL
		}
	}
	if assetURL == "" || checksumURL == "" {
		return "", fmt.Errorf("release %s не содержит %s и SHA256SUMS.txt", release.TagName, assetName)
	}
	if assetSize <= 0 || assetSize > maxReleaseAsset {
		return "", errors.New("размер release asset вне допустимого диапазона")
	}
	checksumData, err := fetchLimited(ctx, checksumURL, maxChecksumFile)
	if err != nil {
		return "", fmt.Errorf("SHA256SUMS.txt: %w", err)
	}
	expected, err := checksumForAsset(checksumData, assetName)
	if err != nil {
		return "", err
	}
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return "", err
	}
	pending, err := os.CreateTemp(filepath.Dir(executable), ".luxury-optimization-update-*")
	if err != nil {
		return "", fmt.Errorf("каталог приложения недоступен для обновления: %w", err)
	}
	pendingPath := pending.Name()
	defer func() {
		_ = pending.Close()
		_ = os.Remove(pendingPath)
	}()
	actual, written, err := downloadToFile(ctx, assetURL, pending, maxReleaseAsset)
	if err != nil {
		return "", err
	}
	if written != assetSize {
		return "", fmt.Errorf("размер download %d не совпал с metadata %d", written, assetSize)
	}
	if !strings.EqualFold(actual, expected) {
		return "", errors.New("SHA-256 release asset не совпал")
	}
	if err := pending.Sync(); err != nil {
		return "", err
	}
	if err := pending.Close(); err != nil {
		return "", err
	}
	message, err := installDownloadedUpdate(pendingPath, executable)
	if err != nil {
		return "", err
	}
	pendingPath = ""
	return message, nil
}

func releaseAssetName(goos, goarch string) (string, error) {
	suffix := ""
	if goos == "windows" {
		suffix = ".exe"
	}
	if (goos != "windows" && goos != "linux") || (goarch != "amd64" && goarch != "arm64" && !(goos == "windows" && goarch == "386")) {
		return "", fmt.Errorf("self-update не поддерживает %s/%s", goos, goarch)
	}
	return fmt.Sprintf("Luxury-Optimization-%s-%s%s", goos, goarch, suffix), nil
}

func checksumForAsset(data []byte, assetName string) (string, error) {
	found := ""
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != assetName {
			continue
		}
		if !hashPattern.MatchString(fields[0]) || found != "" {
			return "", errors.New("SHA256SUMS.txt содержит некорректную или дублирующую запись")
		}
		found = strings.ToLower(fields[0])
	}
	if found == "" {
		return "", fmt.Errorf("SHA256SUMS.txt не содержит %s", assetName)
	}
	return found, nil
}

func fetchLimited(ctx context.Context, rawURL string, limit int64) ([]byte, error) {
	response, err := doRequest(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.ContentLength > limit {
		return nil, errors.New("response превышает лимит")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("response превышает лимит")
	}
	return data, nil
}

func downloadToFile(ctx context.Context, rawURL string, file *os.File, limit int64) (string, int64, error) {
	response, err := doRequest(ctx, rawURL)
	if err != nil {
		return "", 0, err
	}
	defer response.Body.Close()
	if response.ContentLength > limit {
		return "", 0, errors.New("release asset превышает лимит")
	}
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, limit+1))
	if err != nil {
		return "", written, err
	}
	if written > limit {
		return "", written, errors.New("release asset превышает лимит")
	}
	return hex.EncodeToString(hash.Sum(nil)), written, nil
}

func doRequest(ctx context.Context, rawURL string) (*http.Response, error) {
	if err := validateReleaseURL(rawURL); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Accept-Encoding", "identity")
	request.Header.Set("User-Agent", "Luxury-Optimization/"+version)
	response, err := updateClient.Do(request)
	if err != nil {
		return nil, err
	}
	if err := validateReleaseURL(response.Request.URL.String()); err != nil {
		response.Body.Close()
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("HTTP %s", response.Status)
	}
	return response, nil
}

func validateReleaseURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.User != nil || parsed.Hostname() == "" {
		return errors.New("некорректный release URL")
	}
	host := strings.ToLower(parsed.Hostname())
	if allowInsecureTestURLs && parsed.Scheme == "http" && isLoopbackHost(host) {
		return nil
	}
	if parsed.Scheme != "https" {
		return errors.New("release URL должен использовать HTTPS")
	}
	if parsed.Port() != "" && parsed.Port() != "443" {
		return errors.New("release HTTPS URL должен использовать порт 443")
	}
	allowed := host == "api.github.com" || host == "github.com" || strings.HasSuffix(host, ".githubusercontent.com")
	if !allowed {
		return fmt.Errorf("недоверенный release host %q", host)
	}
	return nil
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func parseVersion(value string) (parsedVersion, error) {
	match := versionPattern.FindStringSubmatch(value)
	if match == nil {
		return parsedVersion{}, errors.New("invalid semantic version")
	}
	major, majorErr := strconv.Atoi(match[1])
	minor, minorErr := strconv.Atoi(match[2])
	patch, patchErr := strconv.Atoi(match[3])
	if majorErr != nil || minorErr != nil || patchErr != nil {
		return parsedVersion{}, errors.New("semantic version component overflow")
	}
	return parsedVersion{major: major, minor: minor, patch: patch, prerelease: match[4] != ""}, nil
}

func parseChannel(value string) (parsedVersion, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return parsedVersion{}, errors.New("invalid release channel")
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return parsedVersion{}, errors.New("invalid release channel")
	}
	return parsedVersion{major: major, minor: minor}, nil
}

func compareVersion(left, right parsedVersion) int {
	for _, pair := range [][2]int{{left.major, right.major}, {left.minor, right.minor}, {left.patch, right.patch}} {
		if pair[0] < pair[1] {
			return -1
		}
		if pair[0] > pair[1] {
			return 1
		}
	}
	if left.prerelease != right.prerelease {
		if left.prerelease {
			return -1
		}
		return 1
	}
	return 0
}

func updateConfigPath() (string, error) {
	base, err := userConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Luxury-Optimization", "update.json"), nil
}

func loadUpdateConfig() (updateConfig, error) {
	path, err := updateConfigPath()
	if err != nil {
		return updateConfig{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return updateConfig{}, nil
	}
	if err != nil {
		return updateConfig{}, err
	}
	var config updateConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return config, err
	}
	return config, nil
}

func saveUpdateConfig(config updateConfig) error {
	path, err := updateConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), "update-*.tmp")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err = file.Chmod(0o600); err == nil {
		_, err = file.Write(data)
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "never"
	}
	return value.Format(time.RFC3339)
}
