package config

import (
	_ "embed"

	"github.com/goccy/go-yaml"

	"github.com/a-novel-kit/golib/config"
)

//go:embed generations.config.yaml
var defaultGenerationsFile []byte

// GenerationsConfigDefault is the built-in configuration of generation prompts.
var GenerationsConfigDefault = config.MustUnmarshal[Generations](yaml.Unmarshal, defaultGenerationsFile)
