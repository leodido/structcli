package internalscope

import (
	"context"
	"sync"

	"maps"

	"github.com/go-viper/mapstructure/v2"
	structclierrors "github.com/leodido/structcli/errors"
	"github.com/spf13/cobra"
	spf13viper "github.com/spf13/viper"
)

// structcliContextKey is used to store scope in command context
type structcliContextKey struct{}

// Scope holds per-command state for structcli
type Scope struct {
	v                 *spf13viper.Viper
	configV           *spf13viper.Viper
	boundEnvs         map[string]bool
	customDecodeHooks map[string]mapstructure.DecodeHookFunc
	definedFlags      map[string]string
	boundOptions      []any // ordered list of options registered via Bind, unmarshalled in FIFO order
	mu                sync.RWMutex
}

// Get retrieves or creates a scope for the given command
func Get(c *cobra.Command) *Scope {
	ctx := c.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	// Check if command already has scope
	if s, ok := ctx.Value(structcliContextKey{}).(*Scope); ok {
		return s
	}

	// Create new scope (ensures isolation even with context inheritance)
	s := &Scope{
		v:                 spf13viper.New(),
		configV:           spf13viper.New(),
		boundEnvs:         make(map[string]bool),
		customDecodeHooks: make(map[string]mapstructure.DecodeHookFunc),
		definedFlags:      make(map[string]string),
	}

	// Attach to command context
	newCtx := context.WithValue(ctx, structcliContextKey{}, s)
	c.SetContext(newCtx)

	return s
}

// Viper returns the viper instance for the command
func (s *Scope) Viper() *spf13viper.Viper {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.v
}

// ConfigViper returns the dedicated viper instance used for config-file loading.
func (s *Scope) ConfigViper() *spf13viper.Viper {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.configV
}

// ResetConfigViper clears config-file state for the scope while preserving other scope data.
func (s *Scope) ResetConfigViper() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configV = spf13viper.New()
}

// IsEnvBound checks if an environment variable is already bound for this command
func (s *Scope) IsEnvBound(flagName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.boundEnvs[flagName]
}

// SetBound marks an environment variable as bound for this command
func (s *Scope) SetBound(flagName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.boundEnvs[flagName] = true
}

// ClearBoundEnv removes the bound marker for a single flag so that
// BindEnv will re-bind it on the next call. Used by env annotation
// patching when WithAppName retroactively updates env var names.
func (s *Scope) ClearBoundEnv(flagName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.boundEnvs, flagName)
}

// GetBoundEnvs is for testing purposes only
func (s *Scope) GetBoundEnvs() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]bool, len(s.boundEnvs))
	maps.Copy(result, s.boundEnvs)

	return result
}

func (s *Scope) SetCustomDecodeHook(hookName string, hookFunc mapstructure.DecodeHookFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.customDecodeHooks[hookName] = hookFunc
}

func (s *Scope) GetCustomDecodeHook(hookName string) (hookFunc mapstructure.DecodeHookFunc, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hookFunc, ok = s.customDecodeHooks[hookName]

	return
}

// AddDefinedFlag adds a flag to the set of defined flags for this scope, returning an error if it's a duplicate.
func (s *Scope) AddDefinedFlag(name, fieldPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existingPath, ok := s.definedFlags[name]; ok {
		return structclierrors.NewDuplicateFlagError(name, fieldPath, existingPath)
	}
	s.definedFlags[name] = fieldPath

	return nil
}

// AddBoundOptions appends an options struct to the ordered list of bound options for this command.
// Unmarshal order matches call order (FIFO).
func (s *Scope) AddBoundOptions(opts any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.boundOptions = append(s.boundOptions, opts)
}

// BoundOptions returns a copy of the ordered list of bound options for this command.
func (s *Scope) BoundOptions() []any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.boundOptions) == 0 {
		return nil
	}

	result := make([]any, len(s.boundOptions))
	copy(result, s.boundOptions)

	return result
}
