package schemas

import _ "embed"

// ContentDocumentMaxBytes bounds one user-authored content document. It is one tenth of the
// service-genai provider request ceiling, so several project documents can share one generation
// context without letting any individual step dominate it.
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
