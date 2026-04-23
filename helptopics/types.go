package helptopics

// Options configures the help topic commands registered by SetupHelpTopics.
type Options struct {
	// ReferenceSection moves help topic commands from "Available Commands:"
	// into a dedicated "Reference:" section in --help output. When false
	// (default), help topic commands appear as regular subcommands, matching
	// cobra's default behavior.
	ReferenceSection bool
}
