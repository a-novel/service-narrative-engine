package modules

import (
	"github.com/a-novel/service-narrative-engine/internal/lib"
)

type SystemModule struct {
	ID          string `yaml:"id"`
	Namespace   string `yaml:"namespace"`
	Description string `yaml:"description"`

	Schema *lib.RawSchema `yaml:"schema"`
}
