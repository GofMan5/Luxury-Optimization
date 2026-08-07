package optimizer

import (
	"container/heap"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

const (
	storageScanMaxEntries      = 10_000_000
	storageScanMaxDirectories  = 20_000
	storageScanMaxExtensions   = 4_096
	storageScanMaxChildren     = 256
	storageScanMaxLargestFiles = 100
	storageScanMaxWarnings     = 32
	storageScanProgressBatch   = 512
	storageScanMaximumDuration = 15 * time.Minute
	storageScanCacheTTL        = 5 * time.Minute
	storageScanCacheEntries    = 24
)

var errStorageScanLimit = errors.New("storage scan entry limit reached")

type StorageScanStart struct {
	ScanID    string    `json:"scan_id"`
	Root      string    `json:"root"`
	StartedAt time.Time `json:"started_at"`
	Cached    bool      `json:"cached"`
}

type StorageScanStatus struct {
	ScanID             string             `json:"scan_id"`
	State              string             `json:"state"`
	Root               string             `json:"root"`
	StartedAt          time.Time          `json:"started_at"`
	ElapsedMS          int64              `json:"elapsed_ms"`
	FilesScanned       int64              `json:"files_scanned"`
	DirectoriesScanned int64              `json:"directories_scanned"`
	BytesScanned       int64              `json:"bytes_scanned"`
	Skipped            int64              `json:"skipped"`
	CurrentPath        string             `json:"current_path,omitempty"`
	Error              string             `json:"error,omitempty"`
	Report             *StorageScanReport `json:"report,omitempty"`
	Cached             bool               `json:"cached"`
}

type StorageScanReport struct {
	Root        string                 `json:"root"`
	Volume      StorageVolume          `json:"volume"`
	GeneratedAt time.Time              `json:"generated_at"`
	ElapsedMS   int64                  `json:"elapsed_ms"`
	TotalBytes  int64                  `json:"total_bytes"`
	Files       int64                  `json:"files"`
	Directories int64                  `json:"directories"`
	Skipped     int64                  `json:"skipped"`
	Partial     bool                   `json:"partial"`
	Parent      *StorageScanNode       `json:"parent,omitempty"`
	Children    []StorageScanNode      `json:"children"`
	Largest     []StorageScanFile      `json:"largest_files"`
	Extensions  []StorageScanExtension `json:"extensions"`
	Warnings    []string               `json:"warnings,omitempty"`
}

type StorageScanNode struct {
	ID          string `json:"id,omitempty"`
	Deletable   bool   `json:"deletable"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	SizeBytes   int64  `json:"size_bytes"`
	Files       int64  `json:"files"`
	Directories int64  `json:"directories"`
}

type StorageScanFile struct {
	ID        string `json:"id,omitempty"`
	Deletable bool   `json:"deletable"`
	Name      string `json:"name"`
	Relative  string `json:"relative_path"`
	Extension string `json:"extension"`
	SizeBytes int64  `json:"size_bytes"`
	path      string
	identity  os.FileInfo
}

type StorageScanExtension struct {
	Extension string `json:"extension"`
	SizeBytes int64  `json:"size_bytes"`
	Files     int64  `json:"files"`
}

type storageScanJob struct {
	mu            sync.Mutex
	id            string
	root          string
	startedAt     time.Time
	state         string
	files         int64
	directories   int64
	bytes         int64
	skipped       int64
	currentPath   string
	err           string
	report        *StorageScanReport
	allowedDirs   map[string]string
	deleteTargets map[string]storageScanTarget
	deleteTokens  map[string]storageDeleteAuthorization
	cached        bool
	cancel        context.CancelFunc
	cancelled     atomic.Bool
	done          chan struct{}
}

type storageScanAggregate struct {
	name        string
	path        string
	size        int64
	files       int64
	directories int64
	identity    os.FileInfo
}

type storageScanCacheEntry struct {
	root          string
	report        *StorageScanReport
	allowedDirs   map[string]string
	deleteTargets map[string]storageScanTarget
	cachedAt      time.Time
}

type storageScanFileHeap []StorageScanFile

func (items storageScanFileHeap) Len() int           { return len(items) }
func (items storageScanFileHeap) Less(i, j int) bool { return items[i].SizeBytes < items[j].SizeBytes }
func (items storageScanFileHeap) Swap(i, j int)      { items[i], items[j] = items[j], items[i] }
func (items *storageScanFileHeap) Push(value any)    { *items = append(*items, value.(StorageScanFile)) }
func (items *storageScanFileHeap) Pop() any {
	old := *items
	value := old[len(old)-1]
	*items = old[:len(old)-1]
	return value
}

func (s *Service) StartStorageScan(path, parentScanID, nodeID, refreshScanID string) (StorageScanStart, error) {
	root, err := s.resolveStorageScanRoot(path, parentScanID, nodeID, refreshScanID)
	if err != nil {
		return StorageScanStart{}, err
	}
	id, err := newStorageScanID()
	if err != nil {
		return StorageScanStart{}, err
	}
	if refreshScanID == "" {
		if cached, ok := s.cachedStorageScan(root); ok {
			done := make(chan struct{})
			close(done)
			startedAt := time.Now().UTC()
			job := &storageScanJob{id: id, root: root, startedAt: startedAt, state: "complete", report: cached.report, allowedDirs: cached.allowedDirs, deleteTargets: cached.deleteTargets, deleteTokens: make(map[string]storageDeleteAuthorization), cached: true, done: done, files: cached.report.Files, directories: cached.report.Directories, bytes: cached.report.TotalBytes, skipped: cached.report.Skipped}
			s.scanMu.Lock()
			s.scan = job
			s.scanMu.Unlock()
			return StorageScanStart{ScanID: id, Root: root, StartedAt: startedAt, Cached: true}, nil
		}
	}
	if !s.probeMu.TryLock() {
		return StorageScanStart{}, errors.New("another heavy diagnostic is already running")
	}
	ctx, cancel := context.WithTimeout(context.Background(), storageScanMaximumDuration)
	job := &storageScanJob{id: id, root: root, startedAt: time.Now().UTC(), state: "scanning", allowedDirs: make(map[string]string), deleteTargets: make(map[string]storageScanTarget), deleteTokens: make(map[string]storageDeleteAuthorization), cancel: cancel, done: make(chan struct{})}
	s.scanMu.Lock()
	s.scan = job
	s.scanMu.Unlock()
	go func() {
		s.runStorageScan(ctx, job)
		cancel()
		s.probeMu.Unlock()
		close(job.done)
	}()
	return StorageScanStart{ScanID: id, Root: root, StartedAt: job.startedAt}, nil
}

func (s *Service) StorageScanStatus(id string) (StorageScanStatus, error) {
	job, err := s.storageScanByID(id)
	if err != nil {
		return StorageScanStatus{}, err
	}
	return job.snapshot(), nil
}

func (s *Service) CancelStorageScan(id string) (bool, error) {
	job, err := s.storageScanByID(id)
	if err != nil {
		return false, err
	}
	job.mu.Lock()
	if job.state != "scanning" {
		job.mu.Unlock()
		return false, nil
	}
	job.cancelled.Store(true)
	job.cancel()
	job.mu.Unlock()
	select {
	case <-job.done:
	case <-time.After(2 * time.Second):
	}
	return true, nil
}

func (s *Service) storageScanByID(id string) (*storageScanJob, error) {
	if len(id) != 32 {
		return nil, errors.New("invalid storage scan ID")
	}
	s.scanMu.Lock()
	job := s.scan
	s.scanMu.Unlock()
	if job == nil || job.id != id {
		return nil, errors.New("storage scan was not found")
	}
	return job, nil
}

func (s *Service) resolveStorageScanRoot(path, parentScanID, nodeID, refreshScanID string) (string, error) {
	initial := path != "" && parentScanID == "" && nodeID == "" && refreshScanID == ""
	drill := path == "" && parentScanID != "" && nodeID != "" && refreshScanID == ""
	refresh := path == "" && parentScanID == "" && nodeID == "" && refreshScanID != ""
	if !initial && !drill && !refresh {
		return "", errors.New("provide one volume path, directory node, or refresh scan ID")
	}
	if refresh {
		job, err := s.storageScanByID(refreshScanID)
		if err != nil {
			return "", err
		}
		job.mu.Lock()
		resolved, state := job.root, job.state
		job.mu.Unlock()
		if state != "complete" {
			return "", errors.New("storage scan is unavailable for refresh")
		}
		canonical, err := canonicalStorageScanRoot(resolved)
		if err != nil || !sameStoragePath(canonical, resolved) {
			return "", errors.New("storage refresh target changed after the scan")
		}
		return canonical, nil
	}
	if drill {
		job, err := s.storageScanByID(parentScanID)
		if err != nil {
			return "", err
		}
		job.mu.Lock()
		resolved := job.allowedDirs[nodeID]
		state := job.state
		job.mu.Unlock()
		if state != "complete" || resolved == "" {
			return "", errors.New("directory node is unavailable for drill-down")
		}
		canonical, err := canonicalStorageScanRoot(resolved)
		if err != nil {
			return "", err
		}
		if !sameStoragePath(canonical, resolved) {
			return "", errors.New("directory target changed after the storage scan")
		}
		return canonical, nil
	}
	root, err := canonicalStorageScanRoot(path)
	if err != nil {
		return "", err
	}
	volume, err := storageVolumeForPath(root)
	if err != nil {
		return "", err
	}
	volumeRoot, err := canonicalStorageScanRoot(volume.Path)
	if err != nil {
		return "", err
	}
	if !sameStoragePath(root, volumeRoot) {
		return "", errors.New("initial storage analysis must start from a listed local volume")
	}
	return root, nil
}

func canonicalStorageScanRoot(path string) (string, error) {
	if path == "" || len(path) > 4096 {
		return "", errors.New("storage scan path is required and must be at most 4096 bytes")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(filepath.Clean(absolute))
	if err != nil {
		return "", fmt.Errorf("storage scan path: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", errors.New("storage scan path must be an existing directory")
	}
	volume, err := storageVolumeForPath(resolved)
	if err != nil {
		return "", err
	}
	if volume.Kind == "remote" {
		return "", errors.New("remote storage scans are excluded")
	}
	return filepath.Clean(resolved), nil
}

func sameStoragePath(left, right string) bool {
	left, right = filepath.Clean(left), filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func newStorageScanID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func (s *Service) runStorageScan(ctx context.Context, job *storageScanJob) {
	report, allowed, targets, err := scanStorageTree(ctx, job.root, job.id, job.updateProgress)
	job.mu.Lock()
	completed := false
	switch {
	case job.cancelled.Load():
		job.state = "cancelled"
		job.err = "storage scan cancelled"
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, errStorageScanLimit):
		report.Partial = true
		report.Warnings = appendBounded(report.Warnings, "scan reached its bounded time or entry limit")
		job.state, job.report, job.allowedDirs, job.deleteTargets = "complete", &report, allowed, targets
		completed = true
	case err != nil:
		job.state = "failed"
		job.err = displayText(err.Error())
	default:
		job.state, job.report, job.allowedDirs, job.deleteTargets = "complete", &report, allowed, targets
		completed = true
	}
	job.files, job.directories, job.bytes, job.skipped = report.Files, report.Directories, report.TotalBytes, report.Skipped
	job.currentPath = ""
	job.mu.Unlock()
	if completed {
		s.cacheStorageScan(job.root, &report, allowed, targets)
	}
}

func (s *Service) cachedStorageScan(root string) (storageScanCacheEntry, bool) {
	key := storageScanCacheKey(root)
	now := time.Now()
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	entry, ok := s.scanCache[key]
	if !ok || now.Sub(entry.cachedAt) > storageScanCacheTTL {
		return storageScanCacheEntry{}, false
	}
	entry.allowedDirs = maps.Clone(entry.allowedDirs)
	entry.deleteTargets = maps.Clone(entry.deleteTargets)
	return entry, true
}

func (s *Service) cacheStorageScan(root string, report *StorageScanReport, allowed map[string]string, targets map[string]storageScanTarget) {
	key := storageScanCacheKey(root)
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	if s.scanCache == nil {
		s.scanCache = make(map[string]storageScanCacheEntry)
	}
	if _, exists := s.scanCache[key]; !exists {
		s.scanCacheOrder = append(s.scanCacheOrder, key)
	}
	s.scanCache[key] = storageScanCacheEntry{root: root, report: report, allowedDirs: maps.Clone(allowed), deleteTargets: maps.Clone(targets), cachedAt: time.Now()}
	for len(s.scanCacheOrder) > storageScanCacheEntries {
		oldest := s.scanCacheOrder[0]
		s.scanCacheOrder = s.scanCacheOrder[1:]
		delete(s.scanCache, oldest)
	}
}

func (s *Service) invalidateStorageScanCache(path string) {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	for key, entry := range s.scanCache {
		if storagePathWithin(entry.root, path) || storagePathWithin(path, entry.root) {
			delete(s.scanCache, key)
		}
	}
	kept := s.scanCacheOrder[:0]
	for _, key := range s.scanCacheOrder {
		if _, ok := s.scanCache[key]; ok {
			kept = append(kept, key)
		}
	}
	s.scanCacheOrder = kept
}

func storageScanCacheKey(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func (job *storageScanJob) updateProgress(files, directories, bytes, skipped int64, current string) {
	job.mu.Lock()
	job.files, job.directories, job.bytes, job.skipped = files, directories, bytes, skipped
	job.currentPath = boundedScanText(current, 240)
	job.mu.Unlock()
}

func (job *storageScanJob) snapshot() StorageScanStatus {
	job.mu.Lock()
	defer job.mu.Unlock()
	elapsed := time.Since(job.startedAt).Milliseconds()
	if job.report != nil {
		elapsed = job.report.ElapsedMS
	}
	return StorageScanStatus{
		ScanID: job.id, State: job.state, Root: job.root, StartedAt: job.startedAt, ElapsedMS: elapsed,
		FilesScanned: job.files, DirectoriesScanned: job.directories, BytesScanned: job.bytes, Skipped: job.skipped,
		CurrentPath: job.currentPath, Error: job.err, Report: job.report, Cached: job.cached,
	}
}

func scanStorageTree(ctx context.Context, root, tokenSeed string, progress func(int64, int64, int64, int64, string)) (StorageScanReport, map[string]string, map[string]storageScanTarget, error) {
	started := time.Now()
	volume, err := storageVolumeForPath(root)
	if err != nil {
		return StorageScanReport{}, nil, nil, err
	}
	excludedMounts, err := storageScanMountExclusions(root)
	if err != nil {
		return StorageScanReport{}, nil, nil, fmt.Errorf("storage mount boundaries: %w", err)
	}
	prefix := root
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	directories := make(map[string]*storageScanAggregate)
	overflowDirectories := &storageScanAggregate{name: "Other folders"}
	extensions := make(map[string]*StorageScanExtension)
	overflowExtensions := &StorageScanExtension{Extension: "(other)"}
	directFiles, largest := &storageScanFileHeap{}, &storageScanFileHeap{}
	heap.Init(directFiles)
	heap.Init(largest)
	var totalBytes, files, directoryCount, skipped, entries, directFileBytes, directFileCount int64
	warnings := make([]string, 0, 8)
	lastProgress := time.Now()

	err = walkStorageEntries(ctx, root, func(path string, entry fs.DirEntry, info os.FileInfo, walkErr error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if entries >= storageScanMaxEntries {
			return errStorageScanLimit
		}
		entries++
		relative := ""
		relative = strings.TrimPrefix(path, prefix)
		if walkErr != nil {
			skipped++
			warnings = appendBounded(warnings, boundedScanText(relative+": "+walkErr.Error(), 512))
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if excludedMounts[path] {
			skipped++
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || storageEntryIsReparse(info) {
			skipped++
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		first, direct := firstStorageComponent(relative)
		if entry.IsDir() {
			directoryCount++
			if direct {
				if len(directories) < storageScanMaxDirectories {
					directories[first] = &storageScanAggregate{name: boundedScanText(first, 255), path: path, directories: 1, identity: info}
				} else {
					overflowDirectories.directories++
				}
			} else if aggregate := directories[first]; aggregate != nil {
				aggregate.directories++
			} else {
				overflowDirectories.directories++
			}
		} else if info.Mode().IsRegular() {
			size := max(info.Size(), 0)
			files++
			totalBytes = saturatingAdd(totalBytes, size)
			candidate := StorageScanFile{Name: boundedScanText(entry.Name(), 255), Relative: boundedScanText(relative, 1024), Extension: storageExtension(entry.Name()), SizeBytes: size, path: path, identity: info}
			pushLargest(largest, candidate, storageScanMaxLargestFiles)
			addExtension(extensions, overflowExtensions, candidate.Extension, size)
			if direct {
				directFileBytes = saturatingAdd(directFileBytes, size)
				directFileCount++
				pushLargest(directFiles, candidate, storageScanMaxChildren)
			} else if aggregate := directories[first]; aggregate != nil {
				aggregate.size = saturatingAdd(aggregate.size, size)
				aggregate.files++
			} else {
				overflowDirectories.size = saturatingAdd(overflowDirectories.size, size)
				overflowDirectories.files++
			}
		} else {
			skipped++
		}
		if entries%storageScanProgressBatch == 0 || time.Since(lastProgress) >= 250*time.Millisecond {
			progress(files, directoryCount, totalBytes, skipped, relative)
			lastProgress = time.Now()
		}
		return nil
	})
	progress(files, directoryCount, totalBytes, skipped, "")

	report := StorageScanReport{
		Root: root, Volume: volume, GeneratedAt: time.Now().UTC(), ElapsedMS: time.Since(started).Milliseconds(),
		TotalBytes: totalBytes, Files: files, Directories: directoryCount, Skipped: skipped, Warnings: warnings,
	}
	allowed := make(map[string]string)
	targets := make(map[string]storageScanTarget)
	report.Children = buildStorageChildren(tokenSeed, root, directories, overflowDirectories, directFiles, directFileBytes, directFileCount, allowed, targets)
	report.Largest = descendingFiles(largest)
	for index := range report.Largest {
		file := &report.Largest[index]
		file.ID = registerStorageTarget(tokenSeed, root, file.path, file.Name, "file", file.SizeBytes, 1, 0, file.identity, targets)
		_, file.Deletable = targets[file.ID]
	}
	report.Extensions = buildStorageExtensions(extensions, overflowExtensions)
	if parent, ok := storageScanParent(root, volume); ok {
		parent.ID = storageNodeID(tokenSeed, parent.Name+"\x00"+parent.Kind)
		report.Parent = &parent
		allowed[parent.ID] = filepath.Dir(root)
	}
	return report, allowed, targets, err
}

func firstStorageComponent(relative string) (string, bool) {
	if index := strings.IndexRune(relative, filepath.Separator); index >= 0 {
		return relative[:index], false
	}
	return relative, true
}

func storageExtension(name string) string {
	extension := strings.ToLower(filepath.Ext(name))
	if extension == "" {
		return "(none)"
	}
	if len(extension) > 24 {
		return "(other)"
	}
	return boundedScanText(extension, 24)
}

func addExtension(values map[string]*StorageScanExtension, overflow *StorageScanExtension, extension string, size int64) {
	value := values[extension]
	if value == nil {
		if len(values) >= storageScanMaxExtensions {
			overflow.SizeBytes = saturatingAdd(overflow.SizeBytes, size)
			overflow.Files++
			return
		}
		value = &StorageScanExtension{Extension: extension}
		values[extension] = value
	}
	value.SizeBytes = saturatingAdd(value.SizeBytes, size)
	value.Files++
}

func pushLargest(values *storageScanFileHeap, item StorageScanFile, limit int) {
	if values.Len() < limit {
		heap.Push(values, item)
		return
	}
	if (*values)[0].SizeBytes < item.SizeBytes {
		heap.Pop(values)
		heap.Push(values, item)
	}
}

func descendingFiles(values *storageScanFileHeap) []StorageScanFile {
	result := append([]StorageScanFile(nil), (*values)...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].SizeBytes != result[j].SizeBytes {
			return result[i].SizeBytes > result[j].SizeBytes
		}
		return strings.ToLower(result[i].Relative) < strings.ToLower(result[j].Relative)
	})
	return result
}

func buildStorageChildren(seed, root string, directories map[string]*storageScanAggregate, overflow *storageScanAggregate, directFiles *storageScanFileHeap, directFileBytes, directFileCount int64, allowed map[string]string, targets map[string]storageScanTarget) []StorageScanNode {
	values := make([]storageScanNodePath, 0, len(directories)+directFiles.Len()+2)
	for _, value := range directories {
		values = append(values, storageScanNodePath{StorageScanNode: StorageScanNode{Name: value.name, Kind: "directory", SizeBytes: value.size, Files: value.files, Directories: value.directories}, path: value.path, identity: value.identity})
	}
	if overflow.size > 0 || overflow.directories > 0 {
		values = append(values, storageScanNodePath{StorageScanNode: StorageScanNode{Name: overflow.name, Kind: "other", SizeBytes: overflow.size, Files: overflow.files, Directories: overflow.directories}})
	}
	directTop := descendingFiles(directFiles)
	var representedBytes int64
	for _, value := range directTop {
		representedBytes = saturatingAdd(representedBytes, value.SizeBytes)
		values = append(values, storageScanNodePath{StorageScanNode: StorageScanNode{Name: value.Name, Kind: "file", SizeBytes: value.SizeBytes, Files: 1}, path: value.path, identity: value.identity})
	}
	if omittedBytes := max(directFileBytes-representedBytes, 0); omittedBytes > 0 {
		values = append(values, storageScanNodePath{StorageScanNode: StorageScanNode{Name: "Other files", Kind: "other", SizeBytes: omittedBytes, Files: max(directFileCount-int64(len(directTop)), 0)}})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].SizeBytes != values[j].SizeBytes {
			return values[i].SizeBytes > values[j].SizeBytes
		}
		return strings.ToLower(values[i].Name) < strings.ToLower(values[j].Name)
	})
	if len(values) > storageScanMaxChildren {
		omitted := storageScanNodePath{StorageScanNode: StorageScanNode{Name: "Other", Kind: "other"}}
		for _, value := range values[storageScanMaxChildren-1:] {
			omitted.SizeBytes = saturatingAdd(omitted.SizeBytes, value.SizeBytes)
			omitted.Files = saturatingAdd(omitted.Files, value.Files)
			omitted.Directories = saturatingAdd(omitted.Directories, value.Directories)
		}
		values = append(values[:storageScanMaxChildren-1], omitted)
	}
	result := make([]StorageScanNode, len(values))
	for index, value := range values {
		result[index] = value.StorageScanNode
		if value.path != "" && (value.Kind == "directory" || value.Kind == "file") {
			result[index].ID = registerStorageTarget(seed, root, value.path, value.Name, value.Kind, value.SizeBytes, value.Files, value.Directories, value.identity, targets)
			_, result[index].Deletable = targets[result[index].ID]
			if value.Kind == "directory" {
				allowed[result[index].ID] = value.path
			}
		}
	}
	return result
}

type storageScanNodePath struct {
	StorageScanNode
	path     string
	identity os.FileInfo
}

func registerStorageTarget(seed, root, path, name, kind string, size, files, directories int64, identity os.FileInfo, targets map[string]storageScanTarget) string {
	if path == "" {
		return ""
	}
	id := storageNodeID(seed, path)
	if identity != nil && !storageDeleteTargetProtected(root, path, identity) {
		targets[id] = storageScanTarget{path: path, name: name, kind: kind, size: size, files: files, directories: directories, identity: identity}
	}
	return id
}

func buildStorageExtensions(values map[string]*StorageScanExtension, overflow *StorageScanExtension) []StorageScanExtension {
	result := make([]StorageScanExtension, 0, len(values)+1)
	for _, value := range values {
		result = append(result, *value)
	}
	if overflow.Files > 0 {
		merged := false
		for index := range result {
			if result[index].Extension == overflow.Extension {
				result[index].SizeBytes = saturatingAdd(result[index].SizeBytes, overflow.SizeBytes)
				result[index].Files = saturatingAdd(result[index].Files, overflow.Files)
				merged = true
				break
			}
		}
		if !merged {
			result = append(result, *overflow)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SizeBytes != result[j].SizeBytes {
			return result[i].SizeBytes > result[j].SizeBytes
		}
		return result[i].Extension < result[j].Extension
	})
	if len(result) <= 64 {
		return result
	}
	other := StorageScanExtension{Extension: "(other)"}
	for _, value := range result[63:] {
		other.SizeBytes = saturatingAdd(other.SizeBytes, value.SizeBytes)
		other.Files = saturatingAdd(other.Files, value.Files)
	}
	return append(result[:63], other)
}

func storageScanParent(root string, volume StorageVolume) (StorageScanNode, bool) {
	parent := filepath.Dir(root)
	if sameStoragePath(root, volume.Path) || sameStoragePath(parent, root) {
		return StorageScanNode{}, false
	}
	parentVolume, err := storageVolumeForPath(parent)
	if err != nil || !sameStoragePath(parentVolume.Path, volume.Path) {
		return StorageScanNode{}, false
	}
	return StorageScanNode{Name: "..", Kind: "directory"}, true
}

func storageNodeID(seed, path string) string {
	sum := sha256.Sum256([]byte(seed + "\x00" + path))
	return hex.EncodeToString(sum[:12])
}

func saturatingAdd(left, right int64) int64 {
	if right > 0 && left > math.MaxInt64-right {
		return math.MaxInt64
	}
	if right < 0 && left < math.MinInt64-right {
		return math.MinInt64
	}
	return left + right
}

func appendBounded(values []string, value string) []string {
	if len(values) < storageScanMaxWarnings && value != "" {
		return append(values, value)
	}
	return values
}

func boundedScanText(value string, limit int) string {
	value = strings.Map(func(character rune) rune {
		if character < 0x20 || character == 0x7f {
			return -1
		}
		return character
	}, value)
	if len(value) <= limit {
		return value
	}
	for limit > 0 && !utf8.ValidString(value[:limit]) {
		limit--
	}
	return value[:limit] + "…"
}
