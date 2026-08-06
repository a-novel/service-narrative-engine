package core

import (
	"encoding/json"
	"fmt"

	"github.com/a-novel/service-narrative-engine/internal/lib"
	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

var ideaContentDefinition = contentDefinition{
	schema: lib.NewContentSchema(schemas.Idea, schemas.ContentDocumentMaxBytes),
}

// validateIdeaContent validates one complete Idea against its static contract.
func validateIdeaContent(seed string, genre string, title string) error {
	content := map[string]string{
		"title": title,
		"genre": genre,
		"seed":  seed,
	}

	value, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("encode idea content: %w", err)
	}

	err = ideaContentDefinition.validate(value)
	if err != nil {
		return fmt.Errorf("idea: %w", err)
	}

	return nil
}
