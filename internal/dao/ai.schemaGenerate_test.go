package dao_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/lib"
)

// detectLanguage uses AI to detect the language of the given text.
// Returns "en", "fr", or an error if detection fails.
func detectLanguage(ctx context.Context, text string) (string, error) {
	response, err := config.OpenAiClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(
				"You are a language detection system. Analyze the text and respond with ONLY the ISO 639-1 language " +
					"code (e.g., 'en' for English, 'fr' for French). Do not include any other text in your response."),
			openai.UserMessage("Detect the language of this text: " + text),
		},
	})
	if err != nil {
		return "", fmt.Errorf("language detection API call failed: %w", err)
	}

	detectedLang := strings.ToLower(strings.TrimSpace(response.Choices[0].Message.Content))

	return detectedLang, nil
}

// List alternatives values, more relevant giving what you did

func TestSchemaGenerate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping AI-based tests in short mode")

		return
	}

	generateTestSchema := lib.RawSchema{
		Type: lib.NewSchemaType("object"),
		// Marshals to "additionalProperties": false, which OpenAI requires.
		AdditionalProperties: lib.AdditionalPropertiesBool(false),
		Properties: map[string]lib.RawSchema{
			"title": {
				Type:        lib.NewSchemaType("string"),
				Description: "A creative title for the story",
				MaxLength:   lo.ToPtr(128),
			},
			"premise": {
				Type:        lib.NewSchemaType("string"),
				Description: "A short premise for the story",
				MaxLength:   lo.ToPtr(1024),
			},
			"genre": {
				Type:        lib.NewSchemaType("string"),
				Description: "The story genre",
			},
		},
		Required: []string{"title", "premise", "genre"},
	}

	testModule := dao.Module{
		ID:          "test-idea",
		Namespace:   "test",
		Version:     "1.0.0",
		Description: "A test module that generates a story idea.",
		Schema:      &generateTestSchema,
	}

	testCases := []struct {
		name string

		request *dao.SchemaGenerateRequest

		// validateResult is a custom validation function since AI output is non-deterministic.
		validateResult func(t *testing.T, result map[string]any)
	}{
		{
			name: "Success/BasicGeneration",

			request: &dao.SchemaGenerateRequest{
				Module: &testModule,
				Lang:   "en",
				Context: map[string]any{
					"theme": "space exploration",
					"tone":  "optimistic",
				},
				Prefilled: nil,
			},

			validateResult: func(t *testing.T, result map[string]any) {
				t.Helper()

				// Verify result is not nil
				require.NotNil(t, result)

				// Verify all required fields exist
				require.Contains(t, result, "title")
				require.Contains(t, result, "premise")
				require.Contains(t, result, "genre")

				// Verify field types
				title, ok := result["title"].(string)
				require.True(t, ok, "title should be a string")
				require.NotEmpty(t, title, "title should not be empty")

				premise, ok := result["premise"].(string)
				require.True(t, ok, "premise should be a string")
				require.NotEmpty(t, premise, "premise should not be empty")

				genre, ok := result["genre"].(string)
				require.True(t, ok, "genre should be a string")
				require.NotEmpty(t, genre, "genre should not be empty")

				// Verify no extra fields (strict schema)
				require.Len(t, result, 3, "result should only contain the 3 required fields")
			},
		},
		{
			name: "Success/WithPrefilled",

			request: &dao.SchemaGenerateRequest{
				Module: &testModule,
				Lang:   "en",
				Context: map[string]any{
					"setting": "underwater city",
				},
				Prefilled: map[string]any{
					"title": "The Deep",
					"genre": "science fiction",
				},
			},

			validateResult: func(t *testing.T, result map[string]any) {
				t.Helper()

				require.NotNil(t, result)

				// Verify prefilled values are preserved
				require.Equal(t, "The Deep", result["title"])
				require.Equal(t, "science fiction", result["genre"])

				// Verify AI filled the missing field
				premise, ok := result["premise"].(string)
				require.True(t, ok, "premise should be a string")
				require.NotEmpty(t, premise, "premise should not be empty")

				// Verify the premise is contextually relevant
				// (We can't test exact content, but we can verify it exists and has reasonable length)
				require.Greater(t, len(premise), 10, "premise should be a meaningful description")
			},
		},
		{
			name: "Success/WithInstruction",

			request: &dao.SchemaGenerateRequest{
				Module: &testModule,
				Lang:   "en",
				Context: map[string]any{
					"theme": "medieval fantasy",
				},
				Instruction: "The genre MUST be exactly 'fantasy'. The title MUST contain the word 'Dragon'.",
			},

			validateResult: func(t *testing.T, result map[string]any) {
				t.Helper()

				require.NotNil(t, result)

				// Verify all required fields exist
				require.Contains(t, result, "title")
				require.Contains(t, result, "premise")
				require.Contains(t, result, "genre")

				// Verify field types
				title, ok := result["title"].(string)
				require.True(t, ok, "title should be a string")
				require.NotEmpty(t, title, "title should not be empty")

				// Verify instruction was followed: title must contain "Dragon"
				require.Contains(t, strings.ToLower(title), "dragon",
					"title should contain 'Dragon' as instructed, got: %s", title)

				// Verify instruction was followed: genre must be "fantasy"
				genre, ok := result["genre"].(string)
				require.True(t, ok, "genre should be a string")
				require.Equal(t, "fantasy", strings.ToLower(genre),
					"genre should be 'fantasy' as instructed, got: %s", genre)

				premise, ok := result["premise"].(string)
				require.True(t, ok, "premise should be a string")
				require.NotEmpty(t, premise, "premise should not be empty")
			},
		},
	}

	repository := dao.NewSchemaGenerate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()

			result, err := repository.Exec(ctx, testCase.request)

			require.NoError(t, err)
			testCase.validateResult(t, result)
		})
	}
}

