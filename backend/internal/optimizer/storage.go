package optimizer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const storageSafetyHeadroom = uint64(256 << 20)

type StorageVolume struct {
	Path           string `json:"path"`
	Name           string `json:"name,omitempty"`
	FileSystem     string `json:"file_system"`
	Kind           string `json:"kind"`
	TotalBytes     uint64 `json:"total_bytes"`
	AvailableBytes uint64 `json:"available_bytes"`
	ReadOnly       bool   `json:"read_only"`
}

type StorageVolumesReport struct {
	Volumes  []StorageVolume `json:"volumes"`
	Skipped  int             `json:"skipped"`
	Warnings []string        `json:"warnings,omitempty"`
}

type StoragePathReport struct {
	Path                 string        `json:"path"`
	Volume               StorageVolume `json:"volume"`
	SizeBytes            int64         `json:"size_bytes"`
	BlockBytes           int           `json:"block_bytes"`
	BufferedWriteMBs     float64       `json:"buffered_write_mb_s"`
	DurableWriteMBs      float64       `json:"durable_write_mb_s"`
	SyncMS               float64       `json:"sync_ms"`
	BufferedReadMBs      float64       `json:"buffered_read_mb_s"`
	SHA256               string        `json:"sha256"`
	Verified             bool          `json:"verified"`
	TemporaryFileRemoved bool          `json:"temporary_file_removed"`
}

