package handlers

import (
	"github.com/a-novel/service-narrative-engine/internal/core"
)

// Project is the REST representation of a project: its fields, and its logo's checksum.
type Project struct {
	ProjectFields

	// LogoChecksum is the SHA-256 of the logo's bytes in lowercase hex, absent while the project has no logo.
	LogoChecksum *string `json:"logoChecksum,omitempty"`
}

func loadProject(project *core.Project) Project {
	return Project{ProjectFields: loadProjectFields(project), LogoChecksum: project.LogoChecksum}
}
