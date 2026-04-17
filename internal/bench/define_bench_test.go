// Benchmarks for the Define() and Unmarshal() paths.
//
// Three struct sizes (small, medium, large) × three operations
// (Define-only, Unmarshal-only, full cycle) = 9 benchmarks.
// All report ns/op, B/op, and allocs/op.
package bench_test

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Option structs
// ---------------------------------------------------------------------------

// --- Small: 3 fields, no nesting, no special tags ---

type smallOpts struct {
	Name    string `flag:"name" default:"world"`
	Port    int    `flag:"port" default:"8080"`
	Verbose bool   `flag:"verbose"`
}

func (o *smallOpts) Attach(c *cobra.Command) error { return nil }

// --- Medium: 10 fields, 1-level nesting, mixed tags ---

type mediumDBConfig struct {
	URL      string        `flag:"db-url" default:"postgres://localhost/dev"`
	MaxConns int           `flag:"db-max-conns" default:"10" flagenv:"true"`
	Timeout  time.Duration `flag:"db-timeout" default:"5s"`
}

type mediumOpts struct {
	Host     string         `flag:"host" default:"localhost" flagenv:"true"`
	Port     int            `flag:"port" default:"8080" flagrequired:"true"`
	LogLevel string         `flag:"log-level" flaggroup:"Logging" default:"info"`
	LogFile  string         `flag:"log-file" flaggroup:"Logging" flagenv:"true"`
	Debug    bool           `flag:"debug" flaghidden:"true"`
	Tags     []string       `flag:"tags"`
	Workers  int            `flag:"workers" default:"4"`
	DB       mediumDBConfig `flaggroup:"Database"`
}

func (o *mediumOpts) Attach(c *cobra.Command) error { return nil }

// --- Large: 20+ fields, nesting, all tag types, presets ---

type largeNetConfig struct {
	BindIP   net.IP   `flag:"bind-ip" default:"127.0.0.1" flagenv:"true"`
	Peers    []net.IP `flag:"peers" flagdescr:"Trusted peer IPs"`
	MaxConns int      `flag:"net-max-conns" default:"100"`
}

type largeOpts struct {
	// Primitives
	BoolF    bool    `flag:"bool-f"`
	StringF  string  `flag:"string-f" default:"hello"`
	IntF     int     `flag:"int-f" default:"42"`
	Int8F    int8    `flag:"int8-f"`
	Int16F   int16   `flag:"int16-f"`
	Int32F   int32   `flag:"int32-f"`
	Int64F   int64   `flag:"int64-f"`
	UintF    uint    `flag:"uint-f"`
	Uint8F   uint8   `flag:"uint8-f"`
	Uint16F  uint16  `flag:"uint16-f"`
	Uint32F  uint32  `flag:"uint32-f"`
	Uint64F  uint64  `flag:"uint64-f"`
	Float32F float32 `flag:"float32-f"`
	Float64F float64 `flag:"float64-f" default:"3.14"`

	// Slices
	StringsF []string `flag:"strings-f" flagenv:"true"`
	IntsF    []int    `flag:"ints-f"`

	// Hook-based types
	DurF time.Duration `flag:"dur-f" default:"30s"`
	IPF  net.IP        `flag:"ip-f" default:"0.0.0.0"`

	// Tags: group, required, hidden, env, description
	APIKey  string `flag:"api-key" flagrequired:"true" flagenv:"true" flagdescr:"API authentication key"`
	Secret  string `flag:"secret" flaghidden:"true" flagenv:"true"`
	Region  string `flag:"region" flaggroup:"Deploy" default:"us-east-1" flagenv:"true"`
	Ignored string `flag:"ignored" flagignore:"true"`

	// Presets
	Level int `flag:"level" flagpreset:"verbose=5;quiet=0" default:"1"`

	// Nesting
	Net largeNetConfig `flaggroup:"Network"`
}

func (o *largeOpts) Attach(c *cobra.Command) error { return nil }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newCmd() *cobra.Command {
	return &cobra.Command{Use: "bench"}
}

// setEnv sets an env var and returns a cleanup function.
func setEnv(t testing.TB, key, value string) {
	t.Helper()
	os.Setenv(key, value)
}

// clearEnvs removes the env vars used by benchmarks.
func clearEnvs() {
	for _, k := range []string{
		"HOST", "LOG_FILE", "DB_MAX_CONNS",
		"STRINGS_F", "API_KEY", "SECRET", "REGION", "BIND_IP",
	} {
		os.Unsetenv(k)
	}
}

// ---------------------------------------------------------------------------
// Define-only benchmarks
// ---------------------------------------------------------------------------

