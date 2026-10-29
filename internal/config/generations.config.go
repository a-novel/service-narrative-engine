package config

// Generations configures how the service prepares the prompts of generations.
type Generations struct {
	// Budgets caps the estimated tokens of a prompt, by generation kind. The frame and the task always
	// enter, and the scenes around the targets share what they leave.
	Budgets map[string]int `json:"budgets" yaml:"budgets"`
}
