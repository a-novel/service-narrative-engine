package config

const (
	// PermissionProjectRead grants reading the caller's own projects.
	PermissionProjectRead = "narrative:project:read"
	// PermissionProjectWrite grants creating, changing and deleting the caller's own projects.
	PermissionProjectWrite = "narrative:project:write"
	// PermissionGenerationUncapped runs the caller's generations on genai's admin tier, which no cap
	// limits.
	PermissionGenerationUncapped = "narrative:generation:uncapped"
)
