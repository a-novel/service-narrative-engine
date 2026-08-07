package lib

import (
	"encoding/json"
	"errors"
	"fmt"
)

var errResponsesRequestNil = errors.New("responses request is nil")

// ResponsesJSONSchemaRequest contains the provider-owned fields needed for one
// strict JSON Schema Responses API request.
type ResponsesJSONSchemaRequest struct {
	// Model selects the provider model.
	Model string
	// Reasoning selects the provider reasoning effort.
	Reasoning string
	// MaxOutputTokens bounds the complete provider output, including reasoning tokens.
	MaxOutputTokens int64
	// SafetyIdentifier attributes abuse to one privacy-preserving end-user identity.
	SafetyIdentifier string
	// Instructions contains guidance sent through the provider's instruction channel.
	Instructions string
	// Input contains the provider input document.
	Input string
	// SchemaName identifies the output contract to the provider.
	SchemaName string
	// OutputSchema is the strict JSON Schema enforced on generated text.
	OutputSchema json.RawMessage
}

type responsesRequest struct {
	Model            string             `json:"model"`
	Reasoning        responsesReasoning `json:"reasoning"`
	MaxOutputTokens  int64              `json:"max_output_tokens"` //nolint:tagliatelle // Provider wire name.
	SafetyIdentifier string             `json:"safety_identifier"` //nolint:tagliatelle // Provider wire name.
	Instructions     string             `json:"instructions"`
	Input            string             `json:"input"`
	Text             responsesText      `json:"text"`
}

type responsesReasoning struct {
	Effort string `json:"effort"`
}

type responsesText struct {
	Format responsesTextFormat `json:"format"`
}

type responsesTextFormat struct {
	Type   string          `json:"type"`
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
	Strict bool            `json:"strict"`
}

// EncodeResponsesJSONSchemaRequest serializes a strict structured-output
// request while keeping provider wire fields out of domain packages.
func EncodeResponsesJSONSchemaRequest(request *ResponsesJSONSchemaRequest) (json.RawMessage, error) {
	if request == nil {
		return nil, errResponsesRequestNil
	}

	payload, err := json.Marshal(&responsesRequest{
		Model:            request.Model,
		Reasoning:        responsesReasoning{Effort: request.Reasoning},
		MaxOutputTokens:  request.MaxOutputTokens,
		SafetyIdentifier: request.SafetyIdentifier,
		Instructions:     request.Instructions,
		Input:            request.Input,
		Text: responsesText{Format: responsesTextFormat{
			Type:   "json_schema",
			Name:   request.SchemaName,
			Schema: request.OutputSchema,
			Strict: true,
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("encode Responses request: %w", err)
	}

	return payload, nil
}
