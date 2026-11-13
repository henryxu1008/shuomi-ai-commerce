//go:build !linux

package main

import "errors"

func dropToPostgres() error { return errors.New("isolated privilege drop requires Linux") }
