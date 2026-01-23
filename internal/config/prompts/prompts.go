package prompts

import (
	_ "embed"

	"github.com/a-novel/service-narrative-engine/internal/config"
)

//go:embed system.en.prompt
var systemEn string

//go:embed system.fr.prompt
var systemFr string

var System = map[string]string{
	config.LangEN: systemEn,
	config.LangFR: systemFr,
}
