// Command app prints the REST read timeout of the default preset and the locations of its JSON Keys
// and genai services. TestAppPresetDefault runs it once per environment, since the preset resolves at
// package initialization.
package main

import (
	"fmt"

	"github.com/a-novel/service-narrative-engine/internal/config"
)

func main() {
	cfg := config.AppPresetDefault

	fmt.Printf(
		"%v %+v %+v\n", cfg.Rest.Timeouts.Read, cfg.Dependencies.ServiceJSONKeys, cfg.Dependencies.ServiceGenai,
	)
}
