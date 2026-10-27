// Command app installs the Sentry reporter of the default preset, as the servers do, then prints its
// REST read timeout. TestAppPresetDefault runs it once per environment, since the preset resolves at
// package initialization.
package main

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/sentryf"

	"github.com/a-novel/service-narrative-engine/internal/config"
)

func main() {
	lo.Must0(sentryf.Init(config.AppPresetDefault.Sentry))
	fmt.Println(config.AppPresetDefault.Rest.Timeouts.Read)
}
