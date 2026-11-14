//go:build !linux

package main

import "errors"

func consumeIsolatedCapability(string) ([]byte, error) {
	return nil, errors.New("isolated capability requires Linux")
}
