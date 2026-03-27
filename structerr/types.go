// Package structerr configures structured error output for structcli-powered CLIs.
package structerr

// Options configures structured error handling.
type Options struct {
	// FlagName is the JSON Schema flag name used to detect if
	// JSON Schema introspection is configured (for schema-enriched errors).
	// If empty, HandleError still works but without schema enrichment.
	FlagName string
}
