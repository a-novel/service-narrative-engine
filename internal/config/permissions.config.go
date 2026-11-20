package config

const (
	// PermissionProjectRead grants reading the caller's own projects.
	PermissionProjectRead = "narrative:project:read"
	// PermissionProjectWrite grants creating, changing and deleting the caller's own projects.
	PermissionProjectWrite = "narrative:project:write"
	// PermissionPluginRead grants reading the plugin registry: its plugins, and the manifest, the modules and the
	// translations of each version.
	PermissionPluginRead = "narrative:plugin:read"
)
