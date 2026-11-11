//go:build !windows

package setup

import "golang.org/x/sys/unix"

func availableSpace(path string) *uint64 {
	var stat unix.Statfs_t
	if unix.Statfs(path, &stat) != nil {
		return nil
	}
	available := uint64(stat.Bavail) * uint64(stat.Bsize)
	return &available
}
