package dao

// Recap condenses a cell or a beat at three levels of detail, as the cells and beats tables store it: a
// document that follows the consolidate output schema.
type Recap struct {
	// Condensed holds about half of the source.
	Condensed string `json:"condensed"`
	// Compact holds about a tenth of the source.
	Compact string `json:"compact"`
	// Essentials holds the source in one sentence.
	Essentials string `json:"essentials"`

	// Names lists the characters, places and objects the source names.
	Names []string `json:"names"`
}