func BenchmarkDefine_Small(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		cmd := newCmd()
		opts := &smallOpts{}
		if err := structcli.Define(cmd, opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDefine_Medium(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		cmd := newCmd()
		opts := &mediumOpts{}
		if err := structcli.Define(cmd, opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDefine_Large(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		cmd := newCmd()
		opts := &largeOpts{}
		if err := structcli.Define(cmd, opts); err != nil {
			b.Fatal(err)
		}
	}
}

// ---------------------------------------------------------------------------
// Unmarshal-only benchmarks
//
// Define once in setup, then benchmark Unmarshal with flag+env values.
// Each iteration resets the command to avoid stale state.
// ---------------------------------------------------------------------------

func BenchmarkUnmarshal_Small(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		cmd := newCmd()
		opts := &smallOpts{}
		if err := structcli.Define(cmd, opts); err != nil {
			b.Fatal(err)
		}

		b.StopTimer()
		cmd.Flags().Set("name", "bench")
		cmd.Flags().Set("port", "9090")
		cmd.Flags().Set("verbose", "true")
		b.StartTimer()

		if err := structcli.Unmarshal(cmd, opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshal_Medium(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		cmd := newCmd()
		opts := &mediumOpts{}
		if err := structcli.Define(cmd, opts); err != nil {
			b.Fatal(err)
		}

		b.StopTimer()
		cmd.Flags().Set("host", "0.0.0.0")
		cmd.Flags().Set("port", "3000")
		cmd.Flags().Set("log-level", "debug")
		cmd.Flags().Set("workers", "8")
		cmd.Flags().Set("db-url", "postgres://prod/db")
		cmd.Flags().Set("db-timeout", "10s")
		setEnv(b, "HOST", "envhost")
		setEnv(b, "LOG_FILE", "/var/log/app.log")
		setEnv(b, "DB_MAX_CONNS", "50")
		b.StartTimer()

		if err := structcli.Unmarshal(cmd, opts); err != nil {
			b.Fatal(err)
		}

		b.StopTimer()
		clearEnvs()
		b.StartTimer()
	}
}

func BenchmarkUnmarshal_Large(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		cmd := newCmd()
		opts := &largeOpts{}
		if err := structcli.Define(cmd, opts); err != nil {
			b.Fatal(err)
		}

		b.StopTimer()
		cmd.Flags().Set("string-f", "benchval")
		cmd.Flags().Set("int-f", "99")
		cmd.Flags().Set("float64-f", "2.718")
		cmd.Flags().Set("dur-f", "1m")
		cmd.Flags().Set("ip-f", "10.0.0.1")
		cmd.Flags().Set("api-key", "key123")
		cmd.Flags().Set("region", "eu-west-1")
		cmd.Flags().Set("level", "5")
		cmd.Flags().Set("bind-ip", "192.168.1.1")
		setEnv(b, "STRINGS_F", "a,b,c")
		setEnv(b, "API_KEY", "envkey")
		setEnv(b, "SECRET", "s3cret")
		setEnv(b, "REGION", "ap-south-1")
		setEnv(b, "BIND_IP", "10.0.0.2")
		b.StartTimer()

		if err := structcli.Unmarshal(cmd, opts); err != nil {
			b.Fatal(err)
		}

		b.StopTimer()
		clearEnvs()
		b.StartTimer()
	}
}

// ---------------------------------------------------------------------------
// Full cycle benchmarks: Define → set flags/env → Unmarshal
// ---------------------------------------------------------------------------

func BenchmarkFullCycle_Small(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		cmd := newCmd()
		opts := &smallOpts{}
		if err := structcli.Define(cmd, opts); err != nil {
			b.Fatal(err)
		}
		cmd.Flags().Set("name", "bench")
		cmd.Flags().Set("port", "9090")
		if err := structcli.Unmarshal(cmd, opts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFullCycle_Medium(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		cmd := newCmd()
		opts := &mediumOpts{}
		if err := structcli.Define(cmd, opts); err != nil {
			b.Fatal(err)
		}
		cmd.Flags().Set("host", "0.0.0.0")
		cmd.Flags().Set("port", "3000")
		cmd.Flags().Set("log-level", "debug")
		cmd.Flags().Set("workers", "8")
		cmd.Flags().Set("db-timeout", "10s")
		setEnv(b, "HOST", "envhost")
		setEnv(b, "DB_MAX_CONNS", "50")
		if err := structcli.Unmarshal(cmd, opts); err != nil {
			b.Fatal(err)
		}
		clearEnvs()
	}
}

func BenchmarkFullCycle_Large(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		cmd := newCmd()
		opts := &largeOpts{}
		if err := structcli.Define(cmd, opts); err != nil {
			b.Fatal(err)
		}
		cmd.Flags().Set("string-f", "benchval")
		cmd.Flags().Set("int-f", "99")
		cmd.Flags().Set("float64-f", "2.718")
		cmd.Flags().Set("dur-f", "1m")
		cmd.Flags().Set("ip-f", "10.0.0.1")
		cmd.Flags().Set("api-key", "key123")
		cmd.Flags().Set("region", "eu-west-1")
		cmd.Flags().Set("level", "5")
		cmd.Flags().Set("bind-ip", "192.168.1.1")
		setEnv(b, "STRINGS_F", "a,b,c")
		setEnv(b, "API_KEY", "envkey")
		setEnv(b, "SECRET", "s3cret")
		setEnv(b, "REGION", "ap-south-1")
		setEnv(b, "BIND_IP", "10.0.0.2")
		if err := structcli.Unmarshal(cmd, opts); err != nil {
			b.Fatal(err)
		}
		clearEnvs()
	}
}
