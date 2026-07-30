package core

import (
	"encoding/json"
	"fmt"
)

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

	schema, err := loadContentSchema(ideaOutputSchema)
	if err != nil {
		return fmt.Errorf("load Idea schema: %w", err)
	}

	err = schema.validatePartial(value)
	if err != nil {
		return fmt.Errorf("%w: Idea: %w", ErrInvalidRequest, err)
	}

	return nil
}
