package core

import (
	"errors"
	"unicode/utf8"

	"github.com/a-novel/service-narrative-engine/internal/lib"
	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

var errManuscriptMarkRangeInvalid = errors.New("manuscript mark range is invalid")

var manuscriptContentDefinition = contentDefinition{
	schema:            lib.NewContentSchema(schemas.Manuscript, schemas.ContentDocumentMaxBytes),
	validateSemantics: validateManuscriptContent,
}

// validateManuscriptContent checks mark ranges against their text after JSON
// Schema validation has established every inspected value's shape.
func validateManuscriptContent(instance map[string]any) error {
	blocks, hasBlocks := instance["blocks"].([]any)
	if !hasBlocks {
		return nil
	}

	for _, rawBlock := range blocks {
		block, blockOK := rawBlock.(map[string]any)
		if !blockOK {
			continue
		}

		data, dataOK := block["data"].(map[string]any)
		if !dataOK {
			continue
		}

		text, hasText := data["text"].(string)

		marks, hasMarks := data["marks"].([]any)
		if !hasText || !hasMarks {
			continue
		}

		textLength := utf8.RuneCountInString(text)

		for _, rawMark := range marks {
			mark, markOK := rawMark.(map[string]any)
			if !markOK {
				continue
			}

			start, hasStart := mark["start"].(float64)

			end, hasEnd := mark["end"].(float64)
			if !hasStart || !hasEnd {
				continue
			}

			if start < 0 || end <= start || end > float64(textLength) {
				return errManuscriptMarkRangeInvalid
			}
		}
	}

	return nil
}
