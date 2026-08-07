package optimizer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoragePathProbeVerifiesAndRemovesTemporaryFile(t *testing.T) {
	directory := t.TempDir()
	report, err := measureStoragePath(context.Background(), directory, 8, 64)
	if err != nil || !report.Verified || !report.TemporaryFileRemoved || report.SizeBytes != 8<<20 || report.DurableWriteMBs <= 0 || report.BufferedReadMBs <= 0 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	matches, err := filepath.Glob(filepath.Join(directory, ".luxury-storage-probe-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files remain: %v err=%v", matches, err)
	}
}

func TestCancelledStoragePathProbeStillCleansUp(t *testing.T) {
	directory := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := measureStoragePath(ctx, directory, 8, 64); err == nil {
		t.Fatal("cancelled probe succeeded")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatalf("cancelled probe left files: %v err=%v", entries, err)
	}
}

func TestHeavyDiagnosticsRejectConcurrentRuns(t *testing.T) {
	service := NewService()
	service.probeMu.Lock()
	defer service.probeMu.Unlock()
	if _, err := service.StorageTest(context.Background(), t.TempDir(), 8, 64); err == nil {
		t.Fatal("concurrent storage probe accepted")
	}
	if _, err := service.NetworkBufferbloat(context.Background(), BufferbloatRequest{ProbeAddress: "1.1.1.1:443", DurationMS: 2000, Streams: 1}); err == nil {
		t.Fatal("concurrent bufferbloat probe accepted")
	}
}

