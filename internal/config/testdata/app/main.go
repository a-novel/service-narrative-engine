// Command app prints the REST read timeout of the default preset and the location of its JSON Keys
// service. TestAppPresetDefault runs it once per environment, since the preset resolves at package
// initialization.
package main

import (
	"fmt"

	"github.com/a-novel/service-narrative-engine/v2/internal/config"
)

func main() {
	cfg := config.AppPresetDefault

	fmt.Printf("%v %+v\n", cfg.Rest.Timeouts.Read, cfg.Dependencies.ServiceJSONKeys)
}
