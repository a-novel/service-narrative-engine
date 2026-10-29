package config

// Generations configures the model calls of generations.
type Generations struct {
	// Kinds configures each generation kind, keyed by kind.
	Kinds map[string]GenerationKind `json:"kinds" yaml:"kinds"`
}

// GenerationKind configures the model calls of one generation kind. A prompt's size is a token estimate,
// four characters to a token.
type GenerationKind struct {
	// Model is the OpenAI model the kind runs on, such as gpt-6.1-terra.
	Model string `json:"model" yaml:"model"`
	// Reasoning is the reasoning effort the model spends, such as low. Reasoning tokens are billed as output.
	Reasoning string `json:"reasoning" yaml:"reasoning"`
	// MaxOutputTokens caps what one call writes, reasoning included.
	MaxOutputTokens int `json:"maxOutputTokens" yaml:"maxOutputTokens"`

	// Budget is the size a prompt fills up to. The frame, the task and the scenes of the targets always enter,
	// and the context around them shares what they leave.
	Budget int `json:"budget" yaml:"budget"`
	// Limit is the largest prompt the kind sends, when its frame, task and target scenes alone outgrow the
	// budget. A larger prompt is refused.
	Limit int `json:"limit" yaml:"limit"`
}
