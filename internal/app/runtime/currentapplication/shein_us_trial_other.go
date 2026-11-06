//go:build !linux

package currentapplication

import (
	"errors"
	"io/fs"
)

func openSHEINUSTrialDirectory(string) (fs.FS, func() error, error) {
	return nil, nil, errors.New("SHEIN US trial requires a private Linux read-only mount")
}
