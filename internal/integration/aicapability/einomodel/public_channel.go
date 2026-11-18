package einomodel

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"time"

	"task-processor/internal/integration/httpimage"
)

var publicIPv6Space = netip.MustParsePrefix("2000::/3")
var transitionIPv6 = netip.MustParsePrefix("2002::/16")
var specialIPv6 = netip.MustParsePrefix("2001::/23")

func NewRegisteredChannelTransport() *http.Transport {
	transport := NewSingleSendHTTPTransport()
	transport.Proxy = nil
	transport.DialTLSContext = nil
	transport.DialTLS = nil
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport.DialContext = publicChannelDial(net.DefaultResolver.LookupNetIP, dialer.DialContext)
	return transport
}

func publicChannelDial(
	resolve func(context.Context, string, string) ([]netip.Addr, error),
	dial func(context.Context, string, string) (net.Conn, error),
) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		unavailable := errors.New("registered channel public address unavailable")
		host, port, err := net.SplitHostPort(address)
		if err != nil || port != "443" {
			return nil, unavailable
		}
		ips := []netip.Addr{}
		if literal, parseErr := netip.ParseAddr(host); parseErr == nil {
			ips = append(ips, literal)
		} else {
			ips, err = resolve(ctx, "ip", host)
		}
		if err != nil || len(ips) == 0 {
			return nil, unavailable
		}
		for _, ip := range ips {
			if !ip.IsGlobalUnicast() || ip.Zone() != "" || httpimage.IsPrivateIP(ip.AsSlice()) ||
				ip.Is6() && (!publicIPv6Space.Contains(ip) || transitionIPv6.Contains(ip) || specialIPv6.Contains(ip)) {
				return nil, unavailable
			}
		}
		// TLS still receives the requested URL host from the HTTP transport.
		return dial(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}
}
