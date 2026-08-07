package optimizer

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"

	"golang.org/x/sys/windows"
)

func listStorageVolumes() (StorageVolumesReport, error) {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return StorageVolumesReport{}, err
	}
	report := StorageVolumesReport{Volumes: make([]StorageVolume, 0, 8)}
	for index := 0; index < 26; index++ {
		if mask&(1<<index) == 0 {
			continue
		}
		root := fmt.Sprintf("%c:\\", 'A'+index)
		rootPointer, _ := windows.UTF16PtrFromString(root)
		driveType := windows.GetDriveType(rootPointer)
		if driveType == windows.DRIVE_REMOTE || driveType == windows.DRIVE_CDROM || driveType == windows.DRIVE_NO_ROOT_DIR {
			report.Skipped++
			continue
		}
		volume, err := windowsStorageVolume(root)
		if err != nil {
			report.Skipped++
			if len(report.Warnings) < 16 {
				report.Warnings = append(report.Warnings, root+": "+displayText(err.Error()))
			}
			continue
		}
		report.Volumes = append(report.Volumes, volume)
	}
	sort.Slice(report.Volumes, func(i, j int) bool { return report.Volumes[i].Path < report.Volumes[j].Path })
	return report, nil
}

func storageVolumeForPath(path string) (StorageVolume, error) {
	pointer, err := windows.UTF16PtrFromString(filepath.Clean(path))
	if err != nil {
		return StorageVolume{}, err
	}
	const rootSize = 1024
	root := make([]uint16, rootSize)
	if err := windows.GetVolumePathName(pointer, &root[0], rootSize); err != nil {
		return StorageVolume{}, err
	}
	value := windows.UTF16ToString(root)
	if value == "" {
		return StorageVolume{}, errors.New("windows did not resolve the volume root")
	}
	return windowsStorageVolume(value)
}

func windowsStorageVolume(root string) (StorageVolume, error) {
	pointer, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return StorageVolume{}, err
	}
	var available, total, free uint64
	if err := windows.GetDiskFreeSpaceEx(pointer, &available, &total, &free); err != nil {
		return StorageVolume{}, err
	}
	const nameSize, fileSystemSize = 261, 64
	name, fileSystem := make([]uint16, nameSize), make([]uint16, fileSystemSize)
	var serial, maximumComponentLength, flags uint32
	if err := windows.GetVolumeInformation(pointer, &name[0], nameSize, &serial, &maximumComponentLength, &flags, &fileSystem[0], fileSystemSize); err != nil {
		return StorageVolume{}, err
	}
	return StorageVolume{
		Path: root, Name: windows.UTF16ToString(name), FileSystem: windows.UTF16ToString(fileSystem),
		Kind: windowsDriveKind(windows.GetDriveType(pointer)), TotalBytes: total, AvailableBytes: available,
		ReadOnly: flags&windows.FILE_READ_ONLY_VOLUME != 0,
	}, nil
}

func windowsDriveKind(value uint32) string {
	switch value {
	case windows.DRIVE_FIXED:
		return "fixed"
	case windows.DRIVE_REMOVABLE:
		return "removable"
	case windows.DRIVE_REMOTE:
		return "remote"
	case windows.DRIVE_RAMDISK:
		return "ramdisk"
	default:
		return "unknown"
	}
}
