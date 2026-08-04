package core

import (
	"errors"
	"unicode/utf8"

	"github.com/a-novel/service-narrative-engine/internal/lib"
	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

var errManuscriptMarkRangeInvalid = errors.New("manuscript mark range is invalid")

var manuscriptContentSchema = lib.NewContentSchema(
	schemas.Manuscript,
	schemas.ContentDocumentMaxBytes,
)

// validateManuscriptContent enforces relationships JSON Schema cannot express. It deliberately
// works on the decoded JSON tree so Manuscript remains an opaque document in Go.
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

			start, hasStart := manuscriptJSONInteger(mark["start"])

			end, hasEnd := manuscriptJSONInteger(mark["end"])
			if !hasStart || !hasEnd {
				continue
			}

			if start < 0 || end <= start || end > textLength {
				return errManuscriptMarkRangeInvalid
			}
		}
	}

	return nil
}

func manuscriptJSONInteger(value any) (int, bool) {
	number, ok := value.(float64)
	if !ok {
		return 0, false
	}

	return int(number), true
}
