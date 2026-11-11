//go:build windows

package setup

import "golang.org/x/sys/windows"

func availableSpace(path string) *uint64 {
	p, e := windows.UTF16PtrFromString(path)
	if e != nil {
		return nil
	}
	var available, total, free uint64
	if windows.GetDiskFreeSpaceEx(p, &available, &total, &free) != nil {
		return nil
	}
	return &available
}