func TestStorageTreeSummarizesFoldersFilesAndExtensions(t *testing.T) {
	root := t.TempDir()
	for path, size := range map[string]int{
		filepath.Join(root, "Games", "game.pak"):   4096,
		filepath.Join(root, "Cache", "shader.bin"): 2048,
		filepath.Join(root, "readme.txt"):          1024,
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, make([]byte, size), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var progressCalls int
	report, allowed, targets, err := scanStorageTree(context.Background(), root, "test-seed", func(_, _, _, _ int64, _ string) { progressCalls++ })
	if err != nil || report.TotalBytes != 7168 || report.Files != 3 || report.Directories != 2 || len(report.Children) < 3 || len(report.Largest) != 3 || progressCalls == 0 {
		t.Fatalf("report=%+v allowed=%v targets=%v progress=%d err=%v", report, allowed, targets, progressCalls, err)
	}
	var childBytes int64
	var directoryTokens int
	for _, child := range report.Children {
		childBytes += child.SizeBytes
		if child.Kind == "directory" {
			if child.ID == "" || allowed[child.ID] == "" || targets[child.ID].path == "" {
				t.Fatalf("directory is not drillable: %+v", child)
			}
			directoryTokens++
		} else if child.Kind == "file" && (child.ID == "" || targets[child.ID].path == "") {
			t.Fatalf("file is not deletable: %+v", child)
		}
	}
	if childBytes != report.TotalBytes || directoryTokens != 2 || report.Largest[0].Name != "game.pak" || report.Largest[0].ID == "" || !report.Largest[0].Deletable || targets[report.Largest[0].ID].path == "" {
		t.Fatalf("partition/largest mismatch: children=%+v largest=%+v", report.Children, report.Largest)
	}
	foundPAK := false
	for _, extension := range report.Extensions {
		foundPAK = foundPAK || extension.Extension == ".pak" && extension.Files == 1 && extension.SizeBytes == 4096
	}
	if !foundPAK {
		t.Fatalf("extension aggregation missing: %+v", report.Extensions)
	}
}

func TestStorageScanCancellationAndAuthorization(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, _, err := scanStorageTree(ctx, root, "test-seed", func(_, _, _, _ int64, _ string) {}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled scan err=%v", err)
	}
	service := NewService()
	if _, err := service.StartStorageScan(root, "", "", ""); err == nil {
		t.Fatal("arbitrary initial directory accepted")
	}
	canonical, err := canonicalStorageScanRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	service.scan = &storageScanJob{id: "0123456789abcdef0123456789abcdef", root: canonical, state: "complete", allowedDirs: map[string]string{"allowed": canonical}, startedAt: time.Now()}
	if resolved, err := service.resolveStorageScanRoot("", service.scan.id, "allowed", ""); err != nil || !sameStoragePath(resolved, canonical) {
		t.Fatalf("drill-down resolved=%q err=%v", resolved, err)
	}
	if _, err := service.resolveStorageScanRoot("", service.scan.id, "missing", ""); err == nil {
		t.Fatal("unknown drill-down token accepted")
	}
	if resolved, err := service.resolveStorageScanRoot("", "", "", service.scan.id); err != nil || !sameStoragePath(resolved, canonical) {
		t.Fatalf("refresh resolved=%q err=%v", resolved, err)
	}
}

func TestStorageChildrenAreBoundedAndPartitioned(t *testing.T) {
	directories := make(map[string]*storageScanAggregate)
	var total int64
	for index := 0; index < 400; index++ {
		name := filepath.Base(filepath.Join("root", string(rune(0x1000+index))))
		size := int64(index + 1)
		directories[name] = &storageScanAggregate{name: name, path: filepath.Join("C:\\", name), size: size, files: 1, directories: 1}
		total += size
	}
	allowed := make(map[string]string)
	targets := make(map[string]storageScanTarget)
	children := buildStorageChildren("seed", `C:\`, directories, &storageScanAggregate{}, &storageScanFileHeap{}, 0, 0, allowed, targets)
	if len(children) != storageScanMaxChildren || len(allowed) != storageScanMaxChildren-1 {
		t.Fatalf("children=%d allowed=%d", len(children), len(allowed))
	}
	var actual int64
	for _, child := range children {
		actual += child.SizeBytes
	}
	if actual != total || children[len(children)-1].Kind != "other" {
		t.Fatalf("bounded partition total=%d want=%d tail=%+v", actual, total, children[len(children)-1])
	}
}

func TestStorageDeleteRequiresPreviewAndRevalidatesTarget(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "large.bin")
	if err := os.WriteFile(path, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, allowed, targets, err := scanStorageTree(context.Background(), root, "delete-seed", func(_, _, _, _ int64, _ string) {})
	if err != nil {
		t.Fatal(err)
	}
	var nodeID string
	for _, file := range report.Largest {
		if file.Name == "large.bin" {
			nodeID = file.ID
		}
	}
	service := NewService()
	service.scan = &storageScanJob{id: "0123456789abcdef0123456789abcdef", root: root, state: "complete", report: &report, allowedDirs: allowed, deleteTargets: targets, deleteTokens: make(map[string]storageDeleteAuthorization), startedAt: time.Now()}
	preview, err := service.PreviewStorageDelete(service.scan.id, nodeID)
	if err != nil || preview.ConfirmationToken == "" || preview.Name != "large.bin" {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	if err := os.WriteFile(path, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConfirmStorageDelete(service.scan.id, preview.ConfirmationToken); err == nil {
		t.Fatal("changed target was deleted")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("changed target disappeared: %v", err)
	}
}

func TestStorageDeleteConfirmationIsSingleUseAndRecycles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "remove.me")
	if err := os.WriteFile(path, []byte("temporary"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, allowed, targets, err := scanStorageTree(context.Background(), root, "delete-seed", func(_, _, _, _ int64, _ string) {})
	if err != nil {
		t.Fatal(err)
	}
	nodeID := report.Largest[0].ID
	service := NewService()
	service.scan = &storageScanJob{id: "abcdef0123456789abcdef0123456789", root: root, state: "complete", report: &report, allowedDirs: allowed, deleteTargets: targets, deleteTokens: make(map[string]storageDeleteAuthorization), startedAt: time.Now()}
	preview, err := service.PreviewStorageDelete(service.scan.id, nodeID)
	if err != nil {
		t.Fatal(err)
	}
	previousTrash := storageTrashTarget
	storageTrashTarget = func(path string) error { return os.Remove(path) }
	defer func() { storageTrashTarget = previousTrash }()
	result, err := service.ConfirmStorageDelete(service.scan.id, preview.ConfirmationToken)
	if err != nil || !result.Deleted || !result.Recycled {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if _, err := service.ConfirmStorageDelete(service.scan.id, preview.ConfirmationToken); err == nil {
		t.Fatal("confirmation token was reusable")
	}
}

func TestStorageScanReusesVisitedFolderUntilExplicitRefresh(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatal(err)
	}
	canonical, err := canonicalStorageScanRoot(child)
	if err != nil {
		t.Fatal(err)
	}
	report := &StorageScanReport{Root: canonical, GeneratedAt: time.Now().UTC(), Children: []StorageScanNode{}, Largest: []StorageScanFile{}, Extensions: []StorageScanExtension{}}
	service := NewService()
	service.cacheStorageScan(canonical, report, map[string]string{}, map[string]storageScanTarget{})
	service.scan = &storageScanJob{id: "0123456789abcdef0123456789abcdef", root: parent, state: "complete", allowedDirs: map[string]string{"child": canonical}, startedAt: time.Now()}
	service.probeMu.Lock()
	defer service.probeMu.Unlock()
	started, err := service.StartStorageScan("", service.scan.id, "child", "")
	if err != nil || !started.Cached {
		t.Fatalf("cached start=%+v err=%v", started, err)
	}
	status, err := service.StorageScanStatus(started.ScanID)
	if err != nil || status.State != "complete" || !status.Cached || status.Report != report {
		t.Fatalf("cached status=%+v err=%v", status, err)
	}
	if _, err := service.StartStorageScan("", "", "", started.ScanID); err == nil {
		t.Fatal("explicit refresh reused cache while heavy diagnostic lock was held")
	}
	service.invalidateStorageScanCache(filepath.Join(canonical, "deleted.bin"))
	if _, ok := service.cachedStorageScan(canonical); ok {
		t.Fatal("parent cache survived a child mutation")
	}
}
