package core

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

// The types below are the OpenAI Responses request a call sends, which genai forwards as is. Struct fields
// marshal in a fixed order, so a plan that holds still gives the same bytes, and genai replays the call.
//
//nolint:tagliatelle // OpenAI's own field names.
type (
	responsesRequest struct {
		Model           string             `json:"model"`
		Reasoning       responsesReasoning `json:"reasoning"`
		MaxOutputTokens int                `json:"max_output_tokens"`
		// Store keeps the call out of the provider's 30-day copy.
		Store          bool   `json:"store"`
		PromptCacheKey string `json:"prompt_cache_key"`
		// PromptCacheOptions caches only at the breakpoints the input marks, so the task, which changes with
		// every call, is never written to the cache.
		PromptCacheOptions responsesCacheOptions `json:"prompt_cache_options"`
		Input              []*responsesMessage   `json:"input"`
		Text               responsesText         `json:"text"`
	}

	responsesReasoning struct {
		Effort string `json:"effort"`
	}

	responsesCacheOptions struct {
		Mode string `json:"mode"`
	}

	responsesMessage struct {
		Role    string              `json:"role"`
		Content []*responsesContent `json:"content"`
	}

	responsesContent struct {
		Type       string                 `json:"type"`
		Text       string                 `json:"text"`
		Breakpoint *responsesCacheOptions `json:"prompt_cache_breakpoint,omitempty"`
	}

	responsesText struct {
		Format responsesFormat `json:"format"`
	}

	responsesFormat struct {
		Type   string          `json:"type"`
		Name   string          `json:"name"`
		Schema json.RawMessage `json:"schema"`
		Strict bool            `json:"strict"`
	}
)

// responsesRoleUser is the role of the messages that carry the author's project.
const responsesRoleUser = "user"

// explicitCache marks a cache breakpoint the call places itself.
var explicitCache = &responsesCacheOptions{Mode: "explicit"}

// inputMessage returns an input message of a role, holding text and ending at a cache breakpoint when one
// is set.
func inputMessage(role, text string, breakpoint *responsesCacheOptions) *responsesMessage {
	return &responsesMessage{
		Role: role, Content: []*responsesContent{{Type: "input_text", Text: text, Breakpoint: breakpoint}},
	}
}

// newPayload returns the Responses request of one call of a generation of kind. The developer message and
// the frame come first, with a cache breakpoint after the frame, so the next calls of the project read
// them from cache. The neighborhood and the task follow, each as a message of its own. CacheKey groups the
// project's calls for the provider's cache.
func newPayload(kind, cacheKey string, prompt *Prompt) ([]byte, error) {
	request := &responsesRequest{
		Model:              prompt.Model,
		Reasoning:          responsesReasoning{Effort: prompt.Reasoning},
		MaxOutputTokens:    prompt.MaxOutputTokens,
		PromptCacheKey:     cacheKey,
		PromptCacheOptions: *explicitCache,
		Input: []*responsesMessage{
			inputMessage("developer", prompt.Developer, nil),
			inputMessage(responsesRoleUser, prompt.Frame, explicitCache),
			inputMessage(responsesRoleUser, prompt.Neighborhood, nil),
			inputMessage(responsesRoleUser, prompt.Task, nil),
		},
		Text: responsesText{Format: responsesFormat{
			Type: "json_schema", Name: kind, Schema: prompt.Schema, Strict: true,
		}},
	}

	// The prompts hold XML tags, which an HTML-escaping encoder would spell out six characters each.
	var payload bytes.Buffer

	encoder := json.NewEncoder(&payload)
	encoder.SetEscapeHTML(false)

	err := encoder.Encode(request)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	return bytes.TrimSuffix(payload.Bytes(), []byte("\n")), nil
}

// openaiResponse is the part of an OpenAI Responses object a read checks: whether the model finished,
// what it wrote, and the output format the call asked for, named after the kind of the generation.
//
//nolint:tagliatelle // OpenAI's own field names.
type openaiResponse struct {
	Model             string `json:"model"`
	Status            string `json:"status"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type    string `json:"type"`
			Text    string `json:"text"`
			Refusal string `json:"refusal"`
		} `json:"content"`
	} `json:"output"`
	Text struct {
		Format struct {
			Name string `json:"name"`
		} `json:"format"`
	} `json:"text"`
}

// providerOutput is the valid output of a provider's response, with the model that wrote it.
type providerOutput struct {
	text  json.RawMessage
	model string
}

// readOutput returns the output a provider's response holds once it passed the full output schema of its
// kind. It returns instead the reason the response holds no valid output: cut short, refused or malformed.
func readOutput(raw []byte) (*providerOutput, string) {
	response := &openaiResponse{}

	err := json.Unmarshal(raw, response)
	if err != nil {
		return nil, "malformed response: " + err.Error()
	}

	if response.IncompleteDetails != nil {
		return nil, "incomplete output: " + response.IncompleteDetails.Reason
	}

	if response.Status != "completed" {
		return nil, "response " + response.Status
	}

	var text json.RawMessage

	for _, item := range response.Output {
		for _, content := range item.Content {
			switch content.Type {
			case "refusal":
				return nil, "refused: " + content.Refusal
			case "output_text":
				text = append(text, content.Text...)
			}
		}
	}

	schema, err := schemas.Output(response.Text.Format.Name)
	if err != nil {
		return nil, "unknown output format: " + response.Text.Format.Name
	}

	err = schema.Validate(text)
	if err != nil {
		return nil, "malformed output: " + err.Error()
	}

	return &providerOutput{text: text, model: response.Model}, ""
}