func TestSchemaGenerate_Language(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping AI-based tests in short mode")

		return
	}

	// Test schema with enums, filenames, and translatable text fields
	languageTestSchema := lib.RawSchema{
		Type: lib.NewSchemaType("object"),
		// Marshals to "additionalProperties": false, which OpenAI requires.
		AdditionalProperties: lib.AdditionalPropertiesBool(false),
		Properties: map[string]lib.RawSchema{
			"title": {
				Type:        lib.NewSchemaType("string"),
				Description: "A creative title for the story",
			},
			"description": {
				Type:        lib.NewSchemaType("string"),
				Description: "A detailed description of the story concept",
			},
			"medium": {
				Type:        lib.NewSchemaType("string"),
				Enum:        []any{"FILM", "SERIES", "NOVEL", "GAME", "VISUAL NOVEL", "COMIC"},
				Description: "The medium for this story (must be one of the enum values)",
			},
			"ageRating": {
				Type:        lib.NewSchemaType("string"),
				Enum:        []any{"G", "PG", "PG-13", "R", "NC-17"},
				Description: "The age rating for this story (must be one of the enum values)",
			},
			"filename": {
				Type:        lib.NewSchemaType("string"),
				Description: "A suggested filename for this project (e.g., 'my_story_project.txt')",
			},
			"mood": {
				Type:        lib.NewSchemaType("string"),
				Description: "The mood or atmosphere of the story",
				Enum: []any{
					"dark",
					"light",
					"serious",
					"funny",
					"romantic",
					"action-packed",
					"mysterious",
					"epic",
					"sci-fi",
					"fantasy",
				},
			},
		},
		Required: []string{"title", "description", "medium", "ageRating", "filename", "mood"},
	}

	testModule := dao.Module{
		ID:          "language-test",
		Namespace:   "test",
		Version:     "1.0.0",
		Description: "A test module that generates a story idea.",
		Schema:      &languageTestSchema,
	}

	// Language-specific context for testing
	contextByLang := map[string]map[string]any{
		config.LangEN: {
			"theme": "space exploration",
			"tone":  "optimistic and adventurous",
		},
		config.LangFR: {
			"theme": "exploration spatiale",
			"tone":  "optimiste et aventureux",
		},
	}

	// Shared validation function for all languages
	validateResult := func(t *testing.T, result map[string]any, expectedLang string) {
		t.Helper()

		ctx := context.Background()

		// Verify all fields exist
		require.Contains(t, result, "title")
		require.Contains(t, result, "description")
		require.Contains(t, result, "medium")
		require.Contains(t, result, "ageRating")
		require.Contains(t, result, "filename")
		require.Contains(t, result, "mood")

		// Verify translatable text fields are in the expected language
		title, ok := result["title"].(string)
		require.True(t, ok, "title should be a string")
		require.NotEmpty(t, title, "title should not be empty")

		description, ok := result["description"].(string)
		require.True(t, ok, "description should be a string")
		require.NotEmpty(t, description, "description should not be empty")

		mood, ok := result["mood"].(string)
		require.True(t, ok, "mood should be a string")
		require.NotEmpty(t, mood, "mood should not be empty")

		// Use AI to detect language of translatable fields
		combinedText := fmt.Sprintf("%s. %s. %s", title, description, mood)
		detectedLang, err := detectLanguage(ctx, combinedText)
		require.NoError(t, err, "language detection should not fail")
		require.Equal(t, expectedLang, detectedLang,
			"Expected language %s but detected %s in text: %s",
			expectedLang, detectedLang, combinedText)

		// Verify enums are NOT translated (remain in original form)
		medium, ok := result["medium"].(string)
		require.True(t, ok, "medium should be a string")
		require.Contains(t, []string{"FILM", "SERIES", "NOVEL", "GAME", "VISUAL NOVEL", "COMIC"}, medium,
			"medium should be one of the original enum values, not translated")

		ageRating, ok := result["ageRating"].(string)
		require.True(t, ok, "ageRating should be a string")
		require.Contains(t, []string{"G", "PG", "PG-13", "R", "16+", "18+"}, ageRating,
			"ageRating should be one of the original enum values, not translated")

		// Verify filename is NOT translated (should remain in ASCII format)
		filename, ok := result["filename"].(string)
		require.True(t, ok, "filename should be a string")
		require.NotEmpty(t, filename, "filename should not be empty")
		// Filenames should typically not contain non-ASCII characters
		require.Regexp(t, `^[a-zA-Z0-9_\-\.]+$`, filename,
			"filename should remain in ASCII format without translation: %s", filename)
	}

	repository := dao.NewSchemaGenerate()

	// Automatically test all known languages
	for _, lang := range config.KnownLangs {
		testName := "Language_" + strings.ToUpper(lang)

		t.Run(testName, func(t *testing.T) {
			ctx := context.Background()

			// Get language-specific context, or use default English context
			testContext := contextByLang[lang]
			if testContext == nil {
				testContext = contextByLang[config.LangEN]
			}

			result, err := repository.Exec(ctx, &dao.SchemaGenerateRequest{
				Module:    &testModule,
				Lang:      lang,
				Context:   testContext,
				Prefilled: nil,
			})

			require.NoError(t, err)
			require.NotNil(t, result)

			// Log the result for debugging
			resultJSON, _ := json.MarshalIndent(result, "", "  ")
			t.Logf("Generated result for lang=%s:\n%s", lang, string(resultJSON))

			// Run validation
			validateResult(t, result, lang)
		})
	}
}
