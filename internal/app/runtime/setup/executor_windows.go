//go:build windows

package setup

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"task-processor/internal/app/setupcontrol"
)

// Recovery supports the observed native Docker Desktop client chain, not an
// arbitrary shell launcher. A global conservative snapshot also sees orphans.
func (d *Driver) desktopClientsIdle(ctx context.Context) (bool, error) {
	exe, e := exec.LookPath("docker")
	if e != nil {
		return false, e
	}
	exe, e = filepath.EvalSymlinks(exe)
	if e != nil {
		return false, e
	}
	if !strings.HasSuffix(strings.ToLower(filepath.ToSlash(exe)), "/resources/bin/docker.exe") {
		return false, setupcontrol.ErrUnavailable
	}
	b, e := d.run(ctx, "info", "--format", "{{json .ClientInfo}}")
	if e != nil {
		return false, e
	}
	var info struct {
		Os      string
		Plugins []struct{ Name, Vendor, Path string }
	}
	if json.Unmarshal(b, &info) != nil || info.Os != "windows" {
		return false, setupcontrol.ErrUnavailable
	}
	compose := false
	for _, p := range info.Plugins {
		if p.Name != "compose" && p.Name != "buildx" {
			continue
		}
		resolved, e := filepath.EvalSymlinks(p.Path)
		if e != nil || p.Vendor != "Docker Inc." || !strings.EqualFold(filepath.Base(resolved), "docker-"+p.Name+".exe") {
			return false, setupcontrol.ErrUnavailable
		}
		compose = compose || p.Name == "compose"
	}
	if !compose {
		return false, setupcontrol.ErrUnavailable
	}
	return windowsClientsIdle()
}
func windowsClientsIdle() (bool, error) {
	snapshot, e := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if e != nil {
		return false, e
	}
	defer windows.CloseHandle(snapshot)
	return scanWindowsClients(func(p *windows.ProcessEntry32) error { return windows.Process32First(snapshot, p) }, func(p *windows.ProcessEntry32) error { return windows.Process32Next(snapshot, p) })
}
func scanWindowsClients(first, next func(*windows.ProcessEntry32) error) (bool, error) {
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	e := first(&entry)
	idle := true
	for e == nil {
		switch strings.ToLower(windows.UTF16ToString(entry.ExeFile[:])) {
		case "docker.exe", "docker-compose.exe", "docker-buildx.exe", "com.docker.cli.exe":
			idle = false
		}
		e = next(&entry)
	}
	if !errors.Is(e, windows.ERROR_NO_MORE_FILES) {
		return false, e
	}
	return idle, nil
}
