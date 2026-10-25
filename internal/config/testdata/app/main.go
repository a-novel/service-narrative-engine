// Command app prints the REST read timeout of the default preset. TestAppPresetDefault runs it
// once per environment, since the preset resolves at package initialization.
package main

import (
	"fmt"

	"github.com/a-novel/service-narrative-engine/internal/config"
)

func main() {
	fmt.Println(config.AppPresetDefault.Rest.Timeouts.Read)
}
