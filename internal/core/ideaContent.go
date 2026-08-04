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

// validateIdeaContent validates the non-empty fields supplied for one partial
// Idea without inventing omitted values.
func validateIdeaContent(seed string, genre string, title string) error {
	content := make(map[string]string)
	if title != "" {
		content["title"] = title
	}

	if genre != "" {
		content["genre"] = genre
	}

	if seed != "" {
		content["seed"] = seed
	}

	value, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("encode Idea content: %w", err)
	}

	err = ideaContentDefinition.validatePartial(value)
	if err != nil {
		return fmt.Errorf("Idea: %w", err)
	}

	return nil
}
