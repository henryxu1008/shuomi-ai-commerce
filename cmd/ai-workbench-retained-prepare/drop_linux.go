//go:build linux

package main

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

// Consume the root-only capability first, then drop all saved IDs and
// supplementary groups before opening a PostgreSQL connection.
func dropToPostgres() error {
	if !cgoDisabled || os.Geteuid() != 0 {
		return errors.New("isolated capability consumer is not root")
	}
	u, err := user.Lookup("postgres")
	if err != nil {
		return errors.New("postgres OS identity unavailable")
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil || uid <= 0 {
		return errors.New("invalid postgres UID")
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil || gid <= 0 {
		return errors.New("invalid postgres GID")
	}
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return errors.New("cannot set current-thread privilege restriction")
	}
	if err := syscall.Setgroups([]int{}); err != nil {
		return errors.New("cannot drop supplementary groups")
	}
	if err := syscall.Setresgid(gid, gid, gid); err != nil {
		return errors.New("cannot drop group identity")
	}
	if err := syscall.Setresuid(uid, uid, uid); err != nil {
		return errors.New("cannot drop user identity")
	}
	groups, err := os.Getgroups()
	if err != nil || len(groups) != 0 || os.Geteuid() != uid || os.Getegid() != gid {
		return errors.New("process privilege drop unverified")
	}
	ruid, euid, suid := unix.Getresuid()
	rgid, egid, sgid := unix.Getresgid()
	if ruid != uid || euid != uid || suid != uid || rgid != gid || egid != gid || sgid != gid {
		return errors.New("saved identities not dropped")
	}
	if err := syscall.Setresuid(0, 0, 0); err == nil || os.Geteuid() != uid {
		return errors.New("root identity remains recoverable")
	}
	paths, err := filepath.Glob("/proc/self/task/*/status")
	if err != nil || len(paths) == 0 {
		return errors.New("thread credentials unavailable")
	}
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			return errors.New("thread credentials unavailable")
		}
		var uidOK, gidOK, groupsOK, capsOK bool
		for _, line := range strings.Split(string(b), "\n") {
			f := strings.Fields(line)
			if len(f) == 0 {
				continue
			}
			switch f[0] {
			case "Uid:":
				uidOK = len(f) == 5 && f[1] == u.Uid && f[2] == u.Uid && f[3] == u.Uid && f[4] == u.Uid
			case "Gid:":
				gidOK = len(f) == 5 && f[1] == u.Gid && f[2] == u.Gid && f[3] == u.Gid && f[4] == u.Gid
			case "Groups:":
				groupsOK = len(f) == 1
			case "CapEff:":
				capsOK = len(f) == 2 && strings.Trim(f[1], "0") == ""
			}
		}
		if !uidOK || !gidOK || !groupsOK || !capsOK {
			return errors.New("thread privilege drop unverified")
		}
	}
	return nil
}
