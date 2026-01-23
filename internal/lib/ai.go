package lib

import (
	"context"
	"errors"
	"fmt"

	"github.com/openai/openai-go/v3"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/config/env"
	"github.com/a-novel/service-narrative-engine/internal/config/prompts"
)

var ErrUnknownChatCompletionLang = errors.New("unknown chat completion language")

func NewCompletion(
	ctx context.Context,
	lang string,
	params openai.ChatCompletionNewParams,
) (*openai.ChatCompletion, error) {
	if params.Model == "" {
		params.Model = env.OpenAiModel
	}

	systemPrompt, ok := prompts.System[lang]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownChatCompletionLang, lang)
	}

	params.Messages = append(
		[]openai.ChatCompletionMessageParamUnion{openai.SystemMessage(systemPrompt)},
		params.Messages...,
	)

	return config.OpenAiClient.Chat.Completions.New(ctx, params)
}
