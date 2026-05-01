// Example: custom type hooks
//
// Demonstrates the three mechanisms for handling custom types in structcli,
// listed here in precedence order (highest first):
//
//  1. FieldHookProvider — per-field Define/Decode on the options struct
//  2. RegisterType[T]  — per-type hooks registered once in init()
//  3. Built-in registry — time.Duration, zapcore.Level, etc.
//
// Run:
//
//	go run . serve --help
//	go run . serve --listen 0.0.0.0:9090 --mode production --timeout 30s
//	go run . serve --mode prod --workers 8
package main

import (
	"fmt"
	"log"
	"net"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/leodido/structcli"
	"github.com/leodido/structcli/values"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ---------------------------------------------------------------------------
// 1. RegisterType[T] — per-type hooks
// ---------------------------------------------------------------------------

// HostPort is a custom type: "host:port" string with validation.
// Registered once in init(), then usable as a plain struct field.
type HostPort struct {
	Host string
	Port int
}

func (hp HostPort) String() string {
	return net.JoinHostPort(hp.Host, strconv.Itoa(hp.Port))
}

func ParseHostPort(s string) (HostPort, error) {
	host, portStr, err := net.SplitHostPort(s)
	if err != nil {
		return HostPort{}, fmt.Errorf("invalid host:port %q: %w", s, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return HostPort{}, fmt.Errorf("invalid port in %q: %w", s, err)
	}

	return HostPort{Host: host, Port: port}, nil
}

// hostPortValue wraps a *HostPort as a pflag.Value.
type hostPortValue struct{ ref *HostPort }

func (v *hostPortValue) String() string { return v.ref.String() }
func (v *hostPortValue) Type() string   { return "host:port" }
func (v *hostPortValue) Set(s string) error {
	hp, err := ParseHostPort(s)
	if err != nil {
		return err
	}
	*v.ref = hp

	return nil
}

func init() {
	// After this, any struct field of type HostPort works automatically.
	// RegisterType panics if Define is nil, Decode is nil, or the type was already registered.
	structcli.RegisterType(structcli.TypeHooks[HostPort]{
		Define: func(name, short, descr string, _ reflect.StructField, fv reflect.Value) (pflag.Value, string) {
			ref := fv.Addr().Interface().(*HostPort)
			*ref = HostPort{Host: "localhost", Port: 8080} // default

			return &hostPortValue{ref: ref}, descr + " (host:port)"
		},
		Decode: func(input any) (any, error) {
			// Safe: structcli always passes the raw flag string here.
			return ParseHostPort(input.(string))
		},
	})
}

// ---------------------------------------------------------------------------
// 2. FieldHookProvider + FieldCompleter — per-field hooks
// ---------------------------------------------------------------------------

// ServerOptions uses FieldHookProvider for the Mode field.
// Mode is a plain string, but we want custom define/decode/completion behavior
// for this specific field without affecting other string fields.
type ServerOptions struct {
	// HostPort is handled by RegisterType — no special tags needed.
	Listen HostPort `flag:"listen" flagdescr:"Bind address" flagenv:"true"`

	// Mode uses FieldHookProvider for custom define/decode.
	// The default tag sets the value for help text and config/env defaults;
	// the Define hook sets the Go value used when no flag is provided.
	Mode string `flag:"mode" flagdescr:"Server mode" default:"development"`

	// Timeout is a time.Duration — handled by the built-in registry.
	Timeout time.Duration `flag:"timeout" flagdescr:"Request timeout" default:"10s" flagenv:"true"`

	// Workers is a standard int — handled natively by pflag.
	Workers int `flag:"workers" flagdescr:"Worker goroutines" default:"4"`
}

var validModes = []string{"development", "staging", "production"}

// FieldHooks provides per-field Define/Decode hooks.
// Map keys are struct field names (e.g., "Mode"), not flag names ("mode").
// Precedence: these override RegisterType and built-in hooks for the named fields.
func (o *ServerOptions) FieldHooks() map[string]structcli.FieldHook {
	return map[string]structcli.FieldHook{
		"Mode": {
			Define: func(name, short, descr string, _ reflect.StructField, fv reflect.Value) (pflag.Value, string) {
				ref := fv.Addr().Interface().(*string)
				*ref = "development"

				return values.NewString(ref), descr + " {" + strings.Join(validModes, ",") + "}"
			},
			Decode: func(input any) (any, error) {
				// Safe: structcli always passes the raw flag string here.
				s := strings.ToLower(strings.TrimSpace(input.(string)))
				// Accept short aliases.
				switch s {
				case "dev":
					return "development", nil
				case "stage", "stg":
					return "staging", nil
				case "prod":
					return "production", nil
				}
				if slices.Contains(validModes, s) {
					return s, nil
				}

				return nil, fmt.Errorf("invalid mode %q (valid: %s)", s, strings.Join(validModes, ", "))
			},
		},
	}
}

// CompletionHooks provides shell completion for specific fields.
// Works for any field that becomes a flag.
func (o *ServerOptions) CompletionHooks() map[string]structcli.CompleteHookFunc {
	return map[string]structcli.CompleteHookFunc{
		"Mode": func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			var matches []string
			for _, m := range validModes {
				if strings.HasPrefix(m, toComplete) {
					matches = append(matches, m)
				}
			}

			return matches, cobra.ShellCompDirectiveNoFileComp
		},
		"Listen": func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{"localhost:8080", "0.0.0.0:8080", "0.0.0.0:443"}, cobra.ShellCompDirectiveNoFileComp
		},
	}
}

func (o *ServerOptions) Attach(c *cobra.Command) error {
	return structcli.Define(c, o)
}

// ---------------------------------------------------------------------------
// CLI
// ---------------------------------------------------------------------------

var opts = &ServerOptions{}

func buildCLI() (root, serve *cobra.Command) {
	root = &cobra.Command{
		Use:   "customtypes",
		Short: "Custom type hooks example",
	}

	serve = &cobra.Command{
		Use:   "serve",
		Short: "Start the server",
		PreRunE: func(c *cobra.Command, _ []string) error {
			return structcli.Unmarshal(c, opts)
		},
		RunE: func(c *cobra.Command, _ []string) error {
			fmt.Fprintf(c.OutOrStdout(), "listen  = %s\n", opts.Listen)
			fmt.Fprintf(c.OutOrStdout(), "mode    = %s\n", opts.Mode)
			fmt.Fprintf(c.OutOrStdout(), "timeout = %s\n", opts.Timeout)
			fmt.Fprintf(c.OutOrStdout(), "workers = %d\n", opts.Workers)

			return nil
		},
	}

	if err := opts.Attach(serve); err != nil {
		log.Fatalln(err)
	}
	root.AddCommand(serve)

	return root, serve
}

func main() {
	log.SetFlags(0)

	root, _ := buildCLI()
	structcli.ExecuteOrExit(root)
}
