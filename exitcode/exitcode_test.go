package exitcode

import "testing"

func TestConstants(t *testing.T) {
	// Verify exit codes match the documented ranges.
	tests := []struct {
		name     string
		code     int
		wantCat  string
		wantName string
	}{
		// Runtime (0–9)
		{"OK", OK, CategoryOK, "OK"},
		{"Error", Error, CategoryRuntime, "Error"},
		{"PermissionDenied", PermissionDenied, CategoryRuntime, "PermissionDenied"},
		{"Timeout", Timeout, CategoryRuntime, "Timeout"},
		{"Interrupted", Interrupted, CategoryRuntime, "Interrupted"},

		// Input (10–19)
		{"MissingRequiredFlag", MissingRequiredFlag, CategoryInput, "MissingRequiredFlag"},
		{"InvalidFlagValue", InvalidFlagValue, CategoryInput, "InvalidFlagValue"},
		{"UnknownFlag", UnknownFlag, CategoryInput, "UnknownFlag"},
		{"ValidationFailed", ValidationFailed, CategoryInput, "ValidationFailed"},
		{"UnknownCommand", UnknownCommand, CategoryInput, "UnknownCommand"},
		{"InvalidFlagEnum", InvalidFlagEnum, CategoryInput, "InvalidFlagEnum"},

		// Config/env (20–29)
		{"ConfigParseError", ConfigParseError, CategoryConfig, "ConfigParseError"},
		{"ConfigUnknownKey", ConfigUnknownKey, CategoryConfig, "ConfigUnknownKey"},
		{"ConfigInvalidValue", ConfigInvalidValue, CategoryConfig, "ConfigInvalidValue"},
		{"ConfigNotFound", ConfigNotFound, CategoryConfig, "ConfigNotFound"},
		{"EnvInvalidValue", EnvInvalidValue, CategoryConfig, "EnvInvalidValue"},
		{"EnvMissingRequired", EnvMissingRequired, CategoryConfig, "EnvMissingRequired"},
	}

	for _, tt := range tests {
		t.Run(tt.wantName, func(t *testing.T) {
			got := Category(tt.code)
			if got != tt.wantCat {
				t.Errorf("Category(%d) = %q, want %q", tt.code, got, tt.wantCat)
			}
		})
	}
}

func TestCategoryRanges(t *testing.T) {
	// Verify the full range boundaries.
	for code := 1; code <= 9; code++ {
		if cat := Category(code); cat != CategoryRuntime {
			t.Errorf("Category(%d) = %q, want %q", code, cat, CategoryRuntime)
		}
	}
	for code := 10; code <= 19; code++ {
		if cat := Category(code); cat != CategoryInput {
			t.Errorf("Category(%d) = %q, want %q", code, cat, CategoryInput)
		}
	}
	for code := 20; code <= 29; code++ {
		if cat := Category(code); cat != CategoryConfig {
			t.Errorf("Category(%d) = %q, want %q", code, cat, CategoryConfig)
		}
	}
}

func TestCategoryEdgeCases(t *testing.T) {
	// Codes outside defined ranges fall back to runtime.
	if cat := Category(30); cat != CategoryRuntime {
		t.Errorf("Category(30) = %q, want %q", cat, CategoryRuntime)
	}
	if cat := Category(127); cat != CategoryRuntime {
		t.Errorf("Category(127) = %q, want %q", cat, CategoryRuntime)
	}
	if cat := Category(-1); cat != CategoryRuntime {
		t.Errorf("Category(-1) = %q, want %q", cat, CategoryRuntime)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{OK, false},
		{Error, false},
		{PermissionDenied, false},
		{Timeout, false},
		{Interrupted, false},
		{MissingRequiredFlag, true},
		{InvalidFlagValue, true},
		{UnknownFlag, true},
		{ValidationFailed, true},
		{UnknownCommand, true},
		{InvalidFlagEnum, true},
		{ConfigParseError, true},
		{ConfigUnknownKey, true},
		{ConfigInvalidValue, true},
		{ConfigNotFound, true},
		{EnvInvalidValue, true},
		{EnvMissingRequired, true},
		{30, false},  // outside defined ranges
		{127, false}, // outside defined ranges
	}

	for _, tt := range tests {
		got := IsRetryable(tt.code)
		if got != tt.want {
			t.Errorf("IsRetryable(%d) = %v, want %v", tt.code, got, tt.want)
		}
	}
}
