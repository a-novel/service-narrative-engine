package schemas

import _ "embed"

// ContentDocumentMaxBytes bounds one user-authored content document.
const ContentDocumentMaxBytes = 368_800

var (
	// Idea defines the static contract for persisted and generated Idea content.
	//
	//go:embed idea.schema.json
	Idea []byte

	// Manuscript defines the static contract for persisted and generated Manuscript content.
	//
	//go:embed manuscript.schema.json
	Manuscript []byte
)
