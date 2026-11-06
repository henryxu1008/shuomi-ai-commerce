package currentapplication

import (
	"io/fs"
	"os"
	"path"
	"strings"

	"golang.org/x/sys/unix"
	"task-processor/internal/product/dataacquisition"
)

type trialInputDir struct{ *os.File }

func openSHEINUSTrialDirectory(dir string) (fs.FS, func() error, error) {
	if !path.IsAbs(dir) || dir == "/" || path.Clean(dir) != dir {
		return nil, nil, dataacquisition.ErrUnavailable
	}
	const flags = unix.O_RDONLY | unix.O_DIRECTORY | unix.O_NOFOLLOW | unix.O_CLOEXEC
	fd, err := unix.Open("/", flags, 0)
	if err != nil {
		return nil, nil, dataacquisition.ErrUnavailable
	}
	var info unix.Stat_t
	if unix.Fstat(fd, &info) != nil || info.Uid != 0 || info.Mode&0022 != 0 {
		unix.Close(fd)
		return nil, nil, dataacquisition.ErrUnavailable
	}
	for _, component := range strings.Split(strings.TrimPrefix(dir, "/"), "/") {
		next, err := unix.Openat(fd, component, flags, 0)
		unix.Close(fd)
		if err != nil {
			return nil, nil, dataacquisition.ErrUnavailable
		}
		fd = next
		if unix.Fstat(fd, &info) != nil || info.Uid != 0 || info.Mode&0022 != 0 {
			unix.Close(fd)
			return nil, nil, dataacquisition.ErrUnavailable
		}
	}
	var mount unix.Statfs_t
	if info.Mode&07777 != 0700 || unix.Fstatfs(fd, &mount) != nil || mount.Flags&unix.ST_RDONLY == 0 {
		unix.Close(fd)
		return nil, nil, dataacquisition.ErrUnavailable
	}
	root := trialInputDir{os.NewFile(uintptr(fd), "shein-us-trial-inputs")}
	return root, root.Close, nil
}

func (d trialInputDir) Open(name string) (fs.File, error) {
	limit := int64(trialControlFileLimit)
	switch name {
	case "approval.json", "gate.json":
	case "key.json":
		limit = trialKeyFileLimit
	case "usage.json":
		limit = trialUsageFileLimit
	default:
		return nil, dataacquisition.ErrUnavailable
	}
	root, err := d.SyscallConn()
	if err != nil {
		return nil, dataacquisition.ErrUnavailable
	}
	var fd int
	var openErr error
	err = root.Control(func(rootFD uintptr) {
		fd, openErr = unix.Openat(int(rootFD), name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	})
	if err != nil || openErr != nil {
		return nil, dataacquisition.ErrUnavailable
	}
	var info unix.Stat_t
	if unix.Fstat(fd, &info) != nil || info.Mode != unix.S_IFREG|0600 || info.Uid != 0 || info.Nlink != 1 || info.Size > limit {
		unix.Close(fd)
		return nil, dataacquisition.ErrUnavailable
	}
	return os.NewFile(uintptr(fd), name), nil
}
