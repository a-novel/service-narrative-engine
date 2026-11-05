package config

import "time"

// Caps maps roles to the caps genai counts their users' generations against. genai sets no limit of its
// own, so these caps alone bound what a user spends.
type Caps struct {
	// Roles maps a role to the caps of its users. A caller takes the caps of their highest-priority role
	// named here, held or inherited.
	Roles map[string]RoleCaps `json:"roles" yaml:"roles"`
}

// RoleCaps limits the generations of one role's users.
type RoleCaps struct {
	// Concurrency caps the calls one user may have in flight at once. Nil sets no limit.
	Concurrency *int32 `json:"concurrency" yaml:"concurrency"`
	// Windows cap the tokens one user's calls consume over rolling windows, keyed by how far back each
	// reaches, such as 5h.
	Windows map[time.Duration]CapsWindow `json:"windows" yaml:"windows"`
}

// CapsWindow caps the tokens one user's calls consume over a rolling window.
type CapsWindow struct {
	// InputTokens caps the input tokens, cached ones included. Nil sets no limit.
	InputTokens *int64 `json:"inputTokens" yaml:"inputTokens"`
	// OutputTokens caps the output tokens, reasoning ones included. Nil sets no limit.
	OutputTokens *int64 `json:"outputTokens" yaml:"outputTokens"`
}
