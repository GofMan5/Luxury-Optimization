package optimizer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestVersionChannelAndOrdering(t *testing.T) {
	stable, err := parseVersion("v1.0.1000")
	if err != nil || stable.patch != 1000 || stable.prerelease {
		t.Fatalf("stable parse: %+v, %v", stable, err)
	}
	dev, err := parseVersion("1.0.1000-dev")
	if err != nil || !dev.prerelease || compareVersion(stable, dev) <= 0 {
		t.Fatalf("dev ordering: %+v, %v", dev, err)
	}
	channel, err := parseChannel(releaseChannel)
	if err != nil || channel.major != 1 || channel.minor != 0 {
		t.Fatalf("release channel: %+v, %v", channel, err)
	}
	if _, err := parseVersion("1.0.999999999999999999999999999999"); err == nil {
		t.Fatal("overflowing version must be rejected")
	}
}

func TestReleaseAssetNames(t *testing.T) {
	wanted := map[[2]string]string{
		{"windows", "amd64"}: "Luxury-Optimization-windows-amd64.exe",
		{"windows", "386"}:   "Luxury-Optimization-windows-386.exe",
		{"linux", "arm64"}:   "Luxury-Optimization-linux-arm64",
	}
	for target, expected := range wanted {
		actual, err := releaseAssetName(target[0], target[1])
		if err != nil || actual != expected {
			t.Fatalf("%v: %q, %v", target, actual, err)
		}
	}
	if _, err := releaseAssetName("linux", "386"); err == nil {
		t.Fatal("linux/386 must be rejected")
	}
}

func TestChecksumManifestIsExactAndUnique(t *testing.T) {
	name := "Luxury-Optimization-linux-amd64"
	hash := strings.Repeat("a", 64)
	actual, err := checksumForAsset([]byte(hash+"  "+name+"\n"), name)
	if err != nil || actual != hash {
		t.Fatalf("checksum: %q, %v", actual, err)
	}
	if _, err := checksumForAsset([]byte(hash+"  "+name+"\n"+hash+"  "+name+"\n"), name); err == nil {
		t.Fatal("duplicate checksum must be rejected")
	}
}

func TestGitHubCheckHonorsPinnedReleaseChannel(t *testing.T) {
	tag := "v1.0.1"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(writer, `{"tag_name":%q,"html_url":"https://github.com/%s/releases/tag/%s","assets":[]}`, tag, repositorySlug, tag)
	}))
	defer server.Close()
	oldURL, oldClient, oldVersion := releaseAPIURL, updateClient, version
	oldInsecure := allowInsecureTestURLs
	releaseAPIURL, updateClient, version, allowInsecureTestURLs = server.URL, server.Client(), "1.0.0", true
	defer func() {
		releaseAPIURL, updateClient, version, allowInsecureTestURLs = oldURL, oldClient, oldVersion, oldInsecure
	}()

	release, newer, err := checkForUpdate(context.Background())
	if err != nil || !newer || release.TagName != tag {
		t.Fatalf("update check: %+v, %t, %v", release, newer, err)
	}
	tag = "v1.1.0"
	if _, _, err := checkForUpdate(context.Background()); err == nil || !strings.Contains(err.Error(), "канала 1.0.x") {
		t.Fatalf("minor release must be blocked: %v", err)
	}
}

func TestReleaseDownloadIsBoundedAndHashed(t *testing.T) {
	payload := []byte("verified release payload")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write(payload)
	}))
	defer server.Close()
	oldClient := updateClient
	oldInsecure := allowInsecureTestURLs
	updateClient, allowInsecureTestURLs = server.Client(), true
	defer func() { updateClient, allowInsecureTestURLs = oldClient, oldInsecure }()
	file, err := os.CreateTemp(t.TempDir(), "asset-*")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	actual, size, err := downloadToFile(context.Background(), server.URL, file, int64(len(payload)))
	expectedHash := sha256.Sum256(payload)
	if err != nil || size != int64(len(payload)) || actual != hex.EncodeToString(expectedHash[:]) {
		t.Fatalf("download: hash=%s size=%d err=%v", actual, size, err)
	}
}

func TestReleaseURLRejectsLocalNetworkInProduction(t *testing.T) {
	oldInsecure := allowInsecureTestURLs
	allowInsecureTestURLs = false
	defer func() { allowInsecureTestURLs = oldInsecure }()
	if err := validateReleaseURL("http://127.0.0.1/private"); err == nil {
		t.Fatal("production updater must reject loopback HTTP")
	}
	if err := validateReleaseURL("https://github.com:444/private"); err == nil {
		t.Fatal("production updater must reject non-standard HTTPS ports")
	}
}

func TestUpdateLockSerializesInstallers(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	release, err := acquireUpdateLock()
	if err != nil {
		t.Fatal(err)
	}
	if second, err := acquireUpdateLock(); err == nil {
		second()
		release()
		t.Fatal("second update lock must be rejected")
	}
	release()
	again, err := acquireUpdateLock()
	if err != nil {
		t.Fatal(err)
	}
	again()
}

func TestUpdateConfigIsReplaceable(t *testing.T) {
	dir := t.TempDir()
	oldConfigDir := userConfigDir
	userConfigDir = func() (string, error) { return dir, nil }
	defer func() { userConfigDir = oldConfigDir }()
	first := time.Unix(10, 0).UTC()
	second := time.Unix(20, 0).UTC()
	if err := saveUpdateConfig(updateConfig{LastCheck: first}); err != nil {
		t.Fatal(err)
	}
	if err := saveUpdateConfig(updateConfig{LastCheck: second}); err != nil {
		t.Fatal(err)
	}
	config, err := loadUpdateConfig()
	if err != nil || !config.LastCheck.Equal(second) {
		t.Fatalf("recovered config: %+v, %v", config, err)
	}
}

func TestCommonBenchmarkValidation(t *testing.T) {
	before := BenchmarkSet{Label: "before", Runs: []BenchmarkRun{{100, 70, 12}, {101, 71, 11.9}, {99, 69, 12.1}}}
	after := BenchmarkSet{Label: "after", Runs: []BenchmarkRun{{110, 80, 10}, {111, 81, 9.9}, {109, 79, 10.1}}}
	if err := validateBenchmarkSet(before); err != nil {
		t.Fatal(err)
	}
	if comparison := compareBenchmarks(before, after); comparison.Verdict != "measurably_improved" {
		t.Fatalf("unexpected verdict: %s", comparison.Verdict)
	}
}

func TestNetworkTestRejectsExcessiveWorstCaseDuration(t *testing.T) {
	if _, err := measureTCPLatency("127.0.0.1:1", 100, 10*time.Second); err == nil {
		t.Fatal("excessive network test must be rejected before dialing")
	}
}

func TestDisplayTextRemovesTerminalControlsAndBoundsOutput(t *testing.T) {
	value := "safe\x1b[31m\u202E" + strings.Repeat("x", 600)
	actual := displayText(value)
	if strings.ContainsAny(actual, "\x1b\u202E") || len([]rune(actual)) != 512 || !strings.HasSuffix(actual, "…") {
		t.Fatalf("unsafe display text: %q", actual)
	}
}
