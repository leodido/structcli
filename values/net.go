package values

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

func cloneIP(in net.IP) net.IP {
	if in == nil {
		return nil
	}

	return net.IP(cloneBytes([]byte(in)))
}

func cloneIPMask(in net.IPMask) net.IPMask {
	if in == nil {
		return nil
	}

	return net.IPMask(cloneBytes([]byte(in)))
}

func cloneIPNet(in net.IPNet) net.IPNet {
	return net.IPNet{
		IP:   cloneIP(in.IP),
		Mask: cloneIPMask(in.Mask),
	}
}

func cloneIPSlice(in []net.IP) []net.IP {
	if in == nil {
		return nil
	}

	out := make([]net.IP, len(in))
	for i := range in {
		out[i] = cloneIP(in[i])
	}

	return out
}

type ipValue struct {
	ip *net.IP
}

func NewIP(val net.IP, p *net.IP) *ipValue {
	*p = cloneIP(val)

	return &ipValue{ip: p}
}

func (i *ipValue) String() string {
	return (*i.ip).String()
}

func (i *ipValue) Set(s string) error {
	if s == "" {
		return nil
	}

	ip := net.ParseIP(strings.TrimSpace(s))
	if ip == nil {
		return fmt.Errorf("failed to parse IP: %q", s)
	}

	*i.ip = ip

	return nil
}

func (i *ipValue) Type() string {
	return "ip"
}

var _ pflag.Value = (*ipValue)(nil)

type ipMaskValue struct {
	mask *net.IPMask
}

func NewIPMask(val net.IPMask, p *net.IPMask) *ipMaskValue {
	*p = cloneIPMask(val)

	return &ipMaskValue{mask: p}
}

func (i *ipMaskValue) String() string {
	return net.IPMask(*i.mask).String()
}

func (i *ipMaskValue) Set(s string) error {
	mask := ParseIPv4Mask(strings.TrimSpace(s))
	if mask == nil {
		return fmt.Errorf("failed to parse IP mask: %q", s)
	}

	*i.mask = mask

	return nil
}

func (i *ipMaskValue) Type() string {
	return "ipMask"
}

var _ pflag.Value = (*ipMaskValue)(nil)

// ParseIPv4Mask parses masks written as dotted decimal or hex bytes (e.g. ffffff00).
func ParseIPv4Mask(s string) net.IPMask {
	mask := net.ParseIP(s)
	if mask == nil {
		if len(s) != 8 {
			return nil
		}

		m := []int{}
		for i := 0; i < 4; i++ {
			b := "0x" + s[2*i:2*i+2]
			d, err := strconv.ParseInt(b, 0, 0)
			if err != nil {
				return nil
			}
			m = append(m, int(d))
		}

		s = fmt.Sprintf("%d.%d.%d.%d", m[0], m[1], m[2], m[3])
		mask = net.ParseIP(s)
		if mask == nil {
			return nil
		}
	}

	return net.IPv4Mask(mask[12], mask[13], mask[14], mask[15])
}

type ipNetValue struct {
	ipnet *net.IPNet
}

func NewIPNet(val net.IPNet, p *net.IPNet) *ipNetValue {
	*p = cloneIPNet(val)

	return &ipNetValue{ipnet: p}
}

func (i *ipNetValue) String() string {
	return i.ipnet.String()
}

func (i *ipNetValue) Set(value string) error {
	_, n, err := net.ParseCIDR(strings.TrimSpace(value))
	if err != nil {
		return err
	}

	*i.ipnet = *n

	return nil
}

func (i *ipNetValue) Type() string {
	return "ipNet"
}

var _ pflag.Value = (*ipNetValue)(nil)

type ipSliceValue struct {
	value   *[]net.IP
	changed bool
}

func NewIPSlice(val []net.IP, p *[]net.IP) *ipSliceValue {
	*p = cloneIPSlice(val)

	return &ipSliceValue{value: p}
}

func readAsCSV(val string) ([]string, error) {
	if val == "" {
		return []string{}, nil
	}

	csvReader := csv.NewReader(strings.NewReader(val))

	return csvReader.Read()
}

func writeAsCSV(vals []string) (string, error) {
	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	if err := w.Write(vals); err != nil {
		return "", err
	}
	w.Flush()

	return strings.TrimSuffix(b.String(), "\n"), nil
}

func (s *ipSliceValue) Set(val string) error {
	rmQuote := strings.NewReplacer(`"`, "", `'`, "", "`", "")

	ipStrSlice, err := readAsCSV(rmQuote.Replace(val))
	if err != nil && err != io.EOF {
		return err
	}

	out := make([]net.IP, 0, len(ipStrSlice))
	for _, ipStr := range ipStrSlice {
		ip := net.ParseIP(strings.TrimSpace(ipStr))
		if ip == nil {
			return fmt.Errorf("invalid string being converted to IP address: %s", ipStr)
		}
		out = append(out, ip)
	}

	if !s.changed {
		*s.value = out
	} else {
		*s.value = append(*s.value, out...)
	}

	s.changed = true

	return nil
}

func (s *ipSliceValue) Type() string {
	return "ipSlice"
}

func (s *ipSliceValue) String() string {
	// Emit pflag-compatible bracketed CSV for help/default rendering and round-tripping.
	ipStrSlice := make([]string, len(*s.value))
	for i, ip := range *s.value {
		ipStrSlice[i] = ip.String()
	}

	out, _ := writeAsCSV(ipStrSlice)

	return "[" + out + "]"
}

func (s *ipSliceValue) Append(val string) error {
	ip := net.ParseIP(strings.TrimSpace(val))
	if ip == nil {
		return fmt.Errorf("invalid string being converted to IP address: %s", val)
	}
	*s.value = append(*s.value, ip)

	return nil
}

func (s *ipSliceValue) Replace(val []string) error {
	out := make([]net.IP, len(val))
	for i, raw := range val {
		ip := net.ParseIP(strings.TrimSpace(raw))
		if ip == nil {
			return fmt.Errorf("invalid string being converted to IP address: %s", raw)
		}
		out[i] = ip
	}
	*s.value = out

	return nil
}

func (s *ipSliceValue) GetSlice() []string {
	out := make([]string, len(*s.value))
	for i, ip := range *s.value {
		out[i] = ip.String()
	}

	return out
}

var _ pflag.Value = (*ipSliceValue)(nil)
var _ pflag.SliceValue = (*ipSliceValue)(nil)
