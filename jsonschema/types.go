package jsonschema

// Options configures the --jsonschema flag for command-line applications.
type Options struct {
	FlagName string // Name of the persistent flag (defaults to "jsonschema")

	// SchemaOpts configures the schema renderer used by SetupJSONSchema.
	// It accepts the same functional options as JSONSchema().
	SchemaOpts []Opt
}

// Opt is a functional option for JSONSchema.
type Opt func(*Config)

// Config holds the resolved configuration for JSONSchema.
type Config struct {
	FullTree          bool
	EnumInDescription bool // Keep {val1,val2,...} patterns in description fields
}

// Apply applies all options to a Config and returns it.
func Apply(opts ...Opt) *Config {
	cfg := &Config{}
	for _, o := range opts {
		o(cfg)
	}
	return cfg
}

// WithFullTree makes JSONSchema walk the entire command tree depth-first,
// returning a schema for every available command.
func WithFullTree() Opt {
	return func(c *Config) {
		c.FullTree = true
	}
}

// WithEnumInDescription preserves {val1,val2,...} patterns in description fields.
//
// By default, enum patterns are stripped from descriptions since the values
// are already available in the enum array — this produces cleaner output for
// machine consumers. Use this option to keep the original description text intact.
func WithEnumInDescription() Opt {
	return func(c *Config) {
		c.EnumInDescription = true
	}
}
