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
	// budget.
	Limit int `json:"limit" yaml:"limit"`
	// LongLimit raises Limit for a call sent as long context. Zero turns long context off. OpenAI bills input
	// past 272K tokens at twice the rate.
	LongLimit int `json:"longLimit" yaml:"longLimit"`
	// Split runs a generation that fits no single call as several. suggest gives each target a call of its
	// own, and each span of a range whose rewrite passes half the output cap. check checks consecutive
	// windows of scenes whole. consolidate never splits.
	Split bool `json:"split" yaml:"split"`
}
