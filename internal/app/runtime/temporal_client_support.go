package runtime

import (
	"errors"
	"os"
	"strings"

	sdkclient "go.temporal.io/sdk/client"
)

var errTemporalCloseOwnerRequired = errors.New("temporal dial succeeded without close owner")

func validateTemporalDialResult(client sdkclient.Client, closeFn func() error, err error) (sdkclient.Client, func() error, error) {
	if err != nil {
		return nil, nil, err
	}
	if closeFn == nil {
		return nil, nil, errTemporalCloseOwnerRequired
	}
	return client, closeFn, nil
}

func envBool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "y", "on", "enabled":
		return true
	default:
		return false
	}
}
