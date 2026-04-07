package values

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ipValue ---

func TestIPValue_SetAndString(t *testing.T) {
	var ip net.IP
	v := NewIP(net.ParseIP("127.0.0.1"), &ip)

	assert.Equal(t, "127.0.0.1", v.String())
	assert.Equal(t, "ip", v.Type())

	require.NoError(t, v.Set("10.0.0.1"))
	assert.Equal(t, "10.0.0.1", v.String())
	assert.True(t, net.ParseIP("10.0.0.1").Equal(ip))
}

func TestIPValue_SetEmpty(t *testing.T) {
	var ip net.IP
	v := NewIP(net.ParseIP("1.2.3.4"), &ip)
	require.NoError(t, v.Set(""))
	assert.Equal(t, "1.2.3.4", v.String(), "empty string should be a no-op")
}

func TestIPValue_SetInvalid(t *testing.T) {
	var ip net.IP
	v := NewIP(net.ParseIP("1.2.3.4"), &ip)
	err := v.Set("not-an-ip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse IP")
}

func TestIPValue_SetTrimsWhitespace(t *testing.T) {
	var ip net.IP
	v := NewIP(net.IPv4zero, &ip)
	require.NoError(t, v.Set("  192.168.1.1  "))
	assert.Equal(t, "192.168.1.1", v.String())
}

// --- ipMaskValue ---

func TestIPMaskValue_SetAndString(t *testing.T) {
	var mask net.IPMask
	v := NewIPMask(net.IPv4Mask(255, 255, 255, 0), &mask)

	assert.Equal(t, "ffffff00", v.String())
	assert.Equal(t, "ipMask", v.Type())

	require.NoError(t, v.Set("ffff0000"))
	assert.Equal(t, "ffff0000", v.String())
}

func TestIPMaskValue_SetInvalid(t *testing.T) {
	var mask net.IPMask
	v := NewIPMask(net.IPv4Mask(255, 255, 255, 0), &mask)
	err := v.Set("xyz")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse IP mask")
}

func TestIPMaskValue_SetDottedDecimal(t *testing.T) {
	var mask net.IPMask
	v := NewIPMask(net.IPv4Mask(0, 0, 0, 0), &mask)
	require.NoError(t, v.Set("255.255.0.0"))
	assert.Equal(t, "ffff0000", v.String())
}

// --- ParseIPv4Mask ---

func TestParseIPv4Mask_HexValid(t *testing.T) {
	mask := ParseIPv4Mask("ffffff00")
	require.NotNil(t, mask)
	assert.Equal(t, net.IPv4Mask(255, 255, 255, 0), mask)
}

func TestParseIPv4Mask_DottedDecimal(t *testing.T) {
	mask := ParseIPv4Mask("255.255.255.128")
	require.NotNil(t, mask)
	assert.Equal(t, net.IPv4Mask(255, 255, 255, 128), mask)
}

func TestParseIPv4Mask_InvalidHexLength(t *testing.T) {
	assert.Nil(t, ParseIPv4Mask("fff"))
}

func TestParseIPv4Mask_InvalidHexChars(t *testing.T) {
	assert.Nil(t, ParseIPv4Mask("zzzzzzzz"))
}

// --- ipNetValue ---

func TestIPNetValue_SetAndString(t *testing.T) {
	var ipnet net.IPNet
	_, initial, _ := net.ParseCIDR("10.0.0.0/8")
	v := NewIPNet(*initial, &ipnet)

	assert.Equal(t, "10.0.0.0/8", v.String())
	assert.Equal(t, "ipNet", v.Type())

	require.NoError(t, v.Set("192.168.1.0/24"))
	assert.Equal(t, "192.168.1.0/24", v.String())
}

func TestIPNetValue_SetInvalid(t *testing.T) {
	var ipnet net.IPNet
	_, initial, _ := net.ParseCIDR("10.0.0.0/8")
	v := NewIPNet(*initial, &ipnet)
	err := v.Set("not-a-cidr")
	require.Error(t, err)
}

