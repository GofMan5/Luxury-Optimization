package optimizer

import (
	"errors"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

type linuxMount struct {
	path       string
	source     string
	fileSystem string
	readOnly   bool
}

func listStorageVolumes() (StorageVolumesReport, error) {
	data, err := readSmallFile("/proc/self/mountinfo", 4<<20)
	if err != nil {
		return StorageVolumesReport{}, err
	}
	mounts := parseMountInfo(string(data))
	report := StorageVolumesReport{Volumes: make([]StorageVolume, 0, len(mounts))}
	seen := make(map[string]bool)
	for _, mount := range mounts {
		if seen[mount.path] || pseudoFileSystem(mount.fileSystem) {
			report.Skipped++
			continue
		}
		seen[mount.path] = true
		volume, err := linuxStorageVolume(mount)
		if err != nil {
			report.Skipped++
			if len(report.Warnings) < 16 {
				report.Warnings = append(report.Warnings, mount.path+": "+displayText(err.Error()))
			}
			continue
		}
		report.Volumes = append(report.Volumes, volume)
		if len(report.Volumes) == 128 {
			break
		}
	}
	sort.Slice(report.Volumes, func(i, j int) bool { return report.Volumes[i].Path < report.Volumes[j].Path })
	return report, nil
}

func storageVolumeForPath(path string) (StorageVolume, error) {
	report, err := listStorageVolumes()
	if err != nil {
		return StorageVolume{}, err
	}
	path = filepath.Clean(path)
	best := -1
	for index, volume := range report.Volumes {
		root := filepath.Clean(volume.Path)
		if path == root || root == "/" || strings.HasPrefix(path, root+string(filepath.Separator)) {
			if best == -1 || len(root) > len(report.Volumes[best].Path) {
				best = index
			}
		}
	}
	if best == -1 {
		return StorageVolume{}, errors.New("linux mount point was not found")
	}
	return report.Volumes[best], nil
}

func parseMountInfo(value string) []linuxMount {
	result := make([]linuxMount, 0, 32)
	for _, line := range strings.Split(value, "\n") {
		fields := strings.Fields(line)
		separator := -1
		for index, field := range fields {
			if field == "-" {
				separator = index
				break
			}
		}
		if separator < 6 || separator+2 >= len(fields) {
			continue
		}
		result = append(result, linuxMount{
			path: decodeMountField(fields[4]), source: decodeMountField(fields[separator+2]), fileSystem: fields[separator+1],
			readOnly: containsMountOption(fields[5], "ro"),
		})
	}
	return result
}

func decodeMountField(value string) string {
	return strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`).Replace(value)
}

func containsMountOption(options, expected string) bool {
	for _, option := range strings.Split(options, ",") {
		if option == expected {
			return true
		}
	}
	return false
}

func linuxStorageVolume(mount linuxMount) (StorageVolume, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(mount.path, &stat); err != nil {
		return StorageVolume{}, err
	}
	blockSize := uint64(stat.Bsize)
	return StorageVolume{
		Path: mount.path, Name: mount.source, FileSystem: mount.fileSystem, Kind: linuxStorageKind(mount),
		TotalBytes: boundedMultiply(uint64(stat.Blocks), blockSize), AvailableBytes: boundedMultiply(uint64(stat.Bavail), blockSize), ReadOnly: mount.readOnly,
	}, nil
}

func boundedMultiply(left, right uint64) uint64 {
	if right != 0 && left > math.MaxUint64/right {
		return math.MaxUint64
	}
	return left * right
}

func linuxStorageKind(mount linuxMount) string {
	switch {
	case strings.HasPrefix(mount.source, "/dev/"):
		return "device"
	case mount.fileSystem == "overlay":
		return "overlay"
	case strings.HasPrefix(mount.fileSystem, "fuse"):
		return "fuse"
	default:
		return "filesystem"
	}
}

func pseudoFileSystem(value string) bool {
	switch value {
	case "proc", "sysfs", "devtmpfs", "devpts", "tmpfs", "ramfs", "cgroup", "cgroup2", "pstore", "securityfs", "debugfs", "tracefs", "configfs", "mqueue", "hugetlbfs", "fusectl", "rpc_pipefs", "binfmt_misc", "autofs":
		return true
	default:
		return false
	}
}
