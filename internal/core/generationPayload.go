package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

const (
	generationIdempotencyVersion     = 3
	generationSafetyIdentifierDomain = "service-narrative-engine/generation-safety/v1:"
)

// generationInputDocument keeps caller input separate from caller-supplied context
// inside the provider's untrusted input channel.
type generationInputDocument struct {
	Input   json.RawMessage `json:"input"`
	Context json.RawMessage `json:"context"`
}

// buildGenerationPayload keeps client data in the provider's untrusted input channel.
func buildGenerationPayload(
	ownerID uuid.UUID,
	instructions string,
	input json.RawMessage,
	contextValue json.RawMessage,
	outputSchema json.RawMessage,
) (json.RawMessage, error) {
	inputDocument, err := json.Marshal(&generationInputDocument{
		Input:   input,
		Context: contextValue,
	})
	if err != nil {
		return nil, fmt.Errorf("encode generation input: %w", err)
	}

	payload, err := lib.EncodeResponsesJSONSchemaRequest(&lib.ResponsesJSONSchemaRequest{
		Model:            GenerationModelDefault,
		Reasoning:        GenerationReasoningEffortDefault,
		MaxOutputTokens:  GenerationMaxOutputTokensDefault,
		SafetyIdentifier: deriveGenerationSafetyIdentifier(ownerID),
		Instructions:     instructions,
		Input:            string(inputDocument),
		SchemaName:       "project_content_output",
		OutputSchema:     outputSchema,
	})
	if err != nil {
		return nil, fmt.Errorf("encode Responses request: %w", err)
	}

	return payload, nil
}

// deriveGenerationSafetyIdentifier gives the provider a stable abuse identity without disclosing
// the Project owner's internal identifier or allowing the client to choose the attribution key.
func deriveGenerationSafetyIdentifier(ownerID uuid.UUID) string {
	material := make([]byte, 0, len(generationSafetyIdentifierDomain)+len(ownerID))
	material = append(material, generationSafetyIdentifierDomain...)
	material = append(material, ownerID[:]...)
	digest := sha256.Sum256(material)

	return hex.EncodeToString(digest[:])
}

// deriveGenerationIdempotencyKey isolates a caller retry identity to one Project.
func deriveGenerationIdempotencyKey(
	retryIdentity string,
	projectID uuid.UUID,
) (string, error) {
	material, err := json.Marshal(struct {
		Version       int       `json:"version"`
		RetryIdentity string    `json:"retryIdentity"`
		ProjectID     uuid.UUID `json:"projectID"`
	}{
		Version:       generationIdempotencyVersion,
		RetryIdentity: retryIdentity,
		ProjectID:     projectID,
	})
	if err != nil {
		return "", fmt.Errorf("encode idempotency material: %w", err)
	}

	digest := sha256.Sum256(material)

	return hex.EncodeToString(digest[:]), nil
}
