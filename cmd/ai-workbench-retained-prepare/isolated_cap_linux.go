//go:build linux

package main

import (
	"errors"
	"io"
	"os"
	"regexp"
	"syscall"
)

func consumeIsolatedCapability(path string) ([]byte, error) {
	if os.Geteuid() != 0 || !regexp.MustCompile(`^/tmp/ai-workbench-cap-[a-f0-9]{32}$`).MatchString(path) {
		return nil, errors.New("isolated capability unavailable")
	}
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, errors.New("isolated capability unavailable")
	}
	f := os.NewFile(uintptr(fd), "capability")
	defer f.Close()
	var st syscall.Stat_t
	if err := syscall.Fstat(fd, &st); err != nil || st.Uid != 0 || st.Mode&syscall.S_IFMT != syscall.S_IFREG || st.Mode&0777 != 0600 || st.Nlink != 1 || st.Size <= 0 || st.Size > 65536 {
		return nil, errors.New("isolated capability invalid")
	}
	b, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil || len(b) == 0 || len(b) > 65536 {
		return nil, errors.New("isolated capability invalid")
	}
	if err := os.Remove(path); err != nil {
		return nil, errors.New("isolated capability cannot be consumed")
	}
	return b, nil
}
