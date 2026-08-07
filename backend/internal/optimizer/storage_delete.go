package optimizer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	storageDeleteAuthorizationTTL = 45 * time.Second
	storageDeleteMaxPending       = 32
	storageDeleteTypedBytes       = 1 << 30
	storageDeleteTypedEntries     = 1_000
)

type StorageDeletePreview struct {
	ConfirmationToken string    `json:"confirmation_token"`
	Name              string    `json:"name"`
	Kind              string    `json:"kind"`
	SizeBytes         int64     `json:"size_bytes"`
	Files             int64     `json:"files"`
	Directories       int64     `json:"directories"`
	ModifiedAt        time.Time `json:"modified_at"`
	ExpiresAt         time.Time `json:"expires_at"`
	RequiresTypedName bool      `json:"requires_typed_name"`
}

type StorageDeleteResult struct {
	Deleted  bool   `json:"deleted"`
	Recycled bool   `json:"recycled"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
}

type storageScanTarget struct {
	path        string
	name        string
	kind        string
	size        int64
	files       int64
	directories int64
	identity    os.FileInfo
}

type storageDeleteAuthorization struct {
	target    storageScanTarget
	identity  os.FileInfo
	expiresAt time.Time
}

var storageTrashTarget = moveStorageTargetToTrash

func (s *Service) PreviewStorageDelete(scanID, nodeID string) (StorageDeletePreview, error) {
	job, err := s.storageScanByID(scanID)
	if err != nil {
		return StorageDeletePreview{}, err
	}
	job.mu.Lock()
	target, state := job.deleteTargets[nodeID], job.state
	root := job.root
	job.mu.Unlock()
	if state != "complete" || target.path == "" {
		return StorageDeletePreview{}, errors.New("storage target is unavailable for deletion")
	}
	identity, err := validateStorageDeleteTarget(root, target, target.identity)
	if err != nil {
		return StorageDeletePreview{}, err
	}
	token, err := newStorageScanID()
	if err != nil {
		return StorageDeletePreview{}, err
	}
	now := time.Now().UTC()
	expiresAt := now.Add(storageDeleteAuthorizationTTL)
	job.mu.Lock()
	for key, authorization := range job.deleteTokens {
		if !authorization.expiresAt.After(now) {
			delete(job.deleteTokens, key)
		}
	}
	if len(job.deleteTokens) >= storageDeleteMaxPending {
		job.mu.Unlock()
		return StorageDeletePreview{}, errors.New("too many pending storage deletion confirmations")
	}
	job.deleteTokens[token] = storageDeleteAuthorization{target: target, identity: identity, expiresAt: expiresAt}
	job.mu.Unlock()
	size := target.size
	if target.kind == "file" {
		size = max(identity.Size(), 0)
	}
	return StorageDeletePreview{
		ConfirmationToken: token,
		Name:              target.name, Kind: target.kind, SizeBytes: size, Files: target.files, Directories: target.directories,
		ModifiedAt: identity.ModTime().UTC(), ExpiresAt: expiresAt,
		RequiresTypedName: target.kind == "directory" && (size >= storageDeleteTypedBytes || target.files+target.directories >= storageDeleteTypedEntries),
	}, nil
}

func (s *Service) ConfirmStorageDelete(scanID, confirmationToken string) (StorageDeleteResult, error) {
	job, err := s.storageScanByID(scanID)
	if err != nil {
		return StorageDeleteResult{}, err
	}
	if len(confirmationToken) != 32 {
		return StorageDeleteResult{}, errors.New("invalid storage deletion confirmation")
	}
	job.mu.Lock()
	authorization, ok := job.deleteTokens[confirmationToken]
	delete(job.deleteTokens, confirmationToken)
	root, state := job.root, job.state
	job.mu.Unlock()
	if !ok || state != "complete" || !authorization.expiresAt.After(time.Now().UTC()) {
		return StorageDeleteResult{}, errors.New("storage deletion confirmation expired")
	}
	if _, err := validateStorageDeleteTarget(root, authorization.target, authorization.identity); err != nil {
		return StorageDeleteResult{}, err
	}
	if err := storageTrashTarget(authorization.target.path); err != nil {
		return StorageDeleteResult{}, fmt.Errorf("move storage target to Recycle Bin: %w", err)
	}
	if _, err := os.Lstat(authorization.target.path); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return StorageDeleteResult{}, errors.New("storage target still exists after Recycle Bin operation")
		}
		return StorageDeleteResult{}, fmt.Errorf("verify recycled storage target: %w", err)
	}
	s.invalidateStorageScanCache(authorization.target.path)
	return StorageDeleteResult{Deleted: true, Recycled: true, Name: authorization.target.name, Kind: authorization.target.kind}, nil
}

func validateStorageDeleteTarget(root string, target storageScanTarget, expected os.FileInfo) (os.FileInfo, error) {
	if expected == nil || (target.kind != "file" && target.kind != "directory") || !storagePathDescendant(root, target.path) {
		return nil, errors.New("storage deletion target is invalid")
	}
	info, err := os.Lstat(target.path)
	if err != nil {
		return nil, fmt.Errorf("storage deletion target: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || storageEntryIsReparse(info) || target.kind == "file" && !info.Mode().IsRegular() || target.kind == "directory" && !info.IsDir() {
		return nil, errors.New("storage deletion target type changed after the scan")
	}
	if storageDeleteTargetProtected(root, target.path, info) {
		return nil, errors.New("system-managed storage targets are protected")
	}
	resolved, err := filepath.EvalSymlinks(target.path)
	if err != nil || !sameStoragePath(filepath.Clean(resolved), filepath.Clean(target.path)) {
		return nil, errors.New("storage deletion target became a link or reparse point")
	}
	if !os.SameFile(expected, info) || expected.Mode() != info.Mode() || expected.Size() != info.Size() || !expected.ModTime().Equal(info.ModTime()) {
		return nil, errors.New("storage deletion target changed after confirmation")
	}
	return info, nil
}

func storagePathDescendant(root, target string) bool {
	root, target = filepath.Clean(root), filepath.Clean(target)
	if sameStoragePath(root, target) {
		return false
	}
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func storagePathWithin(root, target string) bool {
	return sameStoragePath(root, target) || storagePathDescendant(root, target)
}