func storageCommand(args []string) error {
	if len(args) == 0 || args[0] == "volumes" {
		if len(args) > 0 {
			args = args[1:]
		}
		set := flag.NewFlagSet("storage volumes", flag.ContinueOnError)
		jsonOnly := set.Bool("json", false, "вывести JSON")
		if err := set.Parse(args); err != nil {
			return err
		}
		if set.NArg() != 0 {
			return errors.New("лишние аргументы storage volumes")
		}
		report, err := listStorageVolumes()
		if err != nil {
			return err
		}
		if *jsonOnly {
			data, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}
		for _, volume := range report.Volumes {
			fmt.Printf("%s  %s  %.1f/%.1f GiB free\n", displayText(volume.Path), displayText(volume.FileSystem), float64(volume.AvailableBytes)/(1<<30), float64(volume.TotalBytes)/(1<<30))
		}
		return nil
	}
	if args[0] != "test" {
		return errors.New("storage поддерживает volumes и test")
	}
	set := flag.NewFlagSet("storage test", flag.ContinueOnError)
	path := set.String("path", os.TempDir(), "существующий каталог")
	sizeMB := set.Int("size-mb", 64, "размер временного файла, 8-256 MiB")
	blockKB := set.Int("block-kb", 1024, "размер блока, 64-4096 KiB")
	jsonOnly := set.Bool("json", false, "вывести JSON")
	if err := set.Parse(args[1:]); err != nil {
		return err
	}
	if set.NArg() != 0 {
		return errors.New("лишние аргументы storage test")
	}
	report, err := measureStoragePath(context.Background(), *path, *sizeMB, *blockKB)
	if err != nil {
		return err
	}
	if *jsonOnly {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("Storage %s: durable write %.1f MB/s, buffered read %.1f MB/s, sync %.1f ms, verified=%t\n", displayText(report.Path), report.DurableWriteMBs, report.BufferedReadMBs, report.SyncMS, report.Verified)
	return nil
}

func measureStoragePath(ctx context.Context, rawPath string, sizeMB, blockKB int) (report StoragePathReport, err error) {
	if sizeMB < 8 || sizeMB > 256 || blockKB < 64 || blockKB > 4096 || blockKB&(blockKB-1) != 0 {
		return report, errors.New("size_mb must be 8-256 and block_kb must be a power of two from 64 to 4096")
	}
	path, err := filepath.Abs(rawPath)
	if err != nil {
		return report, err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return report, fmt.Errorf("storage path: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return report, errors.New("storage path must be an existing directory")
	}
	volume, err := storageVolumeForPath(path)
	if err != nil {
		return report, fmt.Errorf("storage capability: %w", err)
	}
	if volume.ReadOnly {
		return report, errors.New("storage path is read-only")
	}
	if volume.Kind == "remote" {
		return report, errors.New("remote storage paths are excluded from the local gaming storage probe")
	}
	sizeBytes := int64(sizeMB) << 20
	blockBytes := blockKB << 10
	requiredBytes := uint64(sizeBytes) + storageSafetyHeadroom // #nosec G115 -- sizeBytes is validated to 8-256 MiB above.
	if volume.AvailableBytes < requiredBytes {
		return report, errors.New("storage probe requires its file size plus 256 MiB of free-space headroom")
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}

	file, err := os.CreateTemp(path, ".luxury-storage-probe-*")
	if err != nil {
		return report, fmt.Errorf("create storage probe: %w", err)
	}
	temporaryPath := file.Name()
	removed := false
	defer func() {
		_ = file.Close()
		if !removed {
			removed = os.Remove(temporaryPath) == nil
		}
		report.TemporaryFileRemoved = removed
	}()

	block := storagePattern(blockBytes)
	writtenHash := sha256.New()
	writeStarted := time.Now()
	for remaining := sizeBytes; remaining > 0; {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		chunk := int64(len(block))
		if remaining < chunk {
			chunk = remaining
		}
		if _, err := file.Write(block[:chunk]); err != nil {
			return report, fmt.Errorf("write storage probe: %w", err)
		}
		_, _ = writtenHash.Write(block[:chunk])
		remaining -= chunk
	}
	writeElapsed := time.Since(writeStarted)
	syncStarted := time.Now()
	if err := file.Sync(); err != nil {
		return report, fmt.Errorf("sync storage probe: %w", err)
	}
	syncElapsed := time.Since(syncStarted)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return report, fmt.Errorf("seek storage probe: %w", err)
	}

	readHash := sha256.New()
	readBuffer := make([]byte, blockBytes)
	readStarted := time.Now()
	readBytes := int64(0)
	for {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		count, readErr := file.Read(readBuffer)
		if count > 0 {
			_, _ = readHash.Write(readBuffer[:count])
			readBytes += int64(count)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return report, fmt.Errorf("read storage probe: %w", readErr)
		}
	}
	readElapsed := time.Since(readStarted)
	if err := file.Close(); err != nil {
		return report, fmt.Errorf("close storage probe: %w", err)
	}
	if err := os.Remove(temporaryPath); err != nil {
		return report, fmt.Errorf("remove storage probe: %w", err)
	}
	removed = true

	writtenDigest, readDigest := writtenHash.Sum(nil), readHash.Sum(nil)
	report = StoragePathReport{
		Path: path, Volume: volume, SizeBytes: sizeBytes, BlockBytes: blockBytes,
		BufferedWriteMBs: bytesPerSecond(sizeBytes, writeElapsed) / 1_000_000,
		DurableWriteMBs:  bytesPerSecond(sizeBytes, writeElapsed+syncElapsed) / 1_000_000,
		SyncMS:           milliseconds(syncElapsed), BufferedReadMBs: bytesPerSecond(readBytes, readElapsed) / 1_000_000,
		SHA256: hex.EncodeToString(readDigest), Verified: readBytes == sizeBytes && string(writtenDigest) == string(readDigest), TemporaryFileRemoved: true,
	}
	if !report.Verified {
		return report, errors.New("storage probe read-back verification failed")
	}
	return report, nil
}

func storagePattern(size int) []byte {
	result := make([]byte, size)
	state := uint64(0x9e3779b97f4a7c15)
	for index := range result {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		result[index] = byte(state)
	}
	return result
}

func bytesPerSecond(bytes int64, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return float64(bytes) / elapsed.Seconds()
}
