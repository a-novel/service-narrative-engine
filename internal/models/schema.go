package models

type SchemaSource string

const (
	SchemaSourceUser     SchemaSource = "USER"
	SchemaSourceAI       SchemaSource = "AI"
	SchemaSourceFork     SchemaSource = "FORK"
	SchemaSourceExternal SchemaSource = "EXTERNAL"
)

func (schemaSource SchemaSource) String() string {
	return string(schemaSource)
}

var KnownSchemaSources = []SchemaSource{
	SchemaSourceUser,
	SchemaSourceAI,
	SchemaSourceFork,
	SchemaSourceExternal,
}