func TestIPNetValue_SetTrimsWhitespace(t *testing.T) {
	var ipnet net.IPNet
	_, initial, _ := net.ParseCIDR("10.0.0.0/8")
	v := NewIPNet(*initial, &ipnet)
	require.NoError(t, v.Set("  172.16.0.0/12  "))
	assert.Equal(t, "172.16.0.0/12", v.String())
}

// --- ipSliceValue ---

func TestIPSliceValue_SetAndString(t *testing.T) {
	var ips []net.IP
	v := NewIPSlice([]net.IP{net.ParseIP("127.0.0.1")}, &ips)

	assert.Equal(t, "[127.0.0.1]", v.String())
	assert.Equal(t, "ipSlice", v.Type())

	require.NoError(t, v.Set("10.0.0.1,10.0.0.2"))
	assert.Len(t, ips, 2)
	assert.Equal(t, "[10.0.0.1,10.0.0.2]", v.String())
}

func TestIPSliceValue_SetAppends(t *testing.T) {
	var ips []net.IP
	v := NewIPSlice(nil, &ips)

	require.NoError(t, v.Set("1.1.1.1"))
	require.NoError(t, v.Set("2.2.2.2"))
	assert.Len(t, ips, 2, "second Set should append")
}

func TestIPSliceValue_SetInvalidIP(t *testing.T) {
	var ips []net.IP
	v := NewIPSlice(nil, &ips)
	err := v.Set("not-an-ip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid string being converted to IP address")
}

func TestIPSliceValue_SetEmpty(t *testing.T) {
	var ips []net.IP
	v := NewIPSlice(nil, &ips)
	require.NoError(t, v.Set(""))
	assert.Empty(t, ips)
}

func TestIPSliceValue_SetStripsQuotes(t *testing.T) {
	var ips []net.IP
	v := NewIPSlice(nil, &ips)
	require.NoError(t, v.Set(`"10.0.0.1","10.0.0.2"`))
	assert.Len(t, ips, 2)
}

func TestIPSliceValue_Append(t *testing.T) {
	var ips []net.IP
	v := NewIPSlice([]net.IP{net.ParseIP("1.1.1.1")}, &ips)

	require.NoError(t, v.Append("2.2.2.2"))
	assert.Len(t, ips, 2)
}

func TestIPSliceValue_AppendInvalid(t *testing.T) {
	var ips []net.IP
	v := NewIPSlice(nil, &ips)
	err := v.Append("bad")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid string being converted to IP address")
}

func TestIPSliceValue_Replace(t *testing.T) {
	var ips []net.IP
	v := NewIPSlice([]net.IP{net.ParseIP("1.1.1.1")}, &ips)

	require.NoError(t, v.Replace([]string{"3.3.3.3", "4.4.4.4"}))
	assert.Len(t, ips, 2)
	assert.Equal(t, []string{"3.3.3.3", "4.4.4.4"}, v.GetSlice())
}

func TestIPSliceValue_ReplaceInvalid(t *testing.T) {
	var ips []net.IP
	v := NewIPSlice(nil, &ips)
	err := v.Replace([]string{"1.1.1.1", "bad"})
	require.Error(t, err)
}

func TestIPSliceValue_GetSlice(t *testing.T) {
	var ips []net.IP
	v := NewIPSlice([]net.IP{net.ParseIP("8.8.8.8"), net.ParseIP("8.8.4.4")}, &ips)
	assert.Equal(t, []string{"8.8.8.8", "8.8.4.4"}, v.GetSlice())
}

// --- clone helpers ---

func TestCloneIP_Nil(t *testing.T) {
	assert.Nil(t, cloneIP(nil))
}

func TestCloneIPMask_Nil(t *testing.T) {
	assert.Nil(t, cloneIPMask(nil))
}

func TestCloneIPSlice_Nil(t *testing.T) {
	assert.Nil(t, cloneIPSlice(nil))
}

func TestCloneIPNet(t *testing.T) {
	_, original, _ := net.ParseCIDR("10.0.0.0/8")
	cloned := cloneIPNet(*original)
	assert.Equal(t, original.String(), cloned.String())
	// Verify it's a copy
	cloned.IP[0] = 99
	assert.NotEqual(t, original.IP[0], cloned.IP[0])
}
