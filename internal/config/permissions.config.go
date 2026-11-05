package config

const (
	// PermissionProjectRead grants reading the caller's own projects.
	PermissionProjectRead = "narrative:project:read"
	// PermissionProjectWrite grants creating, changing and deleting the caller's own projects.
	PermissionProjectWrite = "narrative:project:write"
	// PermissionGenerationUncapped sends the caller's generations to genai with empty caps, which limit
	// nothing.
	PermissionGenerationUncapped = "narrative:generation:uncapped"
)
