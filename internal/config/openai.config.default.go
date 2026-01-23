package config

import (
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"github.com/a-novel/service-narrative-engine/internal/config/env"
)

var OpenAiClient = openai.NewClient(
	option.WithAPIKey(env.OpenAiApiKey),
	option.WithBaseURL(env.OpenAiBaseUrl),
)
