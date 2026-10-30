// Command app installs the Sentry reporter of the default preset, as the servers do, then prints its
// REST read timeout and the credentials of its JSON Keys and genai connections. TestAppPresetDefault
// runs it once per environment, since the preset resolves at package initialization.
package main

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/sentryf"

	"github.com/a-novel/service-narrative-engine/internal/config"
)

func main() {
	cfg := config.AppPresetDefault

	lo.Must0(sentryf.Init(cfg.Sentry))
	fmt.Printf(
		"%v %#v %#v\n",
		cfg.Rest.Timeouts.Read, cfg.Dependencies.ServiceJsonKeysCredentials, cfg.Dependencies.ServiceGenaiCredentials,
	)
}
