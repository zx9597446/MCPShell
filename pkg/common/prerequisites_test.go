package common

import (
	"runtime"
	"testing"
)

func TestCheckOSMatches(t *testing.T) {
	tests := []struct {
		name       string
		requiredOS string
		expected   bool
	}{
		{
			name:       "Empty OS should match",
			requiredOS: "",
			expected:   true,
		},
		{
			name:       "Current OS should match",
			requiredOS: runtime.GOOS,
			expected:   true,
		},
		{
			name:       "Different OS should not match",
			requiredOS: "non-existent-os",
			expected:   false,
		},
		{
			name:       "Case sensitivity - uppercase current OS should not match",
			requiredOS: "DARWIN",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := CheckOSMatches(tt.requiredOS); result != tt.expected {
				t.Errorf("CheckOSMatches(%q) = %v, expected %v", tt.requiredOS, result, tt.expected)
			}
		})
	}
}

func TestCheckExecutableExists(t *testing.T) {
	// Common executables that should exist on most systems
	commonExecutables := []string{"sh", "bash"}

	// Non-existent executables
	nonExistentExecutables := []string{
		"this-executable-does-not-exist-12345",
		"another-non-existent-executable-67890",
	}

	// Test common executables (at least one should exist)
	foundCommon := false
	for _, exe := range commonExecutables {
		if CheckExecutableExists(exe) {
			foundCommon = true
			break
		}
	}

	if !foundCommon {
		t.Errorf("None of the common executables %v were found, at least one should exist", commonExecutables)
	}

	// Test non-existent executables
	for _, exe := range nonExistentExecutables {
		if CheckExecutableExists(exe) {
			t.Errorf("Non-existent executable %q was reported as existing", exe)
		}
	}
}
