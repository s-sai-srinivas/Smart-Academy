//go:build windows

package services

import (
	"golang.org/x/sys/windows"
)

// getFreeDiskSpacePlatform returns free disk space in bytes for Windows systems
func getFreeDiskSpacePlatform(path string) (uint64, error) {
	var freeBytesAvailableToCaller, totalNumberOfBytes, totalNumberOfFreeBytes uint64

	// Convert path to UTF16 pointer for Windows API
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}

	err = windows.GetDiskFreeSpaceEx(pathPtr, &freeBytesAvailableToCaller, &totalNumberOfBytes, &totalNumberOfFreeBytes)
	if err != nil {
		return 0, err
	}
	return freeBytesAvailableToCaller, nil
}
