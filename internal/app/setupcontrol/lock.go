package setupcontrol

import (
	"os"
	"path/filepath"
	"task-processor/internal/platform/privatefile"
)

type instanceLock struct{ file *os.File }

func lockInstance(dir string) (*instanceLock, error) {
	if privatefile.RejectPath(dir) != nil {
		return nil, ErrUnavailable
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	p := filepath.Join(dir, "instance.lock")
	if privatefile.RejectPath(p) != nil {
		return nil, ErrUnavailable
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = lockFile(f); err != nil {
		f.Close()
		return nil, ErrBusy
	}
	return &instanceLock{f}, nil
}
func (l *instanceLock) Close() error { _ = unlockFile(l.file); return l.file.Close() }
