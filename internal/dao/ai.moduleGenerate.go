package dao

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/openai/openai-go/v3"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

//go:embed ai.moduleGenerate.prompt
var moduleGeneratePrompt string

var moduleGeneratePromptTemplate = template.Must(template.New("").Parse(moduleGeneratePrompt))

type ModuleGenerateRequest struct {
	Module *Module

	Lang string

	Context   any
	Prefilled map[string]any
}

type ModuleGenerate struct{}

func NewModuleGenerate() *ModuleGenerate {
	return new(ModuleGenerate)
}

func (repository *ModuleGenerate) Exec(ctx context.Context, request *ModuleGenerateRequest) (map[string]any, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ModuleGenerate")
	defer span.End()

	span.SetAttributes(
		attribute.String("request.module.id", request.Module.ID),
		attribute.String("request.module.namespace", request.Module.Namespace),
		attribute.String("request.module.version", request.Module.Version),
		attribute.String("request.lang", request.Lang),
	)

	strContext, err := json.Marshal(request.Context)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("marshal context: %w", err))
	}

	strPrefilled, err := json.Marshal(request.Prefilled)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("marshal prefilled: %w", err))
	}

	userPrompt := new(strings.Builder)

	err = moduleGeneratePromptTemplate.Execute(userPrompt, map[string]any{
		"context":   string(strContext),
		"prefilled": lo.Ternary[any](len(request.Prefilled) == 0, nil, string(strPrefilled)),
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute prompt template: %w", err))
	}

	res, err := lib.NewCompletion(ctx, request.Lang, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(userPrompt.String()),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:        request.Module.ID,
					Schema:      request.Module.Schema,
					Description: openai.String(request.Module.Description),
					Strict:      openai.Bool(true),
				},
			},
		},
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("generate completion: %w", err))
	}

	var result map[string]any

	err = json.Unmarshal([]byte(res.Choices[0].Message.Content), &result)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("unmarshal completion result: %w", err))
	}

	return result, nil
}
