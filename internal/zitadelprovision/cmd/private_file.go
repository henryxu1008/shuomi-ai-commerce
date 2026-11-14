package main

import "task-processor/internal/platform/privatefile"

func writePrivateFile(path string, data []byte) error { return privatefile.Write(path, data) }
func writePrivateFileWithReplace(path string, data []byte, replace func(string, string) error) error {
	return privatefile.WriteWithReplace(path, data, replace)
}
