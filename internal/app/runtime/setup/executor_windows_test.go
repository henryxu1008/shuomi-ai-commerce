//go:build windows

package setup

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

func TestRecoveryRejectsClientSurvivingControllerAndLauncher(t *testing.T) {
	root := t.TempDir()
	self, e := os.Executable()
	require.NoError(t, e)
	for _, name := range []string{"docker.exe", "docker-compose.exe"} {
		in, e := os.Open(self)
		require.NoError(t, e)
		out, e := os.Create(filepath.Join(root, name))
		require.NoError(t, e)
		_, e = io.Copy(out, in)
		require.NoError(t, e)
		require.NoError(t, out.Close())
		require.NoError(t, in.Close())
	}
	cmd := exec.Command(self, "-test.run=^TestRecoveryNativeProcessHelper$")
	cmd.Env = append(os.Environ(), "SETUP_NATIVE_FIXTURE="+root, "SETUP_NATIVE_ROLE=controller")
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = os.WriteFile(filepath.Join(root, "stop"), []byte("stop"), 0600); _ = cmd.Process.Kill() })
	openChild := func(role string) windows.Handle {
		var pid int
		require.Eventually(t, func() bool {
			b, e := os.ReadFile(filepath.Join(root, role+".pid"))
			if e != nil {
				return false
			}
			pid, e = strconv.Atoi(string(b))
			return e == nil && pid > 0
		}, 5*time.Second, 10*time.Millisecond)
		h, e := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, uint32(pid))
		require.NoError(t, e)
		buf := make([]uint16, 32768)
		size := uint32(len(buf))
		require.NoError(t, windows.QueryFullProcessImageName(h, 0, &buf[0], &size))
		want := "docker.exe"
		if role == "plugin" {
			want = "docker-compose.exe"
		}
		require.Equal(t, filepath.Join(root, want), windows.UTF16ToString(buf[:size]))
		t.Cleanup(func() { _, _ = windows.WaitForSingleObject(h, 5000); _ = windows.CloseHandle(h) })
		return h
	}
	cli, plugin := openChild("cli"), openChild("plugin")
	require.NoError(t, cmd.Process.Kill())
	_ = cmd.Wait()
	idle, e := windowsClientsIdle()
	require.NoError(t, e)
	require.False(t, idle, "orphan client must remain visible without its controller")
	require.NoError(t, windows.TerminateProcess(cli, 1))
	_, e = windows.WaitForSingleObject(cli, 5000)
	require.NoError(t, e)
	r := &auditRunner{}
	d, p := auditFixture(t, r)
	d.clientsIdle = func(context.Context) (bool, error) { return windowsClientsIdle() }
	a, e := d.AuditRecovery(context.Background(), p, true)
	require.NoError(t, e)
	require.False(t, a.Empty)
	require.NotEmpty(t, a.Blocks)
	require.NoError(t, os.WriteFile(filepath.Join(root, "late-action"), []byte("go"), 0600))
	require.Eventually(t, func() bool { _, e := os.Stat(filepath.Join(root, "plugin-continued")); return e == nil }, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, os.WriteFile(filepath.Join(root, "stop"), []byte("stop"), 0600))
	_, e = windows.WaitForSingleObject(plugin, 5000)
	require.NoError(t, e)
}
func TestRecoveryNativeProcessHelper(t *testing.T) {
	root, role := os.Getenv("SETUP_NATIVE_FIXTURE"), os.Getenv("SETUP_NATIVE_ROLE")
	if root == "" {
		t.Skip("isolated process helper")
	}
	if role == "controller" || role == "cli" {
		next, name := "cli", "docker.exe"
		if role == "cli" {
			next, name = "plugin", "docker-compose.exe"
		}
		cmd := exec.Command(filepath.Join(root, name), "-test.run=^TestRecoveryNativeProcessHelper$")
		cmd.Env = append(os.Environ(), "SETUP_NATIVE_ROLE="+next)
		cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
		require.NoError(t, cmd.Start())
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, role+".pid"), []byte(strconv.Itoa(os.Getpid())), 0600))
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if _, e := os.Stat(filepath.Join(root, "stop")); e == nil {
			return
		}
		if role == "plugin" {
			if _, e := os.Stat(filepath.Join(root, "late-action")); e == nil {
				_ = os.WriteFile(filepath.Join(root, "plugin-continued"), []byte("continued"), 0600)
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestClientSnapshotEnumerationErrorIsNotIdle(t *testing.T) {
	for _, tc := range []struct {
		name string
		next error
		idle bool
	}{{"complete clean", windows.ERROR_NO_MORE_FILES, true}, {"middle error", windows.ERROR_ACCESS_DENIED, false}} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			idle, e := scanWindowsClients(func(*windows.ProcessEntry32) error { return nil }, func(*windows.ProcessEntry32) error { calls++; return tc.next })
			require.Equal(t, 1, calls)
			require.Equal(t, tc.idle, idle)
			if tc.idle {
				require.NoError(t, e)
			} else {
				require.Error(t, e)
			}
		})
	}
}
