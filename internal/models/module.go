package models

type ModuleUiParams = map[string]any

type ModuleUi struct {
	// The ID of the ui component to render.d
	Component string `json:"component"`
	// Parameters of the ui component.
	Params ModuleUiParams `json:"params"`
	// Optional target field where the ui editable content will be written. Leave empty for passing through.
	Target string `json:"target"`
}
