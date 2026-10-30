package httpapi

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func membershipSMTP(t *testing.T, messages ...chan<- string) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	number, _ := strconv.Atoi(port)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
				fmt.Fprint(conn, "220 local ESMTP\r\n")
				reader := bufio.NewReader(conn)
				var body strings.Builder
				data := false
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					if data {
						if line == ".\r\n" {
							if len(messages) > 0 {
								messages[0] <- body.String()
							}
							data = false
							fmt.Fprint(conn, "250 queued\r\n")
						} else {
							body.WriteString(line)
						}
						continue
					}
					switch {
					case strings.HasPrefix(line, "EHLO"):
						fmt.Fprint(conn, "250 local\r\n")
					case strings.HasPrefix(line, "DATA"):
						data = true
						fmt.Fprint(conn, "354 go\r\n")
					case strings.HasPrefix(line, "QUIT"):
						fmt.Fprint(conn, "221 bye\r\n")
						return
					default:
						fmt.Fprint(conn, "250 OK\r\n")
					}
				}
			}()
		}
	}()
	return number
}
