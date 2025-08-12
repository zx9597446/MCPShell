package common

// PromptsConfig holds prompt configuration with system and user prompts
type PromptsConfig struct {
	System []string `yaml:"system,omitempty"` // System prompts
	User   []string `yaml:"user,omitempty"`   // User prompts
}

// GetSystemPrompts returns all system prompts joined with newlines
func (p PromptsConfig) GetSystemPrompts() string {
	if len(p.System) == 0 {
		return ""
	}
	// Join with newlines
	result := ""
	for i, prompt := range p.System {
		if i > 0 {
			result += "\n"
		}
		result += prompt
	}
	return result
}

// GetUserPrompts returns all user prompts joined with newlines
func (p PromptsConfig) GetUserPrompts() string {
	if len(p.User) == 0 {
		return ""
	}
	// Join with newlines
	result := ""
	for i, prompt := range p.User {
		if i > 0 {
			result += "\n"
		}
		result += prompt
	}
	return result
}

// HasSystemPrompts returns true if there are any system prompts configured
func (p PromptsConfig) HasSystemPrompts() bool {
	return len(p.System) > 0
}

// HasUserPrompts returns true if there are any user prompts configured
func (p PromptsConfig) HasUserPrompts() bool {
	return len(p.User) > 0
}
