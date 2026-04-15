// Fuzz tests for the string-parsing decode hooks in decode.go.
//
// Scope: only the from=string dispatch path is exercised. The from=slice and
// from=map branches in multi-branch hooks (e.g. StringToDurationSliceHookFunc)
// are not fuzzed because Go's native fuzzer only supports primitive seed types.
// Those branches contain their own parsing logic and would benefit from fuzzing
// via a []byte-seeded generator in the future.
package internalhooks

import (
	"log/slog"
	"net"
	"reflect"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

// decodeHookFuncType is the signature all hooks in this file use.
type decodeHookFuncType = func(reflect.Type, reflect.Type, any) (any, error)

// callStringHook invokes a decode hook with from=string, to=targetType, data=input.
// Panics propagate to the fuzz harness; errors are intentionally ignored since
// the goal is panic-freedom, not correctness of error returns.
//
// Only the from=string path is exercised. The slice/map dispatch branches in
// multi-branch hooks (e.g. StringToDurationSliceHookFunc) are not fuzzed here
// because Go's fuzzer only supports primitive seed types.
func callStringHook(hook decodeHookFuncType, targetType reflect.Type, input string) {
	hook(reflect.TypeOf(""), targetType, input) //nolint:errcheck
}

func FuzzStringToIntSlice(f *testing.F) {
	f.Add("1,2,3")
	f.Add("")
	f.Add(",,,")
	f.Add("  1 , 2 , 3  ")
	f.Add("999999999999999999999")
	f.Add("-1,0,1")
	f.Add("[1,2,3]") // error path — this hook doesn't strip brackets

	hook := StringToIntSliceHookFunc(",").(decodeHookFuncType)
	target := reflect.TypeOf([]int(nil))

	f.Fuzz(func(t *testing.T, input string) {
		callStringHook(hook, target, input)
	})
}

// FuzzStringToCSVStringSlice is not needed — the hook's string branch calls
// readAsCSV(raw) with no preprocessing, so FuzzReadAsCSV covers it.

func FuzzStringToBoolSlice(f *testing.F) {
	f.Add("true,false,true")
	f.Add("")
	f.Add("yes,no")
	f.Add("1,0,1")
	f.Add("TRUE,FALSE")
	f.Add("maybe")

	hook := StringToBoolSliceHookFunc().(decodeHookFuncType)
	target := reflect.TypeOf([]bool(nil))

	f.Fuzz(func(t *testing.T, input string) {
		callStringHook(hook, target, input)
	})
}

func FuzzStringToUintSlice(f *testing.F) {
	f.Add("1,2,3")
	f.Add("")
	f.Add("-1")
	f.Add("18446744073709551615") // max uint64
	f.Add("18446744073709551616") // overflow
	f.Add("0,0,0")

	hook := StringToUintSliceHookFunc().(decodeHookFuncType)
	target := reflect.TypeOf([]uint(nil))

	f.Fuzz(func(t *testing.T, input string) {
		callStringHook(hook, target, input)
	})
}

func FuzzStringToDurationSlice(f *testing.F) {
	f.Add("1s,2m,3h")
	f.Add("")
	f.Add("invalid")
	f.Add("1ns,999999h")
	f.Add("-5s")
	f.Add("1s,")

	hook := StringToDurationSliceHookFunc().(decodeHookFuncType)
	target := reflect.TypeOf([]time.Duration(nil))

	f.Fuzz(func(t *testing.T, input string) {
		callStringHook(hook, target, input)
	})
}

func FuzzStringToStringMap(f *testing.F) {
	f.Add("a=1,b=2")
	f.Add("")
	f.Add("=")
	f.Add("key=")
	f.Add("=value")
	f.Add("a=1,a=2")
	f.Add(`"k=1"="v=2"`)

	hook := StringToStringMapHookFunc().(decodeHookFuncType)
	target := reflect.TypeOf(map[string]string(nil))

	f.Fuzz(func(t *testing.T, input string) {
		callStringHook(hook, target, input)
	})
}

func FuzzStringToIntMap(f *testing.F) {
	f.Add("a=1,b=2")
	f.Add("")
	f.Add("key=notanumber")
	f.Add("a=999999999999999999999")
	f.Add("a=-1")

	hook := StringToIntMapHookFunc().(decodeHookFuncType)
	target := reflect.TypeOf(map[string]int(nil))

	f.Fuzz(func(t *testing.T, input string) {
		callStringHook(hook, target, input)
	})
}

func FuzzStringToInt64Map(f *testing.F) {
	f.Add("a=1,b=2")
	f.Add("")
	f.Add("key=notanumber")
	f.Add("a=9223372036854775807")  // max int64
	f.Add("a=-9223372036854775808") // min int64
	f.Add("a=9223372036854775808")  // overflow

	hook := StringToInt64MapHookFunc().(decodeHookFuncType)
	target := reflect.TypeOf(map[string]int64(nil))

	f.Fuzz(func(t *testing.T, input string) {
		callStringHook(hook, target, input)
	})
}

// StringToRawBytesHookFunc is not fuzzed — it is []byte(s) and cannot panic.

func FuzzStringToHexBytes(f *testing.F) {
	f.Add("48656c6c6f")
	f.Add("")
	f.Add("zzzz")
	f.Add("0")
	f.Add("abcdef")
	f.Add("ABCDEF")
	f.Add(" 48 65 ")

	// Can't target structcli.Hex from internal package, so fuzz the
	// underlying decode function directly.
	f.Fuzz(func(t *testing.T, input string) {
		decodeHexBytes(input)
	})
}

func FuzzStringToBase64Bytes(f *testing.F) {
	f.Add("SGVsbG8=")
	f.Add("")
	f.Add("not-base64!!!")
	f.Add("====")
	f.Add("SGVsbG8")  // no padding
	f.Add("SGVsbG8==") // wrong padding

	f.Fuzz(func(t *testing.T, input string) {
		decodeBase64Bytes(input)
	})
}

func FuzzStringToIPMask(f *testing.F) {
	f.Add("255.255.255.0")
	f.Add("")
	f.Add("ffffff00")
	f.Add("not-a-mask")
	f.Add("255.255.255.256")
	f.Add("0.0.0.0")
	f.Add("ffff0000")

	hook := StringToIPMaskHookFunc().(decodeHookFuncType)
	target := reflect.TypeOf(net.IPMask(nil))

	f.Fuzz(func(t *testing.T, input string) {
		callStringHook(hook, target, input)
	})
}

// FuzzStringToIPSlice is not needed — the hook's string branch calls
// parseIPSlice(raw) with no preprocessing, so FuzzParseIPSlice covers it.

func FuzzStringToZapcoreLevel(f *testing.F) {
	f.Add("debug")
	f.Add("info")
	f.Add("warn")
	f.Add("error")
	f.Add("dpanic")
	f.Add("panic")
	f.Add("fatal")
	f.Add("")
	f.Add("INVALID")
	f.Add("DEBUG")

	hook := StringToZapcoreLevelHookFunc().(decodeHookFuncType)
	target := reflect.TypeOf(zapcore.DebugLevel)

	f.Fuzz(func(t *testing.T, input string) {
		callStringHook(hook, target, input)
	})
}

func FuzzStringToSlogLevel(f *testing.F) {
	f.Add("debug")
	f.Add("info")
	f.Add("warn")
	f.Add("error")
	f.Add("")
	f.Add("INVALID")
	f.Add("DEBUG")
	f.Add("INFO+4")

	hook := StringToSlogLevelHookFunc().(decodeHookFuncType)
	target := reflect.TypeOf(slog.LevelInfo)

	f.Fuzz(func(t *testing.T, input string) {
		callStringHook(hook, target, input)
	})
}

func FuzzParseIPv4Mask(f *testing.F) {
	f.Add("255.255.255.0")
	f.Add("ffffff00")
	f.Add("")
	f.Add("not-a-mask")
	f.Add("00000000")
	f.Add("ffffffff")
	f.Add("12345678")
	f.Add("zzzzzzzz")

	f.Fuzz(func(t *testing.T, input string) {
		parseIPv4Mask(input)
	})
}

func FuzzParseIPSlice(f *testing.F) {
	f.Add("192.168.1.1,10.0.0.1")
	f.Add("")
	f.Add("[192.168.1.1]")
	f.Add(`"192.168.1.1","10.0.0.1"`)
	f.Add("::1")
	f.Add(",,,")
	f.Add("not-an-ip")

	f.Fuzz(func(t *testing.T, input string) {
		parseIPSlice(input)
	})
}

func FuzzReadAsCSV(f *testing.F) {
	f.Add("a,b,c")
	f.Add("")
	f.Add(`"a,b",c`)
	f.Add(`"unclosed`)
	f.Add("\x00")
	f.Add("a\nb")

	f.Fuzz(func(t *testing.T, input string) {
		readAsCSV(input)
	})
}

func FuzzNormalizePFlagCollectionString(f *testing.F) {
	f.Add("[a,b,c]")
	f.Add("")
	f.Add("[]")
	f.Add("[")
	f.Add("]")
	f.Add("  [a,b]  ")
	f.Add("no brackets")

	f.Fuzz(func(t *testing.T, input string) {
		normalizePFlagCollectionString(input)
	})
}
