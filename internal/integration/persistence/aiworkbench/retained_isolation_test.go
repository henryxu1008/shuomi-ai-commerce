package aiworkbenchpersistence

import (
	"net"
	"strings"
	"testing"
)

func TestRetainedIsolationRequiresExpectedContainerAndLoopbackOnly(t *testing.T) {
	id := strings.Repeat("a", 64)
	lo := []net.Interface{{Name: "lo", Flags: net.FlagUp | net.FlagLoopback}}
	if err := verifyRetainedIsolation(id, id[:12], lo); err != nil {
		t.Fatal(err)
	}
	if err := verifyRetainedIsolation(id, "different", lo); err == nil {
		t.Fatal("different container accepted")
	}
	if err := verifyRetainedIsolation(id, id[:12], append(lo, net.Interface{Name: "eth0", Flags: net.FlagUp})); err == nil {
		t.Fatal("networked container accepted")
	}
}
