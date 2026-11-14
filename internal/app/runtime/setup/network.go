package setup

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net/netip"
	"sort"
	"strings"
	"task-processor/internal/app/setupcontrol"
)

func validateSubnet(value string) error {
	if value == "" {
		return nil
	}
	p, e := netip.ParsePrefix(value)
	if e != nil || !p.Addr().Is4() || !p.Addr().IsPrivate() || p.Bits() < 16 || p.Bits() > 29 || p.Masked().String() != value {
		return setupcontrol.ErrInvalid
	}
	return nil
}
func configureNetworks(effective map[string]any, value string) error {
	if value == "" {
		return nil
	}
	if e := validateSubnet(value); e != nil {
		return e
	}
	networks, ok := effective["networks"].(map[string]any)
	if !ok || len(networks) < 1 {
		return setupcontrol.ErrInvalid
	}
	names := []string{}
	for name, raw := range networks {
		v, ok := raw.(map[string]any)
		if !ok || v["external"] == true {
			return setupcontrol.ErrInvalid
		}
		names = append(names, name)
	}
	sort.Strings(names)
	p, _ := netip.ParsePrefix(value)
	bits := p.Bits()
	for capacity := 1; capacity < len(names); capacity *= 2 {
		bits++
	}
	if bits > 29 {
		return setupcontrol.ErrInvalid
	}
	address := p.Addr().As4()
	start := binary.BigEndian.Uint32(address[:])
	stride := uint32(1) << uint(32-bits)
	for i, name := range names {
		var addr [4]byte
		binary.BigEndian.PutUint32(addr[:], start+uint32(i)*stride)
		subnet := netip.PrefixFrom(netip.AddrFrom4(addr), bits).String()
		networks[name].(map[string]any)["ipam"] = map[string]any{"config": []any{map[string]any{"subnet": subnet}}}
	}
	return nil
}
func (d *Driver) checkSubnet(ctx context.Context, value string) error {
	if e := validateSubnet(value); e != nil {
		return e
	}
	if value == "" {
		return nil
	}
	candidate, _ := netip.ParsePrefix(value)
	raw, e := d.run(ctx, "network", "ls", "-q")
	if e != nil {
		return e
	}
	ids := strings.Fields(string(raw))
	if len(ids) == 0 {
		return nil
	}
	if len(ids) > 512 {
		return setupcontrol.ErrUnavailable
	}
	raw, e = d.run(ctx, append([]string{"network", "inspect"}, ids...)...)
	if e != nil {
		return e
	}
	var networks []struct {
		IPAM struct{ Config []struct{ Subnet string } }
	}
	if json.Unmarshal(raw, &networks) != nil {
		return setupcontrol.ErrUnavailable
	}
	for _, n := range networks {
		for _, v := range n.IPAM.Config {
			existing, e := netip.ParsePrefix(v.Subnet)
			if e == nil && existing.Overlaps(candidate) {
				return setupcontrol.ErrInvalid
			}
		}
	}
	return nil
}
